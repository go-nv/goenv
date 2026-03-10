package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/sbom"
)

func main() {
	cfg := config.Load()
	mgr := manager.NewManager(cfg, nil)
	enhancer := sbom.NewEnhancer(cfg, mgr)

	// Test stdlib detection on current directory (goenv root)
	projectDirBytes, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting project directory: %v\n", err)
		os.Exit(1)
	}

	if len(projectDirBytes) == 0 {
		fmt.Fprintf(os.Stderr, "Could not determine project directory\n")
		os.Exit(1)
	}

	projectDir := string(projectDirBytes)
	opts := sbom.EnhanceOptions{
		ProjectDir:    projectDir,
		Deterministic: true,
		EmbedDigests:  false,
	}

	// Enhance the test SBOM
	sbomPath := fmt.Sprintf("%s/test-base-sbom.json", projectDir)
	err = enhancer.EnhanceCycloneDX(sbomPath, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enhancing SBOM: %v\n", err)
		os.Exit(1)
	}

	// Read and display the enhanced SBOM
	data, err := os.ReadFile(sbomPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading SBOM: %v\n", err)
		os.Exit(1)
	}

	var sbomData map[string]interface{}
	if err := json.Unmarshal(data, &sbomData); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing SBOM: %v\n", err)
		os.Exit(1)
	}

	// Pretty print
	pretty, _ := json.MarshalIndent(sbomData, "", "  ")
	fmt.Println(string(pretty))
}
