// Package app wires together all components and runs the main event loop.
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/houcemdevops007/d9s/internal/actions"
	"github.com/houcemdevops007/d9s/internal/compose"
	"github.com/houcemdevops007/d9s/internal/config"
	"github.com/houcemdevops007/d9s/internal/dockerapi"
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
	view    *tui.View
}

// New creates a new App.
func New(cfg *config.Config) *App {
	docker := dockerapi.New("")
	comp := compose.New(cfg.DefaultContext)
	act := actions.New(docker, comp)
	st := store.New()

	return &App{
		cfg:     cfg,
		docker:  docker,
		compose: comp,
		actions: act,
		store:   st,
		term:    tui.NewTerminal(),
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
	case k.IsKey && k.Code == tui.KeyUp:
		a.view.MoveUp()
	case k.IsKey && k.Code == tui.KeyDown:
		a.view.MoveDown()
	case k.IsKey && k.Code == tui.KeyTab:
		a.view.TabNext()
	case k.Rune == 'q' || (k.IsKey && (k.Code == tui.KeyCtrlC || k.Code == tui.KeyCtrlD)):
		return true
	case k.Rune == '/':
		a.view.StartSearch()
	case k.Rune == '?':
		a.view.ToggleHelp()
	case k.IsKey && k.Code == tui.KeyCtrlL:
		go a.loadInitialData(ctx)
	case k.Rune == 'l':
		a.view.SetActiveTab(tui.TabLogs)
		go a.loadLogs(ctx)
	case k.Rune == 'e':
		a.view.SetActiveTab(tui.TabEvents)
	case k.Rune == 's':
		a.view.SetActiveTab(tui.TabStats)
		go a.pollStats(ctx)
	case k.Rune == 'i':
		a.view.SetActiveTab(tui.TabInspect)
		go a.loadInspect(ctx)
	case k.Rune == 'r':
		go a.containerAction(ctx, "restart")
	case k.Rune == 'x':
		go a.containerAction(ctx, "stop")
	case k.Rune == 'R' || (k.IsKey && k.Code == tui.KeyDelete):
		go a.containerAction(ctx, "remove")
	case k.Rune == 'S':
		go a.execShell()
	case k.Rune == 'u':
		go a.composeAction(ctx, "up")
	case k.Rune == 'd':
		go a.composeAction(ctx, "down")
	case k.Rune == 'p':
		go a.composeAction(ctx, "pull")
	case k.Rune == 'b':
		go a.composeAction(ctx, "build")
	case k.Rune == 'c':
		a.view.TabNext()
	}
	return false
}

func (a *App) loadInitialData(ctx context.Context) {
	ctxs, err := compose.ListContexts()
	if err == nil {
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
	c := a.view.ActiveContainer()
	if c == nil {
		a.view.SetStatus("No container selected", true)
		return
	}
	data, err := a.docker.InspectContainer(ctx, c.ID)
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

	err := a.actions.ExecShell(c.ID, a.cfg.DefaultContext)

	fmt.Print("\x1b[?1049h")
	a.term.SetRaw() //nolint

	if err != nil {
		a.view.SetStatus("Shell error: "+err.Error(), true)
	}
	a.render()
}
