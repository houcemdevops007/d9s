// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
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
	Contexts         []domain.DockerContext
	Projects         []domain.ComposeProject
	Containers       []domain.Container
	Volumes          []domain.Volume
	Networks         []domain.Network
	Images           []domain.Image
	Events           []domain.RuntimeEvent
	Stats            map[string]domain.RuntimeStats // keyed by container ID
	SecurityResults  map[string]map[string]domain.SecurityScanResult // keyed by [imageID][scanner]
	ScanInProgress   map[string]bool                // keyed by image ID
	ScanningErrors   map[string]string              // keyed by image ID
	Recommendations  map[string][]domain.BestPracticeRecommendation // keyed by image ID
	LastRefreshed    time.Time
	ActiveContext    string
	ActiveProject    string
	Error            string
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
			Stats:           make(map[string]domain.RuntimeStats),
			SecurityResults: make(map[string]map[string]domain.SecurityScanResult),
			ScanInProgress:  make(map[string]bool),
			ScanningErrors:  make(map[string]string),
			Recommendations: make(map[string][]domain.BestPracticeRecommendation),
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
	cp.SecurityResults = make(map[string]map[string]domain.SecurityScanResult, len(s.state.SecurityResults))
	for imgID, scans := range s.state.SecurityResults {
		cp.SecurityResults[imgID] = make(map[string]domain.SecurityScanResult, len(scans))
		for scanner, res := range scans {
			cp.SecurityResults[imgID][scanner] = res
		}
	}
	cp.ScanInProgress = make(map[string]bool, len(s.state.ScanInProgress))
	for k, v := range s.state.ScanInProgress {
		cp.ScanInProgress[k] = v
	}
	cp.ScanningErrors = make(map[string]string, len(s.state.ScanningErrors))
	for k, v := range s.state.ScanningErrors {
		cp.ScanningErrors[k] = v
	}
	cp.Recommendations = make(map[string][]domain.BestPracticeRecommendation, len(s.state.Recommendations))
	for k, v := range s.state.Recommendations {
		cp.Recommendations[k] = v
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

// SetImages replaces the images list.
func (s *Store) SetImages(images []domain.Image) {
	s.mu.Lock()
	s.state.Images = images
	s.mu.Unlock()
	s.notify()
}

// SetScanInProgress updates the scan status for an image.
func (s *Store) SetScanInProgress(imageID string, inProgress bool) {
	s.mu.Lock()
	if s.state.ScanInProgress == nil {
		s.state.ScanInProgress = make(map[string]bool)
	}
	s.state.ScanInProgress[imageID] = inProgress
	if inProgress {
		delete(s.state.ScanningErrors, imageID)
	}
	s.mu.Unlock()
	s.notify()
}

// SetSecurityResult updates the scan result for an image and scanner.
func (s *Store) SetSecurityResult(imageID, scanner string, res domain.SecurityScanResult) {
	s.mu.Lock()
	if s.state.SecurityResults == nil {
		s.state.SecurityResults = make(map[string]map[string]domain.SecurityScanResult)
	}
	if s.state.SecurityResults[imageID] == nil {
		s.state.SecurityResults[imageID] = make(map[string]domain.SecurityScanResult)
	}
	s.state.SecurityResults[imageID][scanner] = res
	s.state.ScanInProgress[imageID] = false
	s.mu.Unlock()
	s.notify()
}

// SetScanningError records an error during scan.
func (s *Store) SetScanningError(imageID string, err string) {
	s.mu.Lock()
	if s.state.ScanningErrors == nil {
		s.state.ScanningErrors = make(map[string]string)
	}
	s.state.ScanningErrors[imageID] = err
	s.state.ScanInProgress[imageID] = false
	s.mu.Unlock()
	s.notify()
}

// SetRecommendations updates recommendations for an image.
func (s *Store) SetRecommendations(imageID string, recs []domain.BestPracticeRecommendation) {
	s.mu.Lock()
	if s.state.Recommendations == nil {
		s.state.Recommendations = make(map[string][]domain.BestPracticeRecommendation)
	}
	s.state.Recommendations[imageID] = recs
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
