// Package dockerapi provides a client for the Docker Engine API via Unix socket.
// It uses only the Go standard library (net/http + encoding/json).
package dockerapi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/houcemdevops007/d9s/internal/domain"
)

const (
	defaultSocket  = "/var/run/docker.sock"
	apiVersion     = "v1.41"
)

// Client communicates with the Docker daemon.
type Client struct {
	http       *http.Client
	socketPath string
}

// New creates a new Docker API client using the given Unix socket path.
// If socketPath is empty, the default /var/run/docker.sock is used.
func New(socketPath string) *Client {
	if socketPath == "" {
		socketPath = defaultSocket
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &Client{
		http:       &http.Client{Transport: transport},
		socketPath: socketPath,
	}
}

func (c *Client) url(path string) string {
	return fmt.Sprintf("http://localhost/%s%s", apiVersion, path)
}

func (c *Client) get(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url(path), nil)
	if err != nil {
		return nil, err
	}
	return c.http.Do(req)
}

func (c *Client) post(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url(path), nil)
	if err != nil {
		return nil, err
	}
	return c.http.Do(req)
}

func (c *Client) delete(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.url(path), nil)
	if err != nil {
		return nil, err
	}
	return c.http.Do(req)
}

// Ping checks daemon connectivity.
func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.get(ctx, "/_ping")
	if err != nil {
		return fmt.Errorf("docker ping: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("docker ping: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// apiContainer is the raw Docker API container list response shape.
type apiContainer struct {
	ID      string            `json:"Id"`
	Names   []string          `json:"Names"`
	Image   string            `json:"Image"`
	ImageID string            `json:"ImageID"`
	Command string            `json:"Command"`
	Created int64             `json:"Created"`
	State   string            `json:"State"`
	Status  string            `json:"Status"`
	Labels  map[string]string `json:"Labels"`
	Ports   []struct {
		IP          string `json:"IP"`
		PrivatePort uint16 `json:"PrivatePort"`
		PublicPort  uint16 `json:"PublicPort"`
		Type        string `json:"Type"`
	} `json:"Ports"`
	NetworkSettings struct {
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
			Gateway   string `json:"Gateway"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
	HostConfig struct {
		NetworkMode string `json:"NetworkMode"`
	} `json:"HostConfig"`
}

func toContainer(a apiContainer) domain.Container {
	name := ""
	if len(a.Names) > 0 {
		name = a.Names[0]
	}
	shortID := a.ID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}

	ports := make([]domain.Port, 0, len(a.Ports))
	for _, p := range a.Ports {
		ports = append(ports, domain.Port{
			IP:          p.IP,
			PrivatePort: p.PrivatePort,
			PublicPort:  p.PublicPort,
			Type:        p.Type,
		})
	}

	networks := make(map[string]domain.ContainerNetwork, len(a.NetworkSettings.Networks))
	for k, v := range a.NetworkSettings.Networks {
		networks[k] = domain.ContainerNetwork{
			IPAddress: v.IPAddress,
			Gateway:   v.Gateway,
		}
	}

	c := domain.Container{
		ID:          a.ID,
		ShortID:     shortID,
		Name:        name,
		Image:       a.Image,
		ImageID:     a.ImageID,
		Command:     a.Command,
		Created:     a.Created,
		State:       a.State,
		Status:      a.Status,
		Labels:      a.Labels,
		Ports:       ports,
		NetworkMode: a.HostConfig.NetworkMode,
		Networks:    networks,
	}

	// Extract Compose labels
	if v, ok := a.Labels["com.docker.compose.project"]; ok {
		c.ComposeProject = v
	}
	if v, ok := a.Labels["com.docker.compose.service"]; ok {
		c.ComposeService = v
	}

	return c
}

// ListContainers returns all containers (running and stopped).
func (c *Client) ListContainers(ctx context.Context, all bool) ([]domain.Container, error) {
	path := "/containers/json"
	if all {
		path += "?all=1"
	}
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	defer resp.Body.Close()

	var raw []apiContainer
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("list containers decode: %w", err)
	}

	containers := make([]domain.Container, 0, len(raw))
	for _, r := range raw {
		containers = append(containers, toContainer(r))
	}
	return containers, nil
}

// InspectContainer returns detailed info for a container.
func (c *Client) InspectContainer(ctx context.Context, id string) (map[string]interface{}, error) {
	resp, err := c.get(ctx, "/containers/"+id+"/json")
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("container not found: %s", id)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("inspect container decode: %w", err)
	}
	return result, nil
}

// Logs streams container logs. It returns a channel of log lines.
// The caller must cancel ctx to stop streaming.
func (c *Client) Logs(ctx context.Context, id string, tail int) (<-chan domain.LogLine, error) {
	path := fmt.Sprintf("/containers/%s/logs?stdout=1&stderr=1&timestamps=1&tail=%d&follow=1", id, tail)
	resp, err := c.get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("logs: %w", err)
	}

	ch := make(chan domain.LogLine, 256)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			// Docker multiplexed stream: first 8 bytes are a header
			// [stream_type(1), 0, 0, 0, size(4)] then payload
			// We strip the header if present (non-printable first byte).
			if len(line) > 8 && (line[0] == 1 || line[0] == 2) {
				line = line[8:]
			}
			// Parse timestamp prefix if present
			var ts time.Time
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
					ts = t
					line = parts[1]
				}
			}
			logLine := domain.LogLine{
				ContainerID: id,
				Timestamp:   ts,
				Text:        line,
			}
			select {
			case ch <- logLine:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

// apiStatsResponse is the Docker stats API response (non-stream single shot).
type apiStatsResponse struct {
	Read     string `json:"read"`
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64   `json:"total_usage"`
			PercpuUsage []uint64 `json:"percpu_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs     int    `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemCPUUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
		Stats struct {
			Cache uint64 `json:"cache"`
		} `json:"stats"`
	} `json:"memory_stats"`
	PidsStats struct {
		Current uint64 `json:"current"`
	} `json:"pids_stats"`
	Name string `json:"name"`
}

// Stats fetches a single stats snapshot for a container.
func (c *Client) Stats(ctx context.Context, id string) (domain.RuntimeStats, error) {
	path := fmt.Sprintf("/containers/%s/stats?stream=false", id)
	resp, err := c.get(ctx, path)
	if err != nil {
		return domain.RuntimeStats{}, fmt.Errorf("stats: %w", err)
	}
	defer resp.Body.Close()

	var s apiStatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return domain.RuntimeStats{}, fmt.Errorf("stats decode: %w", err)
	}

	// Calculate CPU %
	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage) - float64(s.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(s.CPUStats.SystemCPUUsage) - float64(s.PreCPUStats.SystemCPUUsage)
	numCPU := float64(s.CPUStats.OnlineCPUs)
	if numCPU == 0 {
		numCPU = float64(len(s.CPUStats.CPUUsage.PercpuUsage))
	}
	cpuPercent := 0.0
	if sysDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / sysDelta) * numCPU * 100.0
	}

	// Memory
	memUsage := s.MemoryStats.Usage - s.MemoryStats.Stats.Cache
	memLimit := s.MemoryStats.Limit
	memPercent := 0.0
	if memLimit > 0 {
		memPercent = float64(memUsage) / float64(memLimit) * 100.0
	}

	name := s.Name
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}

	return domain.RuntimeStats{
		ContainerID: id,
		Name:        name,
		CPUPercent:  cpuPercent,
		MemUsage:    memUsage,
		MemLimit:    memLimit,
		MemPercent:  memPercent,
		PidsCount:   s.PidsStats.Current,
		Timestamp:   time.Now(),
	}, nil
}

// apiEvent is the Docker events stream response shape.
type apiEvent struct {
	Type   string `json:"Type"`
	Action string `json:"Action"`
	Actor  struct {
		ID         string            `json:"ID"`
		Attributes map[string]string `json:"Attributes"`
	} `json:"Actor"`
	Time      int64 `json:"time"`
	TimeNano  int64 `json:"timeNano"`
}

// Events streams Docker daemon events until ctx is cancelled.
func (c *Client) Events(ctx context.Context) (<-chan domain.RuntimeEvent, error) {
	resp, err := c.get(ctx, "/events")
	if err != nil {
		return nil, fmt.Errorf("events: %w", err)
	}

	ch := make(chan domain.RuntimeEvent, 128)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		dec := json.NewDecoder(resp.Body)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			var e apiEvent
			if err := dec.Decode(&e); err != nil {
				if err == io.EOF {
					return
				}
				continue
			}
			actor := e.Actor.ID
			if name, ok := e.Actor.Attributes["name"]; ok {
				actor = name
			}
			msg := fmt.Sprintf("%s %s %s", e.Type, e.Action, actor)
			event := domain.RuntimeEvent{
				Time:    time.Unix(e.Time, 0),
				Type:    e.Type,
				Action:  e.Action,
				Actor:   actor,
				Message: msg,
			}
			select {
			case ch <- event:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

// ContainerStart starts a container.
func (c *Client) ContainerStart(ctx context.Context, id string) error {
	resp, err := c.post(ctx, "/containers/"+id+"/start")
	if err != nil {
		return fmt.Errorf("container start: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("container start: %s", string(body))
	}
	return nil
}

// ContainerStop stops a container.
func (c *Client) ContainerStop(ctx context.Context, id string) error {
	resp, err := c.post(ctx, "/containers/"+id+"/stop")
	if err != nil {
		return fmt.Errorf("container stop: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("container stop: %s", string(body))
	}
	return nil
}

// ContainerRestart restarts a container.
func (c *Client) ContainerRestart(ctx context.Context, id string) error {
	resp, err := c.post(ctx, "/containers/"+id+"/restart")
	if err != nil {
		return fmt.Errorf("container restart: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("container restart: %s", string(body))
	}
	return nil
}

// ContainerRemove removes a container (force=true).
func (c *Client) ContainerRemove(ctx context.Context, id string) error {
	resp, err := c.delete(ctx, "/containers/"+id+"?force=1")
	if err != nil {
		return fmt.Errorf("container remove: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("container remove: %s", string(body))
	}
	return nil
}

// ListVolumes returns all Docker volumes.
func (c *Client) ListVolumes(ctx context.Context) ([]domain.Volume, error) {
	resp, err := c.get(ctx, "/volumes")
	if err != nil {
		return nil, fmt.Errorf("list volumes: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Volumes []struct {
			Name       string            `json:"Name"`
			Driver     string            `json:"Driver"`
			Mountpoint string            `json:"Mountpoint"`
			Labels     map[string]string `json:"Labels"`
			Scope      string            `json:"Scope"`
		} `json:"Volumes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("list volumes decode: %w", err)
	}

	vols := make([]domain.Volume, 0, len(result.Volumes))
	for _, v := range result.Volumes {
		vols = append(vols, domain.Volume{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			Labels:     v.Labels,
			Scope:      v.Scope,
		})
	}
	return vols, nil
}

// ListNetworks returns all Docker networks.
func (c *Client) ListNetworks(ctx context.Context) ([]domain.Network, error) {
	resp, err := c.get(ctx, "/networks")
	if err != nil {
		return nil, fmt.Errorf("list networks: %w", err)
	}
	defer resp.Body.Close()

	var raw []struct {
		ID     string            `json:"Id"`
		Name   string            `json:"Name"`
		Driver string            `json:"Driver"`
		Scope  string            `json:"Scope"`
		Labels map[string]string `json:"Labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("list networks decode: %w", err)
	}

	nets := make([]domain.Network, 0, len(raw))
	for _, n := range raw {
		nets = append(nets, domain.Network{
			ID:     n.ID,
			Name:   n.Name,
			Driver: n.Driver,
			Scope:  n.Scope,
			Labels: n.Labels,
		})
	}
	return nets, nil
}

// ListContexts returns Docker contexts using CLI.
func ListContexts() ([]domain.DockerContext, error) {
	// Implemented in compose package for symmetry; we return empty here.
	// Contexts are managed via CLI parsing.
	return nil, nil
}
