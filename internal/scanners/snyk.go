// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
package scanners

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/houcemdevops007/d9s/internal/domain"
)

// SnykScanner implements the Scanner interface using the Snyk CLI.
type SnykScanner struct {
	binaryPath string
	dockerHost string
}

// NewSnykScanner creates a new Snyk scanner.
func NewSnykScanner(dockerHost string) *SnykScanner {
	path, err := exec.LookPath("snyk")
	if err != nil {
		path = "snyk"
	}
	return &SnykScanner{binaryPath: path, dockerHost: dockerHost}
}

func (s *SnykScanner) Name() string { return "Snyk" }

// snykResponse is the JSON structure from 'snyk container test --json'.
type snykResponse struct {
	Vulnerabilities []struct {
		ID               string `json:"id"`
		Title            string `json:"title"`
		Description      string `json:"description"`
		Severity         string `json:"severity"`
		PackageName      string `json:"packageName"`
		Version          string `json:"version"`
		FixedIn          []string `json:"fixedIn"`
		References       []struct {
			URL   string `json:"url"`
		} `json:"references"`
	} `json:"vulnerabilities"`
	Error string `json:"error"`
}

func (s *SnykScanner) ScanImage(ctx context.Context, imageID string) (domain.SecurityScanResult, error) {
	result := domain.SecurityScanResult{
		ImageID:  imageID,
		Scanner:  "Snyk",
		ScanTime: time.Now(),
	}

	// Command: snyk container test <imageID> --json
	cmd := exec.CommandContext(ctx, s.binaryPath, "container", "test", imageID, "--json")
	if s.dockerHost != "" {
		cmd.Env = append(os.Environ(), "DOCKER_HOST="+s.dockerHost)
	}
	output, err := cmd.Output()
	// Snyk exits with code 1 if vulnerabilities are found, so we check output even if err != nil
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return result, fmt.Errorf("snyk execution failed: %w (check if snyk is installed and authenticated)", err)
		}
	}

	if len(output) == 0 {
		return result, fmt.Errorf("snyk returned empty output (check 'snyk auth')")
	}

	var raw snykResponse
	if err := json.Unmarshal(output, &raw); err != nil {
		// Sometimes Snyk returns a single error object instead of the full report
		var errResp struct{ Error string `json:"error"` }
		if err2 := json.Unmarshal(output, &errResp); err2 == nil && errResp.Error != "" {
			return result, fmt.Errorf("snyk error: %s", errResp.Error)
		}
		return result, fmt.Errorf("decode snyk output: %w", err)
	}

	for _, v := range raw.Vulnerabilities {
		fixed := ""
		if len(v.FixedIn) > 0 {
			fixed = v.FixedIn[0]
		}
		url := ""
		if len(v.References) > 0 {
			url = v.References[0].URL
		}

		result.Vulnerabilities = append(result.Vulnerabilities, domain.VulnerabilityFinding{
			ID:           v.ID,
			Title:        v.Title,
			Description:  v.Description,
			Severity:     strings.ToUpper(v.Severity),
			Package:      v.PackageName,
			Version:      v.Version,
			FixedVersion: fixed,
			PrimaryURL:   url,
		})
		updateSummary(&result.Summary, strings.ToUpper(v.Severity))
	}

	return result, nil
}
