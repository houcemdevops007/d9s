// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
// Package app wires together all components and runs the main event loop.
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/houcemdevops007/d9s/internal/actions"
	"github.com/houcemdevops007/d9s/internal/compose"
	"github.com/houcemdevops007/d9s/internal/config"
	"github.com/houcemdevops007/d9s/internal/dockerapi"
	"github.com/houcemdevops007/d9s/internal/domain"
	"github.com/houcemdevops007/d9s/internal/scanners"
	"github.com/houcemdevops007/d9s/internal/store"
	"github.com/houcemdevops007/d9s/internal/tui"
)

// App is the top-level application object.
type App struct {
	cfg     *config.Config
	docker  *dockerapi.Client
	compose *compose.Runner
	actions *actions.Runner
	store   *store.Store
	term    *tui.Terminal
	view     *tui.View
	trivy    *scanners.TrivyScanner
	snyk     *scanners.SnykScanner
	bestRecs *scanners.BestPracticesEngine
	dockerHost string
}

// New creates a new App.
func New(cfg *config.Config, dockerHost string) *App {
	docker := dockerapi.New(dockerHost)
	comp := compose.New(cfg.DefaultContext, dockerHost)
	act := actions.New(docker, comp, dockerHost)
	st := store.New()

	return &App{
		cfg:     cfg,
		docker:  docker,
		compose: comp,
		actions: act,
		store:   st,
		term:     tui.NewTerminal(),
		trivy:    scanners.NewTrivyScanner(dockerHost),
		snyk:     scanners.NewSnykScanner(dockerHost),
		bestRecs: scanners.NewBestPracticesEngine(),
		dockerHost: dockerHost,
	}
}

// Run starts the application and blocks until the user quits.
func (a *App) Run() error {
	if err := a.term.SetRaw(); err != nil {
		return fmt.Errorf("set raw terminal: %w", err)
	}
	defer a.term.Restore()

	w, h := a.term.Size()
	theme := tui.DefaultTheme()
	a.view = tui.NewView(w, h, theme)

	fmt.Print("\x1b[?1049h")
	defer fmt.Print("\x1b[?1049l" + tui.ShowCursor())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go a.loadInitialData(ctx)
	go a.startEventWatcher(ctx)
	go a.startStatsPoller(ctx)
	go a.startRefreshTicker(ctx)

	resizeCh := tui.NotifyResize()
	keyCh := make(chan tui.Key, 10)
	go func() {
		for {
			k := tui.ReadKey()
			keyCh <- k
		}
	}()

	storeCh := a.store.Subscribe()
	a.render()

	for {
		select {
		case <-resizeCh:
			w, h = a.term.Size()
			a.view.Resize(w, h)
			a.render()

		case <-storeCh:
			st := a.store.Snapshot()
			a.view.UpdateFromStore(st)
			a.render()

		case k := <-keyCh:
			if a.handleKey(ctx, k) {
				return nil
			}
			a.render()
		}
	}
}

func (a *App) render() {
	os.Stdout.WriteString(a.view.Render())
}

func (a *App) handleKey(ctx context.Context, k tui.Key) bool {
	if a.view.ConfirmActive() {
		switch {
		case k.Rune == 'y' || k.Rune == 'Y':
			a.view.HideConfirm()
		case k.Rune == 'n' || k.Rune == 'N' || (k.IsKey && k.Code == tui.KeyEsc):
			a.view.HideConfirm()
			a.view.SetStatus("Action cancelled.", false)
		}
		return false
	}

	if a.view.ShowHelp() {
		a.view.ToggleHelp()
		return false
	}

	if a.view.IsSearching() {
		switch {
		case k.IsKey && k.Code == tui.KeyEnter:
			a.view.CommitSearch()
		case k.IsKey && k.Code == tui.KeyEsc:
			a.view.ClearSearch()
		case k.IsKey && k.Code == tui.KeyBackspace:
			a.view.BackspaceSearch()
		case k.Rune != 0:
			a.view.AppendSearch(k.Rune)
		}
		return false
	}

	switch {
	case k.IsKey && k.Code == tui.KeyEsc:
		a.view.SetActivePanel(tui.PanelContainers)
		a.view.SetActiveTab(tui.TabLogs)
	case k.IsKey && k.Code == tui.KeyUp:
		a.view.MoveUp()
	case k.IsKey && k.Code == tui.KeyDown:
		a.view.MoveDown()
	case k.IsKey && k.Code == tui.KeyLeft:
		a.view.PrevTab()
	case k.IsKey && k.Code == tui.KeyRight:
		a.view.NextTab()
	case k.IsKey && k.Code == tui.KeyTab:
		a.view.TabNext()
	case k.Rune == 'q' || (k.IsKey && (k.Code == tui.KeyCtrlC || k.Code == tui.KeyCtrlD)):
		return true
	case k.Rune == '/':
		a.view.StartSearch()
	case k.Rune == '?':
		a.view.ToggleHelp()
	case k.Rune == 'k':
		a.view.ScrollDetail(-3)
	case k.Rune == 'j':
		a.view.ScrollDetail(3)
	case k.IsKey && k.Code == tui.KeyCtrlL:
		go a.loadInitialData(ctx)
	case k.Rune == 'l':
		a.view.SetActiveTab(tui.TabLogs)
		go a.loadLogs(ctx)
	case k.Rune == 'e':
		a.view.SetActiveTab(tui.TabEvents)
	case k.Rune == 's' && a.view.ActivePanel() != tui.PanelImages:
		a.view.SetActiveTab(tui.TabStats)
		go a.pollStats(ctx)
	case k.Rune == 'i':
		a.view.SetActiveTab(tui.TabInspect)
		go a.loadInspect(ctx)
	case k.IsKey && k.Code == tui.KeyEnter:
		pan := a.view.ActivePanel()
		tab := a.view.ActiveTab()
		if pan == tui.PanelContexts {
			go a.switchContext(ctx)
		} else if pan == tui.PanelContainers {
			a.view.SetActiveTab(tui.TabLogs)
			go a.loadLogs(ctx)
		} else if pan == tui.PanelVolumes || pan == tui.PanelNetworks {
			a.view.SetActiveTab(tui.TabInspect)
			go a.loadInspect(ctx)
		} else if pan == tui.PanelImages {
			if tab == tui.TabTrivy || tab == tui.TabSnyk || tab == tui.TabRecommendations {
				// do nothing
			} else {
				a.view.SetActiveTab(tui.TabInspect)
				go a.loadInspect(ctx)
			}
		}
	case k.Rune == 'r':
		go a.containerAction(ctx, "restart")
	case k.Rune == 'x':
		go a.containerAction(ctx, "stop")
	case k.Rune == 'R' || (k.IsKey && k.Code == tui.KeyDelete):
		go a.containerAction(ctx, "remove")
	case k.Rune == 'S':
		a.execShell() // synchronous block
	case k.Rune == 'u':
		go a.composeAction(ctx, "up")
	case k.Rune == 'd':
		go a.composeAction(ctx, "down")
	case k.Rune == 'p':
		go a.composeAction(ctx, "pull")
	case k.Rune == 'b':
		go a.composeAction(ctx, "build")
	case k.Rune == 'c':
		a.view.SetActivePanel(tui.PanelContainers)
	case k.Rune == 'g' || k.Rune == 'i':
		a.view.SetActivePanel(tui.PanelImages)
	case k.Rune == 'v':
		a.view.SetActivePanel(tui.PanelVolumes)
	case k.Rune == 'n':
		a.view.SetActivePanel(tui.PanelNetworks)
	case k.Rune == 's' && a.view.ActivePanel() == tui.PanelImages:
		tab := a.view.ActiveTab()
		if tab != tui.TabTrivy && tab != tui.TabSnyk && tab != tui.TabRecommendations {
			a.view.SetActiveTab(tui.TabTrivy)
		}
		go a.scanImage(ctx)
	}
	return false
}

func (a *App) loadInitialData(ctx context.Context) {
	ctxs, err := compose.ListContexts()
	
	// If connecting remotely via TCP or non-default socket, prepend it as the active context
	if a.dockerHost != "" {
		remoteCtx := domain.DockerContext{
			Name:        a.dockerHost,
			Description: "Remote Host (DOCKER_HOST)",
			Endpoint:    a.dockerHost,
			Current:     true,
		}
		// Unmark others
		for i := range ctxs {
			ctxs[i].Current = false
		}
		ctxs = append([]domain.DockerContext{remoteCtx}, ctxs...)
	}

	// Add configured hosts
	for _, h := range a.cfg.Hosts {
		if h == a.dockerHost {
			continue // Already added
		}
		exists := false
		for _, c := range ctxs {
			if c.Endpoint == h || c.Name == h {
				exists = true
				break
			}
		}
		if !exists {
			ctxs = append(ctxs, domain.DockerContext{
				Name:        h,
				Description: "Configured Remote Host",
				Endpoint:    h,
				Current:     false,
			})
		}
	}

	if err == nil || len(ctxs) > 0 {
		a.store.SetContexts(ctxs)
		for _, c := range ctxs {
			if c.Current {
				a.store.SetActiveContext(c.Name)
			}
		}
	}

	containers, err := a.docker.ListContainers(ctx, true)
	if err != nil {
		a.store.SetError("Docker: " + err.Error())
		return
	}
	a.store.SetContainers(containers)
	a.store.SetError("")

	projects, err := a.compose.Projects(ctx)
	if err == nil {
		for i, p := range projects {
			svcs, serr := a.compose.ServiceContainers(ctx, p.WorkingDir)
			if serr == nil {
				projects[i].Services = svcs
			}
		}
		a.store.SetProjects(projects)
	}

	images, err := a.docker.ListImages(ctx)
	if err == nil {
		a.store.SetImages(images)
	}

	vols, err := a.docker.ListVolumes(ctx)
	if err == nil {
		a.store.SetVolumes(vols)
	}
	nets, err := a.docker.ListNetworks(ctx)
	if err == nil {
		a.store.SetNetworks(nets)
	}
}

func (a *App) startEventWatcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		ch, err := a.docker.Events(ctx)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		for ev := range ch {
			a.store.AddEvent(ev)
			if ev.Type == "container" {
				if containers, err := a.docker.ListContainers(ctx, true); err == nil {
					a.store.SetContainers(containers)
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func (a *App) startStatsPoller(ctx context.Context) {
	ticker := time.NewTicker(a.cfg.StatsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.pollStats(ctx)
		}
	}
}

func (a *App) startRefreshTicker(ctx context.Context) {
	ticker := time.NewTicker(a.cfg.RefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.loadInitialData(ctx)
		}
	}
}

func (a *App) pollStats(ctx context.Context) {
	c := a.view.ActiveContainer()
	if c == nil {
		return
	}
	if stats, err := a.docker.Stats(ctx, c.ID); err == nil {
		a.store.SetStats(c.ID, stats)
	}
}

func (a *App) loadLogs(ctx context.Context) {
	c := a.view.ActiveContainer()
	if c == nil {
		a.view.SetStatus("No container selected", true)
		return
	}
	a.view.SetStatus("Loading logs for "+c.ShortName()+"...", false)

	ch, err := a.docker.Logs(ctx, c.ID, a.cfg.LogTailLines)
	if err != nil {
		a.view.SetStatus("Logs error: "+err.Error(), true)
		return
	}

	var lines []string
	timeout := time.After(3 * time.Second)
	for {
		select {
		case line, ok := <-ch:
			if !ok {
				goto done
			}
			lines = append(lines, line.Text)
			if len(lines) >= a.cfg.LogTailLines {
				goto done
			}
		case <-timeout:
			goto done
		case <-ctx.Done():
			return
		}
	}
done:
	a.view.SetLogs(lines)
	a.view.ClearStatus()
	a.render()
}

func (a *App) loadInspect(ctx context.Context) {
	var data interface{}
	var err error

	switch a.view.ActivePanel() {
	case tui.PanelContainers:
		c := a.view.ActiveContainer()
		if c == nil {
			a.view.SetStatus("No container selected", true)
			return
		}
		data, err = a.docker.InspectContainer(ctx, c.ID)
	case tui.PanelImages:
		img := a.view.ActiveImage()
		if img == nil {
			a.view.SetStatus("No image selected", true)
			return
		}
		data, err = a.docker.InspectImage(ctx, img.ID)
	case tui.PanelVolumes:
		vol := a.view.ActiveVolume()
		if vol == nil {
			a.view.SetStatus("No volume selected", true)
			return
		}
		data, err = a.docker.InspectVolume(ctx, vol.Name)
	case tui.PanelNetworks:
		net := a.view.ActiveNetwork()
		if net == nil {
			a.view.SetStatus("No network selected", true)
			return
		}
		data, err = a.docker.InspectNetwork(ctx, net.ID)
	default:
		a.view.SetStatus("Nothing to inspect in this panel", true)
		return
	}

	if err != nil {
		a.view.SetStatus("Inspect error: "+err.Error(), true)
		return
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		a.view.SetStatus("Inspect encode error: "+err.Error(), true)
		return
	}
	a.view.SetInspect(string(b))
	a.view.ClearStatus()
	a.render()
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	if s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-32) + s[1:]
	}
	return s
}

func (a *App) containerAction(ctx context.Context, action string) {
	c := a.view.ActiveContainer()
	if c == nil {
		a.view.SetStatus("No container selected", true)
		return
	}
	name := c.ShortName()
	a.view.SetStatus(fmt.Sprintf("%s %s...", capitalize(action), name), false)
	a.render()

	var err error
	switch action {
	case "restart":
		err = a.actions.ContainerRestart(ctx, c.ID)
	case "stop":
		err = a.actions.ContainerStop(ctx, c.ID)
	case "remove":
		err = a.actions.ContainerRemove(ctx, c.ID)
	}

	if err != nil {
		a.view.SetStatus(fmt.Sprintf("%s failed: %s", action, err.Error()), true)
	} else {
		a.view.SetStatus(fmt.Sprintf("%s %s: OK", capitalize(action), name), false)
		if containers, err := a.docker.ListContainers(ctx, true); err == nil {
			a.store.SetContainers(containers)
		}
	}
	a.render()
}

func (a *App) composeAction(ctx context.Context, action string) {
	p := a.view.ActiveProject()
	if p == nil {
		a.view.SetStatus("No project selected", true)
		return
	}
	a.view.SetStatus(fmt.Sprintf("compose %s %s...", action, p.Name), false)
	a.render()

	var err error
	switch action {
	case "up":
		err = a.actions.ComposeUp(ctx, p.WorkingDir)
	case "down":
		err = a.actions.ComposeDown(ctx, p.WorkingDir)
	case "pull":
		err = a.actions.ComposePull(ctx, p.WorkingDir)
	case "build":
		err = a.actions.ComposeBuild(ctx, p.WorkingDir)
	}

	if err != nil {
		a.view.SetStatus(fmt.Sprintf("compose %s failed: %s", action, err.Error()), true)
	} else {
		a.view.SetStatus(fmt.Sprintf("compose %s %s: OK", action, p.Name), false)
	}
	a.render()
}

func (a *App) execShell() {
	c := a.view.ActiveContainer()
	if c == nil {
		a.view.SetStatus("No container selected", true)
		return
	}
	a.term.Restore()
	fmt.Print("\x1b[?1049l" + tui.ShowCursor())

	activeCtx := ""
	if cCtx := a.view.ActiveContext(); cCtx != nil {
		activeCtx = cCtx.Name
	}
	err := a.actions.ExecShell(c.ID, activeCtx)

	fmt.Print("\x1b[?1049h" + tui.HideCursor())
	a.term.SetRaw() //nolint

	if err != nil {
		a.view.SetStatus("Shell error: "+err.Error(), true)
	}
	a.render()
}

func (a *App) scanImage(ctx context.Context) {
	img := a.view.ActiveImage()
	if img == nil {
		a.view.SetStatus("No image selected", true)
		return
	}

	a.view.SetStatus("Scanning image "+img.Repository+":"+img.Tag+"...", false)
	a.store.SetScanInProgress(img.ID, true)
	a.render()

	// 1. Run Trivy scan
	tResult, tErr := a.trivy.ScanImage(ctx, img.ID)
	if tErr != nil {
		a.store.SetScanningError(img.ID, "Trivy", "Trivy: "+tErr.Error())
	} else {
		a.store.SetSecurityResult(img.ID, "Trivy", tResult)
	}

	// 2. Run Snyk scan
	sResult, sErr := a.snyk.ScanImage(ctx, img.ID)
	if sErr != nil {
		a.store.SetScanningError(img.ID, "Snyk", "Snyk: "+sErr.Error())
	} else {
		a.store.SetSecurityResult(img.ID, "Snyk", sResult)
	}

	// 3. Best Practices (correlate with Trivy by default or combined)
	// For simplicity, we use the Trivy result for best practices analysis if available.
	if tErr == nil {
		details, derr := a.docker.InspectImage(ctx, img.ID)
		if derr == nil {
			recs := a.bestRecs.Analyze(details, tResult)
			a.store.SetRecommendations(img.ID, recs)
		}
	}

	a.view.SetStatus("Scan & analysis complete for "+img.Repository, false)
	a.render()
}

func (a *App) switchContext(ctx context.Context) {
	c := a.view.ActiveContext()
	if c == nil {
		return
	}

	a.view.SetStatus(fmt.Sprintf("Switching context to %s...", c.Name), false)
	a.render()

	if strings.HasPrefix(c.Name, "tcp://") || strings.HasPrefix(c.Name, "unix://") || strings.HasPrefix(c.Name, "http://") || strings.HasPrefix(c.Name, "https://") {
		// Custom remote host
		a.dockerHost = c.Name
		a.docker = dockerapi.New(a.dockerHost)
		a.compose = compose.New(a.cfg.DefaultContext, a.dockerHost)
		a.actions = actions.New(a.docker, a.compose, a.dockerHost)
		a.trivy = scanners.NewTrivyScanner(a.dockerHost)
		a.snyk = scanners.NewSnykScanner(a.dockerHost)
	} else {
		// Native docker context
		if err := compose.SwitchContext(c.Name); err != nil {
			a.view.SetStatus(fmt.Sprintf("Failed to switch context: %s", err), true)
			a.render()
			return
		}
		a.dockerHost = ""
		a.docker = dockerapi.New("")
		a.compose = compose.New(c.Name, "")
		a.actions = actions.New(a.docker, a.compose, "")
		a.trivy = scanners.NewTrivyScanner("")
		a.snyk = scanners.NewSnykScanner("")
	}

	// Clear local state
	a.store.SetContainers(nil)
	a.store.SetImages(nil)
	a.store.SetVolumes(nil)
	a.store.SetNetworks(nil)
	a.store.SetProjects(nil)
	
	// Reload
	a.loadInitialData(ctx)
	a.view.SetStatus(fmt.Sprintf("Switched to %s", c.Name), false)
	a.render()
}
