package compliance

import (
	"testing"

	"github.com/go-nv/goenv/internal/sbom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func result(critical, high, medium, low int) *sbom.ScanResult {
	return &sbom.ScanResult{
		Summary: sbom.VulnerabilitySummary{
			Critical: critical,
			High:     high,
			Medium:   medium,
			Low:      low,
			Total:    critical + high + medium + low,
		},
	}
}

// TestCheckFailOnCondition locks in the CI gate semantics. This complements the
// e2e test that proved the gate is actually reached on the stdout path (it was
// previously only applied when writing results to a file).
func TestCheckFailOnCondition(t *testing.T) {
	cases := []struct {
		name    string
		failOn  string
		result  *sbom.ScanResult
		wantErr bool
	}{
		{"empty never fails", "", result(9, 9, 9, 9), false},
		{"any with vulns fails", "any", result(0, 0, 0, 1), true},
		{"any with none passes", "any", result(0, 0, 0, 0), false},
		{"critical with critical fails", "critical", result(1, 0, 0, 0), true},
		{"critical with only high passes", "critical", result(0, 5, 0, 0), false},
		{"high with high fails", "high", result(0, 1, 0, 0), true},
		{"high with critical fails", "high", result(1, 0, 0, 0), true},
		{"high with only medium passes", "high", result(0, 0, 3, 0), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := checkFailOnCondition(c.result, c.failOn)
			if c.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
