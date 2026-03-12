// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
// Package actions provides confirmed user operations on containers and projects.
package actions

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/houcemdevops007/d9s/internal/compose"
	"github.com/houcemdevops007/d9s/internal/dockerapi"
)

// Runner exposes all user-facing actions.
type Runner struct {
	docker  *dockerapi.Client
	compose *compose.Runner
}

// New creates an action runner.
func New(docker *dockerapi.Client, composeRunner *compose.Runner) *Runner {
	return &Runner{docker: docker, compose: composeRunner}
}

// ContainerStart starts a container.
func (r *Runner) ContainerStart(ctx context.Context, id string) error {
	return r.docker.ContainerStart(ctx, id)
}

// ContainerStop stops a container.
func (r *Runner) ContainerStop(ctx context.Context, id string) error {
	return r.docker.ContainerStop(ctx, id)
}

// ContainerRestart restarts a container.
func (r *Runner) ContainerRestart(ctx context.Context, id string) error {
	return r.docker.ContainerRestart(ctx, id)
}

// ContainerRemove removes a container.
func (r *Runner) ContainerRemove(ctx context.Context, id string) error {
	return r.docker.ContainerRemove(ctx, id)
}

// ExecShell opens an interactive shell inside a container.
// This replaces the current process with the shell session.
func (r *Runner) ExecShell(containerID string, dockerContext string) error {
	args := []string{"exec", "-it", containerID, "sh", "-c", "which bash && exec bash || exec sh"}
	if dockerContext != "" {
		args = append([]string{"--context", dockerContext}, args...)
	}
	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("exec shell: %w", err)
	}
	return nil
}

// ComposeUp runs docker compose up -d.
func (r *Runner) ComposeUp(ctx context.Context, projectDir string) error {
	return r.compose.Up(ctx, projectDir)
}

// ComposeDown runs docker compose down.
func (r *Runner) ComposeDown(ctx context.Context, projectDir string) error {
	return r.compose.Down(ctx, projectDir)
}

// ComposePull runs docker compose pull.
func (r *Runner) ComposePull(ctx context.Context, projectDir string) error {
	return r.compose.Pull(ctx, projectDir)
}

// ComposeBuild runs docker compose build.
func (r *Runner) ComposeBuild(ctx context.Context, projectDir string) error {
	return r.compose.Build(ctx, projectDir)
}

// ImageRemove removes a Docker image.
func (r *Runner) ImageRemove(ctx context.Context, id string) error {
	return r.docker.ImageRemove(ctx, id)
}

// VolumeRemove removes a Docker volume.
func (r *Runner) VolumeRemove(ctx context.Context, name string) error {
	return r.docker.VolumeRemove(ctx, name)
}

// NetworkRemove removes a Docker network.
func (r *Runner) NetworkRemove(ctx context.Context, id string) error {
	return r.docker.NetworkRemove(ctx, id)
}
