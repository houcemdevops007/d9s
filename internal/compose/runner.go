// Package compose wraps the docker compose CLI for project-level operations.
package compose

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/houcemdevops007/d9s/internal/domain"
)

// Runner executes docker compose commands.
type Runner struct {
	dockerContext string // optional: --context flag
}

// New creates a Runner for the given Docker context.
func New(dockerContext string) *Runner {
	return &Runner{dockerContext: dockerContext}
}

func (r *Runner) baseArgs() []string {
	args := []string{"compose"}
	if r.dockerContext != "" {
		args = append([]string{"--context", r.dockerContext}, args...)
	}
	return args
}

func (r *Runner) run(ctx context.Context, projectDir string, extra ...string) ([]byte, []byte, error) {
	args := append(r.baseArgs(), extra...)
	cmd := exec.CommandContext(ctx, "docker", args...)
	if projectDir != "" {
		cmd.Dir = projectDir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// psEntry is the JSON shape from docker compose ps --format json.
type psEntry struct {
	ID         string `json:"ID"`
	Name       string `json:"Name"`
	Command    string `json:"Command"`
	Project    string `json:"Project"`
	Service    string `json:"Service"`
	Created    string `json:"CreatedAt"`
	State      string `json:"State"`
	Status     string `json:"Status"`
	Health     string `json:"Health"`
	ExitCode   int    `json:"ExitCode"`
	Publishers []struct {
		URL           string `json:"URL"`
		TargetPort    int    `json:"TargetPort"`
		PublishedPort int    `json:"PublishedPort"`
		Protocol      string `json:"Protocol"`
	} `json:"Publishers"`
}

// PS runs docker compose ps --format json for the given project directory.
func (r *Runner) PS(ctx context.Context, projectDir string) ([]psEntry, error) {
	stdout, _, err := r.run(ctx, projectDir, "ps", "--format", "json", "--all")
	if err != nil {
		return nil, fmt.Errorf("compose ps: %w", err)
	}

	raw := bytes.TrimSpace(stdout)
	if len(raw) == 0 {
		return nil, nil
	}

	// docker compose ps --format json outputs either a JSON array or newline-delimited JSON objects
	if raw[0] == '[' {
		var entries []psEntry
		if err := json.Unmarshal(raw, &entries); err != nil {
			return nil, fmt.Errorf("compose ps parse: %w", err)
		}
		return entries, nil
	}

	// Newline-delimited JSON
	var entries []psEntry
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var e psEntry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// LSEntry is the JSON shape from docker compose ls --format json.
type LSEntry struct {
	Name        string `json:"Name"`
	Status      string `json:"Status"`
	ConfigFiles string `json:"ConfigFiles"`
}

// LS lists all compose projects visible from the current context.
func (r *Runner) LS(ctx context.Context) ([]LSEntry, error) {
	args := append(r.baseArgs(), "ls", "--format", "json", "--all")
	cmd := exec.CommandContext(ctx, "docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("compose ls: %w (stderr: %s)", err, stderr.String())
	}

	raw := bytes.TrimSpace(stdout.Bytes())
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var entries []LSEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("compose ls parse: %w", err)
	}
	return entries, nil
}

// Projects returns a list of ComposeProjects discovered via docker compose ls.
func (r *Runner) Projects(ctx context.Context) ([]domain.ComposeProject, error) {
	entries, err := r.LS(ctx)
	if err != nil {
		return nil, err
	}

	projects := make([]domain.ComposeProject, 0, len(entries))
	for _, e := range entries {
		cfg := strings.Split(e.ConfigFiles, ",")
		workDir := ""
		if len(cfg) > 0 && cfg[0] != "" {
			// Derive workdir from config path
			parts := strings.Split(cfg[0], string(os.PathSeparator))
			if len(parts) > 1 {
				workDir = strings.Join(parts[:len(parts)-1], string(os.PathSeparator))
			}
		}
		projects = append(projects, domain.ComposeProject{
			Name:        e.Name,
			ConfigFiles: cfg,
			WorkingDir:  workDir,
			Status:      e.Status,
		})
	}
	return projects, nil
}

// ServiceContainers maps compose ps output to ComposeServices.
func (r *Runner) ServiceContainers(ctx context.Context, projectDir string) ([]domain.ComposeService, error) {
	entries, err := r.PS(ctx, projectDir)
	if err != nil {
		return nil, err
	}

	byService := make(map[string]*domain.ComposeService)
	for _, e := range entries {
		svc, ok := byService[e.Service]
		if !ok {
			s := &domain.ComposeService{
				Name:    e.Service,
				Project: e.Project,
				State:   e.State,
			}
			byService[e.Service] = s
			svc = s
		}
		shortID := e.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		svc.Containers = append(svc.Containers, domain.Container{
			ID:             e.ID,
			ShortID:        shortID,
			Name:           e.Name,
			State:          e.State,
			Status:         e.Status,
			ComposeProject: e.Project,
			ComposeService: e.Service,
		})
		svc.Replicas = len(svc.Containers)
	}

	services := make([]domain.ComposeService, 0, len(byService))
	for _, s := range byService {
		services = append(services, *s)
	}
	return services, nil
}

// Up runs docker compose up -d for the given project directory.
func (r *Runner) Up(ctx context.Context, projectDir string) error {
	_, stderr, err := r.run(ctx, projectDir, "up", "-d")
	if err != nil {
		return fmt.Errorf("compose up: %w (stderr: %s)", err, string(stderr))
	}
	return nil
}

// Down runs docker compose down for the given project directory.
func (r *Runner) Down(ctx context.Context, projectDir string) error {
	_, stderr, err := r.run(ctx, projectDir, "down")
	if err != nil {
		return fmt.Errorf("compose down: %w (stderr: %s)", err, string(stderr))
	}
	return nil
}

// Pull runs docker compose pull for the given project directory.
func (r *Runner) Pull(ctx context.Context, projectDir string) error {
	_, stderr, err := r.run(ctx, projectDir, "pull")
	if err != nil {
		return fmt.Errorf("compose pull: %w (stderr: %s)", err, string(stderr))
	}
	return nil
}

// Build runs docker compose build for the given project directory.
func (r *Runner) Build(ctx context.Context, projectDir string) error {
	_, stderr, err := r.run(ctx, projectDir, "build")
	if err != nil {
		return fmt.Errorf("compose build: %w (stderr: %s)", err, string(stderr))
	}
	return nil
}

// Logs streams compose logs. Returns lines on the channel until ctx is done.
func (r *Runner) Logs(ctx context.Context, projectDir string, service string) (<-chan string, error) {
	args := append(r.baseArgs(), "logs", "--follow", "--timestamps")
	if service != "" {
		args = append(args, service)
	}
	cmd := exec.CommandContext(ctx, "docker", args...)
	if projectDir != "" {
		cmd.Dir = projectDir
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("compose logs start: %w", err)
	}

	ch := make(chan string, 256)
	go func() {
		defer close(ch)
		scanner := newScanner(stdout)
		for scanner.Scan() {
			select {
			case ch <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}
		cmd.Wait()
	}()
	return ch, nil
}

// ListContexts uses docker context ls --format json to list Docker contexts.
func ListContexts() ([]domain.DockerContext, error) {
	cmd := exec.Command("docker", "context", "ls", "--format", "json")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("context ls: %w", err)
	}

	type ctxEntry struct {
		Name        string `json:"Name"`
		Description string `json:"Description"`
		DockerEndpoint string `json:"DockerEndpoint"`
		Current     bool   `json:"Current"`
	}

	raw := bytes.TrimSpace(stdout.Bytes())
	if len(raw) == 0 {
		return nil, nil
	}

	var contexts []domain.DockerContext

	// Try array format first
	if raw[0] == '[' {
		var entries []ctxEntry
		if err := json.Unmarshal(raw, &entries); err == nil {
			for _, e := range entries {
				contexts = append(contexts, domain.DockerContext{
					Name:        e.Name,
					Description: e.Description,
					Endpoint:    e.DockerEndpoint,
					Current:     e.Current,
				})
			}
			return contexts, nil
		}
	}

	// Newline-delimited JSON
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var e ctxEntry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		contexts = append(contexts, domain.DockerContext{
			Name:        e.Name,
			Description: e.Description,
			Endpoint:    e.DockerEndpoint,
			Current:     e.Current,
		})
	}
	return contexts, nil
}

// SwitchContext runs docker context use <name>.
func SwitchContext(name string) error {
	cmd := exec.Command("docker", "context", "use", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("context switch: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}
