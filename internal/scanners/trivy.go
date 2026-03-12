// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
package scanners

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/houcemdevops007/d9s/internal/domain"
)

// Scanner defines the interface for image security scanners.
type Scanner interface {
	ScanImage(ctx context.Context, imageID string) (domain.SecurityScanResult, error)
	Name() string
}

// TrivyScanner implements the Scanner interface using the Trivy CLI.
type TrivyScanner struct {
	binaryPath string
	dockerHost string
}

// NewTrivyScanner creates a new Trivy scanner.
func NewTrivyScanner(dockerHost string) *TrivyScanner {
	path, err := exec.LookPath("trivy")
	if err != nil {
		path = "trivy" // fallback to raw command
	}
	return &TrivyScanner{binaryPath: path, dockerHost: dockerHost}
}

func (s *TrivyScanner) Name() string { return "Trivy" }

// trivyResponse is the simplified JSON structure from 'trivy image --format json'.
type trivyResponse struct {
	Results []struct {
		Target          string `json:"Target"`
		Class           string `json:"Class"`
		Type            string `json:"Type"`
		Vulnerabilities []struct {
			VulnerabilityID  string `json:"VulnerabilityID"`
			PkgName          string `json:"PkgName"`
			InstalledVersion string `json:"InstalledVersion"`
			FixedVersion     string `json:"FixedVersion"`
			Title            string `json:"Title"`
			Description      string `json:"Description"`
			Severity         string `json:"Severity"`
			PrimaryURL       string `json:"PrimaryURL"`
		} `json:"Vulnerabilities"`
		Misconfigurations []struct {
			ID          string `json:"ID"`
			Type        string `json:"Type"`
			Title       string `json:"Title"`
			Message     string `json:"Message"`
			Resolution  string `json:"Resolution"`
			Severity    string `json:"Severity"`
		} `json:"Misconfigurations"`
		Secrets []struct {
			RuleID   string `json:"RuleID"`
			Category string `json:"Category"`
			Severity string `json:"Severity"`
			Title    string `json:"Title"`
			Match    string `json:"Match"`
		} `json:"Secrets"`
	} `json:"Results"`
}

func (s *TrivyScanner) ScanImage(ctx context.Context, imageID string) (domain.SecurityScanResult, error) {
	result := domain.SecurityScanResult{
		ImageID:  imageID,
		Scanner:  "Trivy",
		ScanTime: time.Now(),
	}

	// Command: trivy image --format json --quiet --no-progress <imageID>
	cmd := exec.CommandContext(ctx, s.binaryPath, "image", "--format", "json", "--quiet", "--no-progress", imageID)
	if s.dockerHost != "" {
		cmd.Env = append(os.Environ(), "DOCKER_HOST="+s.dockerHost)
	}
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return result, fmt.Errorf("trivy scan failed: %s", string(exitErr.Stderr))
		}
		return result, fmt.Errorf("trivy scan failed: %w", err)
	}

	var raw trivyResponse
	if err := json.Unmarshal(output, &raw); err != nil {
		return result, fmt.Errorf("decode trivy output: %w", err)
	}

	for _, res := range raw.Results {
		// Vulns
		for _, v := range res.Vulnerabilities {
			result.Vulnerabilities = append(result.Vulnerabilities, domain.VulnerabilityFinding{
				ID:           v.VulnerabilityID,
				Title:        v.Title,
				Description:  v.Description,
				Severity:     v.Severity,
				Package:      v.PkgName,
				Version:      v.InstalledVersion,
				FixedVersion: v.FixedVersion,
				PrimaryURL:   v.PrimaryURL,
			})
			updateSummary(&result.Summary, v.Severity)
		}
		// Misconfigs
		for _, m := range res.Misconfigurations {
			result.Misconfigs = append(result.Misconfigs, domain.MisconfigurationFinding{
				ID:         m.ID,
				Type:       m.Type,
				Title:      m.Title,
				Severity:   m.Severity,
				Message:    m.Message,
				Resolution: m.Resolution,
			})
			updateSummary(&result.Summary, m.Severity)
		}
		// Secrets
		for _, sec := range res.Secrets {
			result.Secrets = append(result.Secrets, domain.SecretFinding{
				RuleID:   sec.RuleID,
				Category: sec.Category,
				Severity: sec.Severity,
				Title:    sec.Title,
				Match:    sec.Match,
			})
			updateSummary(&result.Summary, sec.Severity)
		}
	}

	return result, nil
}

func updateSummary(s *domain.ScanSummary, sev string) {
	switch sev {
	case "CRITICAL":
		s.Critical++
	case "HIGH":
		s.High++
	case "MEDIUM":
		s.Medium++
	case "LOW":
		s.Low++
	default:
		s.Unknown++
	}
}
