// Package domain defines the core business entities for d9s.
package domain

import "time"

// DockerContext represents a Docker context (local or remote).
type DockerContext struct {
	Name        string
	Description string
	Endpoint    string
	Current     bool
}

// ComposeProject represents a Docker Compose project.
type ComposeProject struct {
	Name        string
	ConfigFiles []string
	WorkingDir  string
	Status      string // running, exited, paused, etc.
	Services    []ComposeService
}

// ComposeService represents a service within a Compose project.
type ComposeService struct {
	Name        string
	Project     string
	Image       string
	State       string
	Replicas    int
	Containers  []Container
}

// Container represents a Docker container.
type Container struct {
	ID          string
	ShortID     string
	Name        string
	Image       string
	ImageID     string
	Command     string
	State       string // running, exited, paused, created, restarting
	Status      string // "Up 5 hours", "Exited (0) 2 hours ago"
	Created     int64
	Ports       []Port
	Labels      map[string]string
	// Compose metadata (from labels or compose ps)
	ComposeProject string
	ComposeService string
	// Network info
	NetworkMode string
	Networks    map[string]ContainerNetwork
}

// ShortName returns the container name without the leading slash.
func (c Container) ShortName() string {
	if len(c.Name) > 0 && c.Name[0] == '/' {
		return c.Name[1:]
	}
	return c.Name
}

// IsRunning returns true if the container is in running state.
func (c Container) IsRunning() bool {
	return c.State == "running"
}

// Port represents a container port mapping.
type Port struct {
	IP          string
	PrivatePort uint16
	PublicPort  uint16
	Type        string
}

// ContainerNetwork holds per-network info for a container.
type ContainerNetwork struct {
	IPAddress string
	Gateway   string
}

// Volume represents a Docker volume.
type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
	Labels     map[string]string
	Scope      string
}

// Network represents a Docker network.
type Network struct {
	ID     string
	Name   string
	Driver string
	Scope  string
	Labels map[string]string
}

// RuntimeEvent is a Docker daemon event.
type RuntimeEvent struct {
	Time    time.Time
	Type    string // container, network, volume, image
	Action  string // start, stop, die, create, destroy, etc.
	Actor   string // ID or name of the actor
	Message string
}

// RuntimeStats holds CPU and memory stats for a container.
type RuntimeStats struct {
	ContainerID string
	Name        string
	CPUPercent  float64
	MemUsage    uint64 // bytes
	MemLimit    uint64 // bytes
	MemPercent  float64
	PidsCount   uint64
	Timestamp   time.Time
}

// LogLine represents a single log line from a container.
type LogLine struct {
	ContainerID string
	Timestamp   time.Time
	Stream      string // stdout or stderr
	Text        string
}
