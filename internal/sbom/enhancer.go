package sbom

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/platform"
	"github.com/go-nv/goenv/internal/utils"
)

// Enhancer adds Go-aware metadata to CycloneDX SBOMs
type Enhancer struct {
	config    *config.Config
	manager   *manager.Manager
	toolchain *Toolchain
}

// NewEnhancer creates a new SBOM enhancer
func NewEnhancer(cfg *config.Config, mgr *manager.Manager) *Enhancer {
	return &Enhancer{
		config:  cfg,
		manager: mgr,
	}
}

// EnhanceOptions configures SBOM enhancement behavior
type EnhanceOptions struct {
	ProjectDir    string
	Deterministic bool
	OfflineMode   bool
	EmbedDigests  bool
	// BinaryPath, when set, points at a compiled Go artifact whose embedded
	// build information is used as the authoritative provenance source.
	BinaryPath string
	// GeneratorVersion is the version of goenv performing the enhancement. It is
	// recorded so an SBOM states exactly which goenv produced it.
	GeneratorVersion string
}

// GoenvMetadata contains Go-specific build and module context
type GoenvMetadata struct {
	GoVersion        string            `json:"go_version"`
	BuildContext     *BuildContext     `json:"build_context,omitempty"`
	ModuleContext    *ModuleContext    `json:"module_context,omitempty"`
	Binary           *BinaryProvenance `json:"binary,omitempty"`
	Timestamp        string            `json:"timestamp,omitempty"`
	Platform         string            `json:"platform"`
	ToolchainPath    string            `json:"toolchain_path,omitempty"`
	GeneratorVersion string            `json:"generator_version,omitempty"`
	SBOMTool         string            `json:"sbom_tool,omitempty"`
	SBOMToolVersion  string            `json:"sbom_tool_version,omitempty"`
}

// BuildContext captures build-time configuration
type BuildContext struct {
	Tags              []string          `json:"tags,omitempty"`
	CgoEnabled        bool              `json:"cgo_enabled"`
	GOOS              string            `json:"goos"`
	GOARCH            string            `json:"goarch"`
	Compiler          string            `json:"compiler"`
	LDFlags           string            `json:"ldflags,omitempty"`
	GCFlags           string            `json:"gcflags,omitempty"`
	BuildFlags        map[string]string `json:"build_flags,omitempty"`
	ConstraintsActive []string          `json:"constraints_active,omitempty"`
	PackagesExcluded  []string          `json:"packages_excluded,omitempty"`
}

// BuildConstraintInfo represents a build constraint found in source files
type BuildConstraintInfo struct {
	File       string
	Constraint string
	Satisfied  bool
}

// RetractedInfo represents retraction information for a module version
type RetractedInfo struct {
	Retracted          bool   `json:"retracted"`
	RetractionReason   string `json:"retraction_reason,omitempty"`
	RecommendedVersion string `json:"recommended_version,omitempty"`
}

// ModuleContext captures Go module metadata
type ModuleContext struct {
	GoModDigest    string             `json:"go_mod_digest,omitempty"`
	GoSumDigest    string             `json:"go_sum_digest,omitempty"`
	Vendored       bool               `json:"vendored"`
	VendorDigest   string             `json:"vendor_digest,omitempty"`
	ModuleProxy    string             `json:"module_proxy,omitempty"`
	Replaces       []ReplaceDirective `json:"replaces,omitempty"`
	RetractedCount int                `json:"retracted_count,omitempty"`
}

// ReplaceDirective documents a replace directive with risk assessment
type ReplaceDirective struct {
	Old       string `json:"old"`
	New       string `json:"new"`
	Type      string `json:"type"`       // "local-path", "version", "fork"
	RiskLevel string `json:"risk_level"` // "high", "medium", "low"
	Reason    string `json:"reason"`
}

// EnhanceCycloneDX adds goenv metadata to a CycloneDX SBOM
func (e *Enhancer) EnhanceCycloneDX(sbomPath string, opts EnhanceOptions) error {
	// Resolve the goenv-managed toolchain once so every enhancement step draws
	// from the same authoritative source (go list / go mod edit / go env).
	if e.toolchain == nil {
		e.toolchain = NewToolchain(e.config, e.manager, opts.ProjectDir, opts.OfflineMode)
	}

	// Read the CycloneDX SBOM
	data, err := os.ReadFile(sbomPath)
	if err != nil {
		return fmt.Errorf("failed to read SBOM: %w", err)
	}

	// Parse as generic JSON
	var sbom map[string]interface{}
	if err := json.Unmarshal(data, &sbom); err != nil {
		return fmt.Errorf("failed to parse SBOM JSON: %w", err)
	}

	// Gather Go-aware metadata
	metadata, err := e.gatherMetadata(opts)
	if err != nil {
		return fmt.Errorf("failed to gather metadata: %w", err)
	}

	// Record which SBOM generator produced the base document, sourced from the
	// tool's own self-reported metadata.tools entry (authoritative).
	metadata.SBOMTool, metadata.SBOMToolVersion = extractBaseToolInfo(sbom)

	// Inject into metadata section
	if err := e.injectMetadata(sbom, metadata, opts); err != nil {
		return fmt.Errorf("failed to inject metadata: %w", err)
	}

	// Enhance components with Go-specific data
	if err := e.enhanceComponents(sbom, opts); err != nil {
		return fmt.Errorf("failed to enhance components: %w", err)
	}

	// Make deterministic if requested
	if opts.Deterministic {
		e.makeDeterministic(sbom)
	}

	// Write enhanced SBOM
	return e.writeSBOM(sbomPath, sbom, opts.Deterministic)
}

// gatherMetadata collects Go build and module context
func (e *Enhancer) gatherMetadata(opts EnhanceOptions) (*GoenvMetadata, error) {
	metadata := &GoenvMetadata{
		Platform: fmt.Sprintf("%s/%s", platform.OS(), platform.Arch()),
	}

	if e.toolchain != nil {
		metadata.ToolchainPath = e.toolchain.GoBinary()
	}

	metadata.GeneratorVersion = opts.GeneratorVersion

	// Get current Go version (resolved patch version when available).
	if version, _, _, err := e.manager.GetCurrentVersionResolved(); err == nil && version != "" {
		metadata.GoVersion = version
	} else if version, _, err := e.manager.GetCurrentVersion(); err == nil {
		metadata.GoVersion = version
	}

	// When a compiled artifact is supplied, its embedded build info is the
	// highest-fidelity provenance available. A failure here is fatal because the
	// user explicitly asked to attest that binary.
	var binProv *BinaryProvenance
	if opts.BinaryPath != "" {
		p, err := ReadBinaryProvenance(opts.BinaryPath)
		if err != nil {
			return nil, err
		}
		binProv = p
		metadata.Binary = p
		if p.GoVersion != "" {
			metadata.GoVersion = p.GoVersion
		}
	}

	// Set timestamp (use build time if deterministic)
	if opts.Deterministic {
		// Use a fixed timestamp based on go.mod mtime for reproducibility
		if modPath := filepath.Join(opts.ProjectDir, "go.mod"); utils.FileExists(modPath) {
			if info, err := os.Stat(modPath); err == nil {
				metadata.Timestamp = info.ModTime().UTC().Format(time.RFC3339)
			}
		}
	} else {
		metadata.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	// Gather build context (binary provenance takes precedence over toolchain env).
	if buildCtx, err := e.gatherBuildContext(opts.ProjectDir, binProv); err == nil {
		metadata.BuildContext = buildCtx
	}

	// Gather module context
	if modCtx, err := e.gatherModuleContext(opts); err == nil {
		metadata.ModuleContext = modCtx
	}

	return metadata, nil
}

// gatherBuildContext extracts build configuration from the most authoritative
// source available. Precedence:
//  1. A compiled binary's embedded build info (exact flags used at build time).
//  2. The goenv-managed toolchain's effective `go env` values.
//  3. Process environment (documented best-effort fallback).
func (e *Enhancer) gatherBuildContext(projectDir string, binProv *BinaryProvenance) (*BuildContext, error) {
	ctx := &BuildContext{
		GOOS:     platform.OS(),
		GOARCH:   platform.Arch(),
		Compiler: "gc",
	}

	switch {
	case binProv != nil:
		// Authoritative: values embedded by the compiler in the artifact.
		applyBinaryBuildContext(ctx, binProv)
	case e.toolchain.Available():
		// Effective build settings from the managed toolchain.
		if env, err := e.toolchain.GoEnv("GOOS", "GOARCH", "CGO_ENABLED", "GOFLAGS", "GOAMD64"); err == nil {
			if v := env["GOOS"]; v != "" {
				ctx.GOOS = v
			}
			if v := env["GOARCH"]; v != "" {
				ctx.GOARCH = v
			}
			ctx.CgoEnabled = env["CGO_ENABLED"] == "1"
			ctx.Tags = parseBuildTags(env["GOFLAGS"])
			if v := env["GOAMD64"]; v != "" {
				ctx.BuildFlags = map[string]string{"GOAMD64": v}
			}
		}
	default:
		// Best-effort fallback when no toolchain is available.
		ctx.CgoEnabled = os.Getenv("CGO_ENABLED") == "1"
		ctx.Tags = parseBuildTags(os.Getenv("GOFLAGS"))
	}

	// Analyze build constraints
	if constraints, excluded, err := e.analyzeBuildConstraints(projectDir, ctx.Tags); err == nil {
		ctx.ConstraintsActive = constraints
		ctx.PackagesExcluded = excluded
	}

	return ctx, nil
}

// applyBinaryBuildContext maps debug/buildinfo settings onto a BuildContext.
// All embedded settings are preserved in BuildFlags so nothing the compiler
// recorded is lost; well-known keys are additionally surfaced as typed fields.
// Note: the Go toolchain deliberately does not embed -ldflags in build info, so
// that field is populated only when available from the managed toolchain path.
func applyBinaryBuildContext(ctx *BuildContext, p *BinaryProvenance) {
	if len(p.Settings) > 0 {
		ctx.BuildFlags = make(map[string]string, len(p.Settings))
		for k, v := range p.Settings {
			ctx.BuildFlags[k] = v
		}
	}
	if v := p.Settings["GOOS"]; v != "" {
		ctx.GOOS = v
	}
	if v := p.Settings["GOARCH"]; v != "" {
		ctx.GOARCH = v
	}
	if v, ok := p.Settings["CGO_ENABLED"]; ok {
		ctx.CgoEnabled = v == "1"
	}
	if v := p.Settings["-compiler"]; v != "" {
		ctx.Compiler = v
	}
	if v := p.Settings["-ldflags"]; v != "" {
		ctx.LDFlags = v
	}
	if v := p.Settings["-gcflags"]; v != "" {
		ctx.GCFlags = v
	}
	if v := p.Settings["-tags"]; v != "" {
		ctx.Tags = splitTags(v)
	}
}

// parseBuildTags extracts build tags from a GOFLAGS string, handling both the
// `-tags=a,b` and `-tags a,b` forms (the former is by far the most common and
// was previously missed entirely).
func parseBuildTags(goflags string) []string {
	fields := strings.Fields(goflags)
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if strings.HasPrefix(f, "-tags=") {
			return splitTags(strings.TrimPrefix(f, "-tags="))
		}
		if f == "-tags" && i+1 < len(fields) {
			return splitTags(fields[i+1])
		}
	}
	return nil
}

// splitTags splits a comma/space separated tag list, trimming quotes.
func splitTags(s string) []string {
	s = strings.Trim(s, "'\"")
	var out []string
	for _, t := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' }) {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// gatherModuleContext extracts Go module metadata
func (e *Enhancer) gatherModuleContext(opts EnhanceOptions) (*ModuleContext, error) {
	ctx := &ModuleContext{}

	projectDir := opts.ProjectDir
	if projectDir == "" {
		projectDir = "."
	}

	// Calculate go.mod digest
	if opts.EmbedDigests {
		if modPath := filepath.Join(projectDir, "go.mod"); utils.FileExists(modPath) {
			if hash, err := fileDigest(modPath); err == nil {
				ctx.GoModDigest = hash
			}
		}

		// Calculate go.sum digest
		if sumPath := filepath.Join(projectDir, "go.sum"); utils.FileExists(sumPath) {
			if hash, err := fileDigest(sumPath); err == nil {
				ctx.GoSumDigest = hash
			}
		}
	}

	// Check for vendoring
	vendorDir := filepath.Join(projectDir, "vendor")
	if utils.DirExists(vendorDir) {
		ctx.Vendored = true
		if opts.EmbedDigests {
			modulesPath := filepath.Join(vendorDir, "modules.txt")
			if utils.FileExists(modulesPath) {
				if hash, err := fileDigest(modulesPath); err == nil {
					ctx.VendorDigest = hash
				}
			}
		}
	}

	// Get module proxy from the managed toolchain (authoritative), falling back
	// to the process environment.
	if !opts.OfflineMode {
		if e.toolchain.Available() {
			if env, err := e.toolchain.GoEnv("GOPROXY"); err == nil {
				ctx.ModuleProxy = env["GOPROXY"]
			}
		}
		if ctx.ModuleProxy == "" {
			ctx.ModuleProxy = os.Getenv("GOPROXY")
		}
	}

	// Parse replace directives authoritatively via `go mod edit -json`, falling
	// back to the lightweight text parser when the toolchain is unavailable.
	if e.toolchain.Available() {
		if mod, err := e.toolchain.ModEdit(); err == nil {
			ctx.Replaces = classifyReplaces(mod.Replace)
		}
	}
	if ctx.Replaces == nil {
		if replaces, err := e.parseReplaceDirectives(projectDir); err == nil {
			ctx.Replaces = replaces
		}
	}

	return ctx, nil
}

// classifyReplaces converts authoritative go.mod replace directives into
// risk-classified ReplaceDirective records. Filesystem replacements (New.Version
// == "") are the highest-risk case because they bypass go.sum verification.
func classifyReplaces(replaces []GoModReplace) []ReplaceDirective {
	out := make([]ReplaceDirective, 0, len(replaces))
	for _, r := range replaces {
		d := ReplaceDirective{
			Old: joinModVer(r.Old.Path, r.Old.Version),
			New: joinModVer(r.New.Path, r.New.Version),
		}
		switch {
		case r.New.Version == "" || strings.HasPrefix(r.New.Path, ".") || filepath.IsAbs(r.New.Path):
			d.Type = "local-path"
			d.RiskLevel = "high"
			d.Reason = "Local/filesystem replacement is not subject to go.sum checksum verification"
		case r.Old.Path != r.New.Path:
			d.Type = "fork"
			d.RiskLevel = "medium"
			d.Reason = "Dependency redirected to a different module path (verify the source)"
		default:
			d.Type = "version"
			d.RiskLevel = "low"
			d.Reason = "Pinned to a specific version of the same module"
		}
		out = append(out, d)
	}
	return out
}

// joinModVer renders a module path with an optional version as "path@version".
func joinModVer(path, version string) string {
	if version == "" {
		return path
	}
	return path + "@" + version
}

// parseReplaceDirectives extracts and classifies replace directives from go.mod
func (e *Enhancer) parseReplaceDirectives(projectDir string) ([]ReplaceDirective, error) {
	modPath := filepath.Join(projectDir, "go.mod")
	if !utils.FileExists(modPath) {
		return nil, nil
	}

	data, err := os.ReadFile(modPath)
	if err != nil {
		return nil, err
	}

	var directives []ReplaceDirective
	lines := strings.Split(string(data), "\n")
	inReplace := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Track replace block
		if strings.HasPrefix(line, "replace (") {
			inReplace = true
			continue
		}
		if inReplace && line == ")" {
			inReplace = false
			continue
		}

		// Parse replace directive
		if strings.HasPrefix(line, "replace ") || (inReplace && line != "") {
			directive := e.parseReplaceLine(line)
			if directive != nil {
				directives = append(directives, *directive)
			}
		}
	}

	return directives, nil
}

// parseReplaceLine parses a single replace directive line
func (e *Enhancer) parseReplaceLine(line string) *ReplaceDirective {
	line = strings.TrimPrefix(line, "replace ")
	parts := strings.Split(line, "=>")
	if len(parts) != 2 {
		return nil
	}

	old := strings.TrimSpace(parts[0])
	new := strings.TrimSpace(parts[1])

	directive := &ReplaceDirective{
		Old: old,
		New: new,
	}

	// Classify type and risk
	if strings.HasPrefix(new, ".") || strings.HasPrefix(new, "/") {
		directive.Type = "local-path"
		directive.RiskLevel = "high"
		directive.Reason = "Local path dependency not subject to checksums"
	} else if strings.Contains(new, "github.com") && !strings.Contains(old, new) {
		directive.Type = "fork"
		directive.RiskLevel = "medium"
		directive.Reason = "Forked dependency - verify source"
	} else {
		directive.Type = "version"
		directive.RiskLevel = "low"
		directive.Reason = "Version override"
	}

	return directive
}

// injectMetadata adds goenv metadata to the SBOM
func (e *Enhancer) injectMetadata(sbom map[string]interface{}, metadata *GoenvMetadata, opts EnhanceOptions) error {
	// Get or create metadata section
	var metadataSection map[string]interface{}
	if meta, ok := sbom["metadata"].(map[string]interface{}); ok {
		metadataSection = meta
	} else {
		metadataSection = make(map[string]interface{})
		sbom["metadata"] = metadataSection
	}

	// CycloneDX requires custom properties in a "properties" array
	// Convert metadata to properties format
	properties := e.convertMetadataToProperties(metadata)

	// Get or create properties array
	var existingProps []interface{}
	if props, ok := metadataSection["properties"].([]interface{}); ok {
		existingProps = props
	}

	// Append goenv properties
	metadataSection["properties"] = append(existingProps, properties...)

	return nil
}

// convertMetadataToProperties converts GoenvMetadata to CycloneDX properties format
func (e *Enhancer) convertMetadataToProperties(metadata *GoenvMetadata) []interface{} {
	properties := []interface{}{}

	// Add Go version
	properties = append(properties, map[string]interface{}{
		"name":  "goenv:go_version",
		"value": metadata.GoVersion,
	})

	// Add platform
	properties = append(properties, map[string]interface{}{
		"name":  "goenv:platform",
		"value": metadata.Platform,
	})

	// Add timestamp
	if metadata.Timestamp != "" {
		properties = append(properties, map[string]interface{}{
			"name":  "goenv:timestamp",
			"value": metadata.Timestamp,
		})
	}

	// Add build context
	if metadata.BuildContext != nil {
		bc := metadata.BuildContext
		properties = append(properties, map[string]interface{}{
			"name":  "goenv:build_context.cgo_enabled",
			"value": fmt.Sprintf("%t", bc.CgoEnabled),
		})
		properties = append(properties, map[string]interface{}{
			"name":  "goenv:build_context.goos",
			"value": bc.GOOS,
		})
		properties = append(properties, map[string]interface{}{
			"name":  "goenv:build_context.goarch",
			"value": bc.GOARCH,
		})
		properties = append(properties, map[string]interface{}{
			"name":  "goenv:build_context.compiler",
			"value": bc.Compiler,
		})

		if len(bc.Tags) > 0 {
			tagsJSON, _ := json.Marshal(bc.Tags)
			properties = append(properties, map[string]interface{}{
				"name":  "goenv:build_context.tags",
				"value": string(tagsJSON),
			})
		}

		if bc.LDFlags != "" {
			properties = append(properties, map[string]interface{}{
				"name":  "goenv:build_context.ldflags",
				"value": bc.LDFlags,
			})
		}
	}

	// Add module context
	if metadata.ModuleContext != nil {
		mc := metadata.ModuleContext
		properties = append(properties, map[string]interface{}{
			"name":  "goenv:module_context.vendored",
			"value": fmt.Sprintf("%t", mc.Vendored),
		})

		if mc.GoModDigest != "" {
			properties = append(properties, map[string]interface{}{
				"name":  "goenv:module_context.go_mod_digest",
				"value": mc.GoModDigest,
			})
		}

		if mc.GoSumDigest != "" {
			properties = append(properties, map[string]interface{}{
				"name":  "goenv:module_context.go_sum_digest",
				"value": mc.GoSumDigest,
			})
		}

		if mc.ModuleProxy != "" {
			properties = append(properties, map[string]interface{}{
				"name":  "goenv:module_context.module_proxy",
				"value": mc.ModuleProxy,
			})
		}

		if len(mc.Replaces) > 0 {
			replacesJSON, _ := json.Marshal(mc.Replaces)
			properties = append(properties, map[string]interface{}{
				"name":  "goenv:module_context.replaces",
				"value": string(replacesJSON),
			})
		}
	}

	// Add authoritative binary provenance (from debug/buildinfo).
	if metadata.Binary != nil {
		b := metadata.Binary
		if b.VCSRevision != "" {
			properties = append(properties, map[string]interface{}{
				"name":  "goenv:binary.vcs_revision",
				"value": b.VCSRevision,
			})
			properties = append(properties, map[string]interface{}{
				"name":  "goenv:binary.vcs_modified",
				"value": fmt.Sprintf("%t", b.VCSModified),
			})
		}
		if b.VCSTime != "" {
			properties = append(properties, map[string]interface{}{
				"name":  "goenv:binary.vcs_time",
				"value": b.VCSTime,
			})
		}
		if b.MainVersion != "" {
			properties = append(properties, map[string]interface{}{
				"name":  "goenv:binary.main_version",
				"value": b.MainVersion,
			})
		}
	}

	if metadata.ToolchainPath != "" {
		properties = append(properties, map[string]interface{}{
			"name":  "goenv:toolchain_path",
			"value": metadata.ToolchainPath,
		})
	}

	// Provenance: exactly which goenv and SBOM generator produced this document.
	if metadata.GeneratorVersion != "" {
		properties = append(properties, map[string]interface{}{
			"name":  "goenv:generator_version",
			"value": metadata.GeneratorVersion,
		})
	}
	if metadata.SBOMTool != "" {
		properties = append(properties, map[string]interface{}{
			"name":  "goenv:sbom_tool",
			"value": metadata.SBOMTool,
		})
	}
	if metadata.SBOMToolVersion != "" {
		properties = append(properties, map[string]interface{}{
			"name":  "goenv:sbom_tool_version",
			"value": metadata.SBOMToolVersion,
		})
	}

	return properties
}

// extractBaseToolInfo reads the SBOM generator's own name and version from a
// CycloneDX document's metadata.tools, handling both the legacy array form and
// the CycloneDX 1.5 object form ({"components": [...]}). Returns empty strings
// when no tool is recorded.
func extractBaseToolInfo(sbom map[string]interface{}) (string, string) {
	meta, ok := sbom["metadata"].(map[string]interface{})
	if !ok {
		return "", ""
	}

	pick := func(m map[string]interface{}) (string, string) {
		name, _ := m["name"].(string)
		version, _ := m["version"].(string)
		return name, version
	}

	switch tools := meta["tools"].(type) {
	case []interface{}:
		// Legacy: metadata.tools is an array of {name, version, vendor}.
		for _, t := range tools {
			if m, ok := t.(map[string]interface{}); ok {
				if name, version := pick(m); name != "" {
					return name, version
				}
			}
		}
	case map[string]interface{}:
		// CycloneDX 1.5: metadata.tools.components is an array of components.
		if comps, ok := tools["components"].([]interface{}); ok {
			for _, c := range comps {
				if m, ok := c.(map[string]interface{}); ok {
					if name, version := pick(m); name != "" {
						return name, version
					}
				}
			}
		}
	}
	return "", ""
}

// GoenvSBOMProvenance is the goenv-recorded provenance read back from an
// enhanced CycloneDX SBOM. It lets downstream steps (e.g. SLSA attestation)
// reuse the authoritative values the enhancer already captured instead of
// re-deriving or hardcoding them.
type GoenvSBOMProvenance struct {
	GoVersion        string
	GeneratorVersion string
	SBOMTool         string
	SBOMToolVersion  string
	GOOS             string
	GOARCH           string
	CGOEnabled       bool
	BuildTags        []string
	Vendored         bool
}

// ReadGoenvProvenance parses an enhanced CycloneDX SBOM and returns the
// goenv:* provenance properties it recorded. Missing properties yield zero
// values, so callers can treat the result as best-effort.
func ReadGoenvProvenance(sbomPath string) (*GoenvSBOMProvenance, error) {
	data, err := os.ReadFile(sbomPath)
	if err != nil {
		return nil, err
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	props := map[string]string{}
	if meta, ok := doc["metadata"].(map[string]interface{}); ok {
		if arr, ok := meta["properties"].([]interface{}); ok {
			for _, p := range arr {
				if m, ok := p.(map[string]interface{}); ok {
					name, _ := m["name"].(string)
					val, _ := m["value"].(string)
					if name != "" {
						props[name] = val
					}
				}
			}
		}
	}

	prov := &GoenvSBOMProvenance{
		GoVersion:        props["goenv:go_version"],
		GeneratorVersion: props["goenv:generator_version"],
		SBOMTool:         props["goenv:sbom_tool"],
		SBOMToolVersion:  props["goenv:sbom_tool_version"],
		GOOS:             props["goenv:build_context.goos"],
		GOARCH:           props["goenv:build_context.goarch"],
		CGOEnabled:       props["goenv:build_context.cgo_enabled"] == "true",
		Vendored:         props["goenv:module_context.vendored"] == "true",
	}
	if tags := props["goenv:build_context.tags"]; tags != "" {
		var parsed []string
		if json.Unmarshal([]byte(tags), &parsed) == nil {
			prov.BuildTags = parsed
		}
	}
	return prov, nil
}

// enhanceComponents adds Go-specific data to individual components: the standard
// library as a first-class component, plus supply-chain annotations (replaced
// and retracted modules) sourced authoritatively from the Go toolchain.
func (e *Enhancer) enhanceComponents(sbom map[string]interface{}, opts EnhanceOptions) error {
	components, ok := sbom["components"].([]interface{})
	if !ok {
		components = []interface{}{}
	}

	// Add the standard library as a component so stdlib CVEs are in scope.
	if stdlibComponent, err := e.createStdlibComponent(opts.ProjectDir); err == nil && stdlibComponent != nil {
		components = append(components, stdlibComponent)
	}

	// Flag components redirected by replace directives (supply-chain risk).
	e.markReplacedComponents(components)

	// Flag components pinned to retracted versions.
	if err := e.markRetractedVersions(components, opts.ProjectDir); err != nil {
		// Non-fatal: retraction data is advisory and may require network access.
		_ = err
	}

	sbom["components"] = components
	return nil
}

// markReplacedComponents annotates components that are redirected by a go.mod
// replace directive, attaching the risk classification so policy engines and
// auditors can see filesystem/fork replacements at a glance.
func (e *Enhancer) markReplacedComponents(components []interface{}) {
	if !e.toolchain.Available() {
		return
	}
	mod, err := e.toolchain.ModEdit()
	if err != nil || len(mod.Replace) == 0 {
		return
	}

	byPath := make(map[string]ReplaceDirective)
	for i, r := range mod.Replace {
		byPath[r.Old.Path] = classifyReplaces(mod.Replace[i : i+1])[0]
	}

	for _, comp := range components {
		c, ok := comp.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := c["name"].(string)
		r, ok := byPath[name]
		if !ok {
			continue
		}
		goenvData := ensureGoenvMap(c)
		goenvData["replaced"] = true
		goenvData["replace_directive"] = map[string]interface{}{
			"target":     r.New,
			"type":       r.Type,
			"risk_level": r.RiskLevel,
			"reason":     r.Reason,
		}
	}
}

// ensureGoenvMap returns the component's "goenv" sub-map, creating it if needed.
func ensureGoenvMap(component map[string]interface{}) map[string]interface{} {
	if existing, ok := component["goenv"].(map[string]interface{}); ok {
		return existing
	}
	m := make(map[string]interface{})
	component["goenv"] = m
	return m
}

// createStdlibComponent determines the standard-library packages compiled into
// the project and emits them as a first-class CycloneDX component. Package
// discovery prefers the authoritative `go list` query (which honors build tags
// and GOOS/GOARCH and is independent of how projectDir is expressed) and falls
// back to a source scan only when the toolchain is unavailable.
func (e *Enhancer) createStdlibComponent(projectDir string) (map[string]interface{}, error) {
	if projectDir == "" {
		projectDir = "."
	}

	stdlibImports := e.stdlibPackages(projectDir)
	if len(stdlibImports) == 0 {
		return nil, nil
	}

	// Prefer the resolved toolchain version (e.g. "1.23.6") for accurate CVE
	// correlation; fall back to the raw selected version.
	goVersion := "unknown"
	if v, _, _, err := e.manager.GetCurrentVersionResolved(); err == nil && v != "" && v != "system" {
		goVersion = v
	} else if v, _, err := e.manager.GetCurrentVersion(); err == nil && v != "" {
		goVersion = v
	}

	// Create stdlib component in CycloneDX format
	component := map[string]interface{}{
		"type":        "library",
		"name":        "golang-stdlib",
		"version":     goVersion,
		"purl":        fmt.Sprintf("pkg:golang/stdlib@%s", goVersion),
		"bom-ref":     fmt.Sprintf("pkg:golang/stdlib@%s", goVersion),
		"description": fmt.Sprintf("Go standard library packages used by this project (%d packages)", len(stdlibImports)),
		"properties": []map[string]interface{}{
			{
				"name":  "goenv:stdlib_packages",
				"value": strings.Join(stdlibImports, ","),
			},
			{
				"name":  "goenv:stdlib_count",
				"value": fmt.Sprintf("%d", len(stdlibImports)),
			},
		},
	}

	return component, nil
}

// stdlibPackages returns the standard-library packages compiled into the
// project. It prefers the authoritative `go list -deps` query and falls back to
// a source-tree scan only when the toolchain is unavailable.
func (e *Enhancer) stdlibPackages(projectDir string) []string {
	if e.toolchain.Available() {
		if pkgs, err := e.toolchain.StdlibPackages(); err == nil && len(pkgs) > 0 {
			return pkgs
		}
	}
	pkgs, _ := e.discoverStdlibImports(projectDir)
	return pkgs
}

// discoverStdlibImports scans Go source files for stdlib imports
func (e *Enhancer) discoverStdlibImports(projectDir string) ([]string, error) {
	stdlibSet := make(map[string]bool)

	// Walk through all .go files
	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip vendor and hidden directories
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == "testdata" || (name != "." && name != ".." && strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Parse the Go file
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return nil // Skip files with parse errors
		}

		// Extract imports
		for _, imp := range node.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			// Check if it's a stdlib package
			if e.isStdlibPackage(importPath) {
				stdlibSet[importPath] = true
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert set to sorted slice
	stdlibImports := make([]string, 0, len(stdlibSet))
	for pkg := range stdlibSet {
		stdlibImports = append(stdlibImports, pkg)
	}
	sort.Strings(stdlibImports)

	return stdlibImports, nil
}

// isStdlibPackage determines if an import path is from the Go standard library
func (e *Enhancer) isStdlibPackage(importPath string) bool {
	// Stdlib packages don't have dots in the first path element
	// (except for some special cases like golang.org/x/...)

	// Explicitly exclude known non-stdlib patterns
	if strings.HasPrefix(importPath, "github.com/") ||
		strings.HasPrefix(importPath, "golang.org/x/") ||
		strings.HasPrefix(importPath, "gopkg.in/") ||
		strings.HasPrefix(importPath, "go.uber.org/") ||
		strings.Contains(importPath, ".com/") ||
		strings.Contains(importPath, ".io/") ||
		strings.Contains(importPath, ".org/") ||
		strings.Contains(importPath, ".net/") {
		return false
	}

	// Internal packages are not stdlib for third-party projects
	if strings.HasPrefix(importPath, e.config.Root) {
		return false
	}

	// Common stdlib packages (non-exhaustive, covers major ones)
	firstSegment := importPath
	if idx := strings.Index(importPath, "/"); idx > 0 {
		firstSegment = importPath[:idx]
	}

	// Stdlib packages typically don't have dots
	return !strings.Contains(firstSegment, ".")
}

// analyzeBuildConstraints scans Go source files for build constraints
func (e *Enhancer) analyzeBuildConstraints(projectDir string, activeTags []string) ([]string, []string, error) {
	if projectDir == "" {
		projectDir = "."
	}

	constraintsMap := make(map[string]bool)
	excludedPackages := []string{}
	satisfiedConstraints := []string{}

	// Build a set of active tags for fast lookup
	tagSet := make(map[string]bool)
	for _, tag := range activeTags {
		tagSet[tag] = true
	}

	// Add GOOS and GOARCH as implicit tags
	tagSet[platform.OS()] = true
	tagSet[platform.Arch()] = true

	// Walk through Go files
	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip vendor and hidden directories
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == "testdata" || (name != "." && name != ".." && strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process .go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Read first few lines for build constraints
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(data), "\n")
		for i := 0; i < len(lines) && i < 10; i++ {
			line := strings.TrimSpace(lines[i])

			// Check for //go:build constraint
			if strings.HasPrefix(line, "//go:build ") {
				constraint := strings.TrimPrefix(line, "//go:build ")
				constraintsMap[constraint] = true

				// Simplified constraint evaluation
				satisfied := e.evaluateConstraint(constraint, tagSet)
				if satisfied {
					satisfiedConstraints = append(satisfiedConstraints, constraint)
				} else {
					// This file would be excluded
					pkgPath := filepath.Dir(path)
					if relPath, err := filepath.Rel(projectDir, pkgPath); err == nil && relPath != "." {
						excludedPackages = append(excludedPackages, relPath)
					}
				}
				break
			}

			// Check for legacy // +build constraint
			if strings.HasPrefix(line, "// +build ") {
				constraint := strings.TrimPrefix(line, "// +build ")
				constraintsMap[constraint] = true

				satisfied := e.evaluateLegacyConstraint(constraint, tagSet)
				if satisfied {
					satisfiedConstraints = append(satisfiedConstraints, "// +build "+constraint)
				} else {
					pkgPath := filepath.Dir(path)
					if relPath, err := filepath.Rel(projectDir, pkgPath); err == nil && relPath != "." {
						excludedPackages = append(excludedPackages, relPath)
					}
				}
				break
			}

			// Stop at package declaration or first non-comment
			if strings.HasPrefix(line, "package ") || (line != "" && !strings.HasPrefix(line, "//")) {
				break
			}
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	// Deduplicate excluded packages
	excludedSet := make(map[string]bool)
	for _, pkg := range excludedPackages {
		excludedSet[pkg] = true
	}
	excludedList := make([]string, 0, len(excludedSet))
	for pkg := range excludedSet {
		excludedList = append(excludedList, pkg)
	}
	sort.Strings(excludedList)
	sort.Strings(satisfiedConstraints)

	return satisfiedConstraints, excludedList, nil
}

// evaluateConstraint evaluates a //go:build constraint
func (e *Enhancer) evaluateConstraint(constraint string, tags map[string]bool) bool {
	// Simplified evaluation: handles AND (&&), OR (||), NOT (!)
	// This is a basic implementation - full constraint parsing is complex

	// Remove parentheses for simple evaluation
	constraint = strings.ReplaceAll(constraint, "(", "")
	constraint = strings.ReplaceAll(constraint, ")", "")

	// Handle OR conditions
	if strings.Contains(constraint, "||") {
		parts := strings.Split(constraint, "||")
		for _, part := range parts {
			if e.evaluateConstraint(strings.TrimSpace(part), tags) {
				return true
			}
		}
		return false
	}

	// Handle AND conditions
	if strings.Contains(constraint, "&&") {
		parts := strings.Split(constraint, "&&")
		for _, part := range parts {
			if !e.evaluateConstraint(strings.TrimSpace(part), tags) {
				return false
			}
		}
		return true
	}

	// Handle NOT
	if strings.HasPrefix(constraint, "!") {
		return !tags[strings.TrimPrefix(constraint, "!")]
	}

	// Simple tag check
	return tags[constraint]
}

// evaluateLegacyConstraint evaluates a legacy // +build constraint
func (e *Enhancer) evaluateLegacyConstraint(constraint string, tags map[string]bool) bool {
	// Legacy format: space-separated = OR, comma-separated = AND, ! = NOT
	// Example: "linux,!cgo darwin" means (linux AND NOT cgo) OR darwin

	orGroups := strings.Fields(constraint)
	for _, group := range orGroups {
		andTags := strings.Split(group, ",")
		allMatch := true
		for _, tag := range andTags {
			tag = strings.TrimSpace(tag)
			if strings.HasPrefix(tag, "!") {
				if tags[strings.TrimPrefix(tag, "!")] {
					allMatch = false
					break
				}
			} else {
				if !tags[tag] {
					allMatch = false
					break
				}
			}
		}
		if allMatch {
			return true
		}
	}
	return false
}

// markRetractedVersions annotates components pinned to a retracted module
// version. When the toolchain and network are available it uses the
// authoritative `go list -m -retracted` query; otherwise it falls back to the
// retract directives declared in the local go.mod (via `go mod edit -json`),
// which works offline.
func (e *Enhancer) markRetractedVersions(components []interface{}, projectDir string) error {
	// module path -> human-readable rationale for the selected (SBOM) version.
	retractions := make(map[string]string)

	if e.toolchain.Available() {
		if r, err := e.toolchain.Retractions(); err == nil {
			for path, rationales := range r {
				retractions[path] = firstNonEmpty(strings.Join(rationales, "; "), "Version retracted upstream")
			}
		}
	}

	// Offline/self-module fallback: retract directives from the local go.mod.
	mainModule, localRetract := e.modfileRetractions()

	for _, comp := range components {
		component, ok := comp.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := component["name"].(string)
		version, _ := component["version"].(string)

		if reason, ok := retractions[name]; ok {
			markComponentRetracted(component, reason)
			continue
		}
		if name == mainModule && versionRetracted(version, localRetract) {
			markComponentRetracted(component, "Version retracted in go.mod")
		}
	}

	return nil
}

// modfileRetractions returns the main module path and its retract directives,
// sourced authoritatively via `go mod edit -json`.
func (e *Enhancer) modfileRetractions() (string, []GoModRetract) {
	if !e.toolchain.Available() {
		return "", nil
	}
	mod, err := e.toolchain.ModEdit()
	if err != nil {
		return "", nil
	}
	return mod.Module.Path, mod.Retract
}

// versionRetracted reports whether version matches a retract directive endpoint.
// Range interiors are intentionally not evaluated here (that requires semver
// ordering); the authoritative online path handles ranges precisely, so this
// offline fallback matches only explicit endpoints to avoid false positives.
func versionRetracted(version string, ranges []GoModRetract) bool {
	for _, r := range ranges {
		if version != "" && (version == r.Low || version == r.High) {
			return true
		}
	}
	return false
}

// markComponentRetracted flags a component as using a retracted version.
func markComponentRetracted(component map[string]interface{}, reason string) {
	g := ensureGoenvMap(component)
	g["retracted"] = true
	g["retraction_reason"] = reason
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// makeDeterministic ensures reproducible output
func (e *Enhancer) makeDeterministic(sbom map[string]interface{}) {
	// Sort components by name
	if components, ok := sbom["components"].([]interface{}); ok {
		sort.Slice(components, func(i, j int) bool {
			ci := components[i].(map[string]interface{})
			cj := components[j].(map[string]interface{})
			nameI, _ := ci["name"].(string)
			nameJ, _ := cj["name"].(string)
			return nameI < nameJ
		})
		sbom["components"] = components
	}

	// Replace random UUID with deterministic one if present
	if metadata, ok := sbom["metadata"].(map[string]interface{}); ok {
		if component, ok := metadata["component"].(map[string]interface{}); ok {
			// Generate deterministic UUID based on component content
			if name, ok := component["name"].(string); ok {
				deterministicUUID := generateDeterministicUUID(name)
				component["bom-ref"] = deterministicUUID
				metadata["component"] = component
			}
		}
		sbom["metadata"] = metadata
	}
}

// writeSBOM writes the enhanced SBOM to disk
func (e *Enhancer) writeSBOM(path string, sbom map[string]interface{}, deterministic bool) error {
	var data []byte
	var err error

	if deterministic {
		// Use deterministic JSON encoding (no random map ordering)
		data, err = json.MarshalIndent(sbom, "", "  ")
	} else {
		data, err = json.MarshalIndent(sbom, "", "  ")
	}

	if err != nil {
		return fmt.Errorf("failed to marshal SBOM: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write SBOM: %w", err)
	}

	return nil
}

// fileDigest computes SHA256 hash of a file
func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}

// generateDeterministicUUID creates a deterministic UUID from content
func generateDeterministicUUID(content string) string {
	hash := sha256.Sum256([]byte(content))
	// Format as UUID v5 style
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		hash[0:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
}

// ComputeSBOMDigest computes a normalized hash of an SBOM for reproducibility
func ComputeSBOMDigest(sbomPath, algorithm string) (string, error) {
	// Read SBOM
	data, err := os.ReadFile(sbomPath)
	if err != nil {
		return "", fmt.Errorf("failed to read SBOM: %w", err)
	}

	// Parse as JSON
	var sbom map[string]interface{}
	if err := json.Unmarshal(data, &sbom); err != nil {
		return "", fmt.Errorf("failed to parse SBOM JSON: %w", err)
	}

	// Normalize for reproducibility (remove timestamps, sort)
	normalized := normalizeSBOM(sbom)

	// Marshal to canonical JSON
	canonical, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("failed to marshal normalized SBOM: %w", err)
	}

	// Compute hash
	switch algorithm {
	case "sha256":
		hash := sha256.Sum256(canonical)
		return hex.EncodeToString(hash[:]), nil
	case "sha512":
		hash := sha512.Sum512(canonical)
		return hex.EncodeToString(hash[:]), nil
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}
}

// VerifyReproducible compares two SBOMs for reproducibility
func VerifyReproducible(sbom1Path, sbom2Path string) (match bool, diff string, err error) {
	// Compute normalized hashes
	hash1, err := ComputeSBOMDigest(sbom1Path, "sha256")
	if err != nil {
		return false, "", fmt.Errorf("failed to hash %s: %w", sbom1Path, err)
	}

	hash2, err := ComputeSBOMDigest(sbom2Path, "sha256")
	if err != nil {
		return false, "", fmt.Errorf("failed to hash %s: %w", sbom2Path, err)
	}

	// Compare hashes
	if hash1 == hash2 {
		return true, "", nil
	}

	// Generate diff information
	diff = fmt.Sprintf("Hash mismatch:\n  %s: %s\n  %s: %s",
		sbom1Path, hash1, sbom2Path, hash2)

	return false, diff, nil
}

// normalizeSBOM removes non-deterministic fields for comparison
func normalizeSBOM(sbom map[string]interface{}) map[string]interface{} {
	normalized := make(map[string]interface{})

	for key, value := range sbom {
		switch key {
		case "metadata":
			// Normalize metadata (remove timestamps)
			if meta, ok := value.(map[string]interface{}); ok {
				normalizedMeta := make(map[string]interface{})
				for k, v := range meta {
					if k != "timestamp" { // Exclude timestamp
						normalizedMeta[k] = v
					}
				}
				normalized[key] = normalizedMeta
			}
		case "components":
			// Sort components by name
			if components, ok := value.([]interface{}); ok {
				sorted := make([]interface{}, len(components))
				copy(sorted, components)
				sort.Slice(sorted, func(i, j int) bool {
					ci := sorted[i].(map[string]interface{})
					cj := sorted[j].(map[string]interface{})
					nameI, _ := ci["name"].(string)
					nameJ, _ := cj["name"].(string)
					return nameI < nameJ
				})
				normalized[key] = sorted
			}
		default:
			normalized[key] = value
		}
	}

	return normalized
}
