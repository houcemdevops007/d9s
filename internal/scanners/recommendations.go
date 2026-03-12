// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
package scanners

import (
	"fmt"
	"strings"

	"github.com/houcemdevops007/d9s/internal/domain"
)

// BestPracticesEngine analyzes image metadata and scan results to provide recommendations.
type BestPracticesEngine struct{}

// NewBestPracticesEngine creates a new engine.
func NewBestPracticesEngine() *BestPracticesEngine {
	return &BestPracticesEngine{}
}

// Analyze evaluates the image and returns a list of recommendations.
func (e *BestPracticesEngine) Analyze(img domain.ImageDetails, scan domain.SecurityScanResult) []domain.BestPracticeRecommendation {
	var recs []domain.BestPracticeRecommendation

	// 1. Root User check
	if img.Config.User == "" || img.Config.User == "0" || img.Config.User == "root" {
		recs = append(recs, domain.BestPracticeRecommendation{
			ID:             "BP-001",
			Title:          "Run as non-root user",
			Severity:       "critical",
			Category:       "runtime",
			Reason:         "The image is configured to run as root by default.",
			Recommendation: "Use 'USER <uid>:<gid>' in your Dockerfile to run as a non-privileged user.",
			Example:        "USER 10001:10001",
		})
	}

	// 2. Secret exposure in ENV
	sensitiveKeys := []string{"PASSWORD", "PASS", "SECRET", "KEY", "TOKEN", "AWS_ACCESS", "CREDENTIALS"}
	for _, env := range img.Config.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToUpper(parts[0])
		for _, s := range sensitiveKeys {
			if strings.Contains(key, s) {
				recs = append(recs, domain.BestPracticeRecommendation{
					ID:             "BP-002",
					Title:          "Potential secret in environment variable",
					Severity:       "critical",
					Category:       "secrets",
					Reason:         fmt.Sprintf("Environment variable '%s' might contain a secret.", parts[0]),
					Recommendation: "Do not store secrets in ENV vars. Use Docker Secrets or a backend Vault.",
					Evidence:       []string{env},
				})
				break
			}
		}
	}

	// 3. Privileged ports
	for _, port := range img.Config.ExposedPorts {
		p := 0
		fmt.Sscanf(port, "%d", &p)
		if p > 0 && p < 1024 {
			recs = append(recs, domain.BestPracticeRecommendation{
				ID:             "BP-003",
				Title:          "Use non-privileged ports",
				Severity:       "warning",
				Category:       "runtime",
				Reason:         fmt.Sprintf("Image exposes privileged port %d.", p),
				Recommendation: "Avoid port numbers below 1024 so the container doesn't need extra capabilities.",
				Example:        "EXPOSE 8080",
			})
		}
	}

	// 4. Critical Vulnerability correlation
	if scan.Summary.Critical > 0 {
		recs = append(recs, domain.BestPracticeRecommendation{
			ID:             "BP-004",
			Title:          "Address Critical Vulnerabilities",
			Severity:       "critical",
			Category:       "packages",
			Reason:         fmt.Sprintf("Image has %d critical vulnerabilities.", scan.Summary.Critical),
			Recommendation: "Update your base image or upgrade vulnerable packages.",
			Evidence:       []string{"Found via Trivy/Snyk scan"},
		})
	}

	// 5. Missing healthcheck
	healthcheckFound := false
	for k := range img.Config.Labels {
		if strings.Contains(strings.ToLower(k), "healthcheck") {
			healthcheckFound = true
			break
		}
	}
	// Note: Inspect JSON might have formal "Healthcheck" field, but we check common markers.
	if !healthcheckFound {
		recs = append(recs, domain.BestPracticeRecommendation{
			ID:             "BP-005",
			Title:          "Missing HEALTHCHECK instruction",
			Severity:       "info",
			Category:       "runtime",
			Reason:         "No healthcheck detected in image configuration.",
			Recommendation: "Add a HEALTHCHECK instruction to allow Docker to monitor container health.",
			Example:        "HEALTHCHECK --interval=30s CMD curl -f http://localhost/ || exit 1",
		})
	}

	return recs
}
