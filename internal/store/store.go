// Package store manages the central application state for d9s.
// It holds snapshots of Docker and Compose state and merges them
// into a unified view for the UI.
package store

import (
	"sync"
	"time"

	"github.com/houcemdevops007/d9s/internal/domain"
)

// State holds the full application state snapshot.
type State struct {
	Contexts       []domain.DockerContext
	Projects       []domain.ComposeProject
	Containers     []domain.Container
	Volumes        []domain.Volume
	Networks       []domain.Network
	Events         []domain.RuntimeEvent
	Stats          map[string]domain.RuntimeStats // keyed by container ID
	LastRefreshed  time.Time
	ActiveContext  string
	ActiveProject  string
	Error          string
}

// Store is the central state manager.
type Store struct {
	mu     sync.RWMutex
	state  State
	subs   []chan struct{} // notify subscribers on state change
}

// New creates an empty Store.
func New() *Store {
	return &Store{
		state: State{
			Stats: make(map[string]domain.RuntimeStats),
		},
	}
}

// Subscribe returns a channel that receives a signal (struct{}) when state changes.
func (s *Store) Subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	s.subs = append(s.subs, ch)
	s.mu.Unlock()
	return ch
}

func (s *Store) notify() {
	for _, sub := range s.subs {
		select {
		case sub <- struct{}{}:
		default:
		}
	}
}

// Snapshot returns a read-only copy of the current state.
func (s *Store) Snapshot() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Shallow copy — callers should not mutate slices.
	cp := s.state
	cp.Stats = make(map[string]domain.RuntimeStats, len(s.state.Stats))
	for k, v := range s.state.Stats {
		cp.Stats[k] = v
	}
	return cp
}

// SetContainers replaces the containers list.
func (s *Store) SetContainers(containers []domain.Container) {
	s.mu.Lock()
	s.state.Containers = containers
	s.state.LastRefreshed = time.Now()
	s.mu.Unlock()
	s.notify()
}

// SetContexts replaces the contexts list.
func (s *Store) SetContexts(contexts []domain.DockerContext) {
	s.mu.Lock()
	s.state.Contexts = contexts
	s.mu.Unlock()
	s.notify()
}

// SetProjects replaces the compose projects list (with services merged in).
func (s *Store) SetProjects(projects []domain.ComposeProject) {
	s.mu.Lock()
	s.state.Projects = projects
	s.mu.Unlock()
	s.notify()
}

// SetActiveContext updates the active Docker context name.
func (s *Store) SetActiveContext(name string) {
	s.mu.Lock()
	s.state.ActiveContext = name
	s.mu.Unlock()
	s.notify()
}

// SetActiveProject updates the active Compose project name.
func (s *Store) SetActiveProject(name string) {
	s.mu.Lock()
	s.state.ActiveProject = name
	s.mu.Unlock()
	s.notify()
}

// SetError records an error message for display.
func (s *Store) SetError(msg string) {
	s.mu.Lock()
	s.state.Error = msg
	s.mu.Unlock()
	s.notify()
}

// AddEvent appends a runtime event, capping at 500 entries.
func (s *Store) AddEvent(ev domain.RuntimeEvent) {
	s.mu.Lock()
	s.state.Events = append(s.state.Events, ev)
	if len(s.state.Events) > 500 {
		s.state.Events = s.state.Events[len(s.state.Events)-500:]
	}
	s.mu.Unlock()
	s.notify()
}

// SetStats updates stats for a specific container.
func (s *Store) SetStats(id string, stats domain.RuntimeStats) {
	s.mu.Lock()
	if s.state.Stats == nil {
		s.state.Stats = make(map[string]domain.RuntimeStats)
	}
	s.state.Stats[id] = stats
	s.mu.Unlock()
	s.notify()
}

// SetVolumes replaces the volumes list.
func (s *Store) SetVolumes(vols []domain.Volume) {
	s.mu.Lock()
	s.state.Volumes = vols
	s.mu.Unlock()
	s.notify()
}

// SetNetworks replaces the networks list.
func (s *Store) SetNetworks(nets []domain.Network) {
	s.mu.Lock()
	s.state.Networks = nets
	s.mu.Unlock()
	s.notify()
}

// ContainersForProject returns containers belonging to a compose project.
func (s *Store) ContainersForProject(projectName string) []domain.Container {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []domain.Container
	for _, c := range s.state.Containers {
		if c.ComposeProject == projectName {
			result = append(result, c)
		}
	}
	return result
}

// ContainersForService returns containers for a given service in a project.
func (s *Store) ContainersForService(projectName, service string) []domain.Container {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []domain.Container
	for _, c := range s.state.Containers {
		if c.ComposeProject == projectName && c.ComposeService == service {
			result = append(result, c)
		}
	}
	return result
}

// FilterContainers returns containers matching a search string (name or image).
func (s *Store) FilterContainers(query string) []domain.Container {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if query == "" {
		cp := make([]domain.Container, len(s.state.Containers))
		copy(cp, s.state.Containers)
		return cp
	}
	var result []domain.Container
	for _, c := range s.state.Containers {
		if contains(c.Name, query) || contains(c.Image, query) || contains(c.ComposeProject, query) {
			result = append(result, c)
		}
	}
	return result
}

func contains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	return len(s) >= len(sub) && indexCI(s, sub) >= 0
}

func indexCI(s, sub string) int {
	sl := toLower(s)
	subl := toLower(sub)
	for i := 0; i <= len(sl)-len(subl); i++ {
		if sl[i:i+len(subl)] == subl {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
