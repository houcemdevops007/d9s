package tui

import (
	"fmt"
	"strings"

	"github.com/houcemdevops007/d9s/internal/domain"
	"github.com/houcemdevops007/d9s/internal/store"
)

// Panel represents a named display panel.
type Panel int

const (
	PanelContainers Panel = iota
	PanelProjects
	PanelContexts
)

// DetailTab represents the active detail tab.
type DetailTab int

const (
	TabLogs DetailTab = iota
	TabEvents
	TabStats
	TabInspect
)

// View holds the full UI rendering state.
type View struct {
	theme          Theme
	width          int
	height         int
	activePanel    Panel
	activeTab      DetailTab
	containers     []domain.Container
	containerIdx   int
	containerFilter string
	projects       []domain.ComposeProject
	projectIdx     int
	contexts       []domain.DockerContext
	contextIdx     int
	logs           []string
	events         []domain.RuntimeEvent
	stats          map[string]domain.RuntimeStats
	inspect        string
	searching      bool
	searchBuf      string
	statusMsg      string
	statusErr      bool
	showHelpFlag   bool
	confirmMsg     string
	confirmActive  bool
}

// NewView creates a new View.
func NewView(width, height int, theme Theme) *View {
	return &View{
		theme:  theme,
		width:  width,
		height: height,
		stats:  make(map[string]domain.RuntimeStats),
	}
}

// UpdateFromStore refreshes view data from the store snapshot.
func (v *View) UpdateFromStore(st store.State) {
	v.containers = st.Containers
	v.projects = st.Projects
	v.contexts = st.Contexts
	v.events = st.Events
	v.stats = st.Stats
	if v.containerIdx >= len(v.containers) && len(v.containers) > 0 {
		v.containerIdx = len(v.containers) - 1
	}
	if v.projectIdx >= len(v.projects) && len(v.projects) > 0 {
		v.projectIdx = len(v.projects) - 1
	}
	if v.contextIdx >= len(v.contexts) && len(v.contexts) > 0 {
		v.contextIdx = len(v.contexts) - 1
	}
}

// SetInspect sets the inspect panel content.
func (v *View) SetInspect(s string) { v.inspect = s }

// SetLogs sets the log lines.
func (v *View) SetLogs(lines []string) { v.logs = lines }

// ActiveContainer returns the currently selected container or nil.
func (v *View) ActiveContainer() *domain.Container {
	filtered := v.filteredContainers()
	if len(filtered) == 0 {
		return nil
	}
	if v.containerIdx >= len(filtered) {
		v.containerIdx = len(filtered) - 1
	}
	c := filtered[v.containerIdx]
	return &c
}

// ActiveProject returns the currently selected project or nil.
func (v *View) ActiveProject() *domain.ComposeProject {
	if len(v.projects) == 0 {
		return nil
	}
	p := v.projects[v.projectIdx]
	return &p
}

// ActiveContext returns the currently selected context or nil.
func (v *View) ActiveContext() *domain.DockerContext {
	if len(v.contexts) == 0 {
		return nil
	}
	c := v.contexts[v.contextIdx]
	return &c
}

func (v *View) filteredContainers() []domain.Container {
	if v.searchBuf == "" {
		return v.containers
	}
	var out []domain.Container
	q := strings.ToLower(v.searchBuf)
	for _, c := range v.containers {
		if strings.Contains(strings.ToLower(c.Name), q) ||
			strings.Contains(strings.ToLower(c.Image), q) ||
			strings.Contains(strings.ToLower(c.ComposeProject), q) {
			out = append(out, c)
		}
	}
	return out
}

// Render builds the full screen render string.
func (v *View) Render() string {
	var b strings.Builder
	b.WriteString("\x1b[2J\x1b[H")
	b.WriteString(HideCursor())

	if v.showHelpFlag {
		v.renderHelp(&b)
		return b.String()
	}

	if v.confirmActive {
		v.renderConfirm(&b)
		return b.String()
	}

	v.renderHeader(&b)

	contentHeight := v.height - 3
	leftW := 24
	rightW := v.width - leftW - 3
	centerW := rightW / 2
	detailW := rightW - centerW

	v.renderLeftPanel(&b, leftW, contentHeight)
	v.renderCenterPanel(&b, leftW+2, centerW, contentHeight)
	v.renderDetailPanel(&b, leftW+2+centerW+1, detailW, contentHeight)
	v.renderColumnSeparators(&b, leftW, centerW, contentHeight)

	b.WriteString(MoveTo(v.height-1, 1))
	b.WriteString(v.theme.Muted)
	b.WriteString(HLine(v.width, "─"))
	b.WriteString(Reset)

	v.renderStatusBar(&b)
	return b.String()
}

func (v *View) renderHeader(b *strings.Builder) {
	t := v.theme
	b.WriteString(MoveTo(1, 1))
	b.WriteString(t.BgHeader)
	b.WriteString(t.TextHeader)
	header := Pad(" ⬡ d9s  Docker TUI", v.width)
	b.WriteString(header)
	b.WriteString(Reset)

	versionStr := "v0.1.0"
	b.WriteString(MoveTo(1, v.width-len(versionStr)-1))
	b.WriteString(t.BgHeader + t.Muted + versionStr + " " + Reset)

	b.WriteString(MoveTo(2, 1))
	b.WriteString(t.Primary)
	b.WriteString(HLine(v.width, "━"))
	b.WriteString(Reset)
}

func (v *View) renderLeftPanel(b *strings.Builder, width, height int) {
	t := v.theme
	row := 3

	b.WriteString(MoveTo(row, 1))
	b.WriteString(t.Primary + bold + Pad(" CONTEXTS", width) + Reset)
	row++

	for i, ctx := range v.contexts {
		b.WriteString(MoveTo(row, 1))
		if v.activePanel == PanelContexts && i == v.contextIdx {
			b.WriteString(t.BgSelected)
		}
		indicator := "  "
		if ctx.Current {
			indicator = t.Success + "✓ " + Reset
			if v.activePanel == PanelContexts && i == v.contextIdx {
				indicator = t.BgSelected + t.Success + "✓ " + Reset + t.BgSelected
			}
		}
		b.WriteString(indicator + Pad(ctx.Name, width-2) + Reset)
		row++
		if row >= 3+height/2 {
			break
		}
	}
	if len(v.contexts) == 0 {
		b.WriteString(MoveTo(row, 1))
		b.WriteString(t.Muted + "  (no contexts)" + Reset)
		row++
	}
	row++

	b.WriteString(MoveTo(row, 1))
	b.WriteString(t.Secondary + bold + Pad(" PROJECTS", width) + Reset)
	row++

	for i, p := range v.projects {
		if row >= 3+height {
			break
		}
		b.WriteString(MoveTo(row, 1))
		if v.activePanel == PanelProjects && i == v.projectIdx {
			b.WriteString(t.BgSelected)
		}
		statusIcon := "  "
		if strings.Contains(strings.ToLower(p.Status), "running") {
			statusIcon = t.Success + "● " + Reset
			if v.activePanel == PanelProjects && i == v.projectIdx {
				statusIcon = t.BgSelected + t.Success + "● " + Reset + t.BgSelected
			}
		}
		b.WriteString(statusIcon + Pad(p.Name, width-2) + Reset)
		row++
	}
	if len(v.projects) == 0 {
		b.WriteString(MoveTo(row, 1))
		b.WriteString(t.Muted + "  (no projects)" + Reset)
	}
}

func (v *View) renderCenterPanel(b *strings.Builder, startCol, width, height int) {
	t := v.theme
	row := 3

	title := "CONTAINERS"
	if v.searchBuf != "" {
		title = "CONTAINERS  /" + v.searchBuf + "_"
	}
	b.WriteString(MoveTo(row, startCol))
	b.WriteString(t.Primary + bold + Pad(" "+title, width) + Reset)
	row++

	b.WriteString(MoveTo(row, startCol))
	b.WriteString(t.Muted + Pad(fmt.Sprintf(" %-14s %-30s %-20s %-10s", "ID", "NAME", "IMAGE", "STATE"), width) + Reset)
	row++

	b.WriteString(MoveTo(row, startCol))
	b.WriteString(t.Muted + HLine(width, "─") + Reset)
	row++

	filtered := v.filteredContainers()
	if len(filtered) == 0 {
		b.WriteString(MoveTo(row, startCol))
		b.WriteString(t.Muted + "  No containers found" + Reset)
		return
	}

	maxRows := height - 4
	for i, c := range filtered {
		if i >= maxRows {
			break
		}
		b.WriteString(MoveTo(row+i, startCol))
		selected := v.activePanel == PanelContainers && i == v.containerIdx
		if selected {
			b.WriteString(t.BgSelected)
		}
		stateColor := t.StateColor(c.State)
		icon := StateIcon(c.State)
		shortName := c.ShortName()
		shortImg := c.Image
		if len(shortImg) > 20 {
			parts := strings.Split(shortImg, "/")
			shortImg = parts[len(parts)-1]
		}
		line := fmt.Sprintf(" %-14s %-30s %-20s %s%s%s",
			Pad(c.ShortID, 14),
			Pad(shortName, 30),
			Pad(shortImg, 20),
			stateColor, Pad(icon+" "+c.State, 10), Reset,
		)
		if selected {
			b.WriteString(line + t.BgSelected)
		} else {
			b.WriteString(line)
		}
		if c.ComposeProject != "" && width > 80 {
			b.WriteString(fmt.Sprintf("%s %s/%s%s", t.Muted, c.ComposeProject, c.ComposeService, Reset))
		}
		b.WriteString(Reset)
	}
}

func (v *View) renderDetailPanel(b *strings.Builder, startCol, width, height int) {
	t := v.theme
	tabs := []struct {
		tab   DetailTab
		label string
	}{
		{TabLogs, "Logs"},
		{TabEvents, "Events"},
		{TabStats, "Stats"},
		{TabInspect, "Inspect"},
	}

	b.WriteString(MoveTo(3, startCol))
	b.WriteString(" ")
	for _, tab := range tabs {
		if v.activeTab == tab.tab {
			b.WriteString(t.Primary + bold + "[" + tab.label + "]" + Reset + " ")
		} else {
			b.WriteString(t.Muted + " " + tab.label + "  " + Reset)
		}
	}

	b.WriteString(MoveTo(4, startCol))
	b.WriteString(t.Primary + HLine(width, "─") + Reset)

	contentStart := 5
	contentRows := height - 3

	switch v.activeTab {
	case TabLogs:
		v.renderLogsDetail(b, startCol, width, contentStart, contentRows)
	case TabEvents:
		v.renderEventsDetail(b, startCol, width, contentStart, contentRows)
	case TabStats:
		v.renderStatsDetail(b, startCol, width, contentStart, contentRows)
	case TabInspect:
		v.renderInspectDetail(b, startCol, width, contentStart, contentRows)
	}
}

func (v *View) renderLogsDetail(b *strings.Builder, col, width, startRow, rows int) {
	t := v.theme
	if len(v.logs) == 0 {
		b.WriteString(MoveTo(startRow, col))
		b.WriteString(t.Muted + " No logs. Select container, press 'l'." + Reset)
		return
	}
	lines := v.logs
	if len(lines) > rows {
		lines = lines[len(lines)-rows:]
	}
	for i, line := range lines {
		b.WriteString(MoveTo(startRow+i, col))
		runes := []rune(line)
		if len(runes) > width-1 {
			runes = runes[:width-1]
		}
		b.WriteString(" " + string(runes) + ClearLine())
	}
}

func (v *View) renderEventsDetail(b *strings.Builder, col, width, startRow, rows int) {
	t := v.theme
	if len(v.events) == 0 {
		b.WriteString(MoveTo(startRow, col))
		b.WriteString(t.Muted + " Waiting for events..." + Reset)
		return
	}
	events := v.events
	if len(events) > rows {
		events = events[len(events)-rows:]
	}
	for i, ev := range events {
		b.WriteString(MoveTo(startRow+i, col))
		timeStr := ev.Time.Format("15:04:05")
		var typeColor string
		switch ev.Type {
		case "container":
			typeColor = t.Primary
		case "network":
			typeColor = t.Secondary
		case "volume":
			typeColor = t.Warning
		default:
			typeColor = t.Muted
		}
		actorW := width - 38
		if actorW < 10 {
			actorW = 10
		}
		line := fmt.Sprintf(" %s%s%s %s%-9s%s %s%-12s%s %s",
			t.Muted, timeStr, Reset,
			typeColor, ev.Type, Reset,
			t.Warning, ev.Action, Reset,
			Pad(ev.Actor, actorW),
		)
		runes := []rune(line)
		if len(runes) > width-1 {
			line = string(runes[:width-1])
		}
		b.WriteString(line + ClearLine())
	}
}

func (v *View) renderStatsDetail(b *strings.Builder, col, width, startRow, rows int) {
	t := v.theme
	if len(v.stats) == 0 {
		b.WriteString(MoveTo(startRow, col))
		b.WriteString(t.Muted + " No stats. Select a container and press 's'." + Reset)
		return
	}
	b.WriteString(MoveTo(startRow, col))
	b.WriteString(t.Muted + fmt.Sprintf(" %-20s %8s %12s %8s %6s", "NAME", "CPU%", "MEM", "MEM%", "PIDS") + Reset)
	row := startRow + 1
	for _, s := range v.stats {
		if row >= startRow+rows {
			break
		}
		b.WriteString(MoveTo(row, col))
		cpuColor := ColorizePercent(t, s.CPUPercent)
		memColor := ColorizePercent(t, s.MemPercent)
		b.WriteString(fmt.Sprintf(" %-20s %s%7.1f%%%s %s%12s%s %s%7.1f%%%s %6d",
			Pad(s.Name, 20),
			cpuColor, s.CPUPercent, Reset,
			memColor, FormatBytes(s.MemUsage)+"/"+FormatBytes(s.MemLimit), Reset,
			memColor, s.MemPercent, Reset,
			s.PidsCount,
		) + ClearLine())
		row++
	}
}

func (v *View) renderInspectDetail(b *strings.Builder, col, width, startRow, rows int) {
	t := v.theme
	if v.inspect == "" {
		b.WriteString(MoveTo(startRow, col))
		b.WriteString(t.Muted + " Select a container, press 'i' to inspect." + Reset)
		return
	}
	lines := strings.Split(v.inspect, "\n")
	for i, line := range lines {
		if i >= rows {
			break
		}
		b.WriteString(MoveTo(startRow+i, col))
		runes := []rune(line)
		if len(runes) > width-2 {
			runes = runes[:width-2]
		}
		b.WriteString(" " + string(runes) + ClearLine())
	}
}

func (v *View) renderColumnSeparators(b *strings.Builder, leftW, centerW, height int) {
	t := v.theme
	for row := 3; row <= 2+height; row++ {
		b.WriteString(MoveTo(row, leftW+1))
		b.WriteString(t.Muted + Separator() + Reset)
	}
	sepCol := leftW + 2 + centerW
	for row := 3; row <= 2+height; row++ {
		b.WriteString(MoveTo(row, sepCol))
		b.WriteString(t.Muted + Separator() + Reset)
	}
}

func (v *View) renderStatusBar(b *strings.Builder) {
	t := v.theme
	b.WriteString(MoveTo(v.height, 1))

	if v.searching {
		b.WriteString(t.Primary + bold + " SEARCH: " + Reset + v.searchBuf + "_ ")
		b.WriteString(t.Muted + " [Enter] confirm  [Esc] cancel" + Reset)
		return
	}

	if v.statusMsg != "" {
		color := t.Success
		if v.statusErr {
			color = t.Error
		}
		b.WriteString(color + bold + " " + v.statusMsg + Reset)
		return
	}

	shortcuts := [][]string{
		{"Tab", "panel"}, {"↑↓", "nav"},
		{"l", "logs"}, {"e", "events"}, {"s", "stats"}, {"i", "inspect"},
		{"r", "restart"}, {"x", "stop"}, {"u", "up"}, {"d", "down"},
		{"/", "search"}, {"?", "help"}, {"q", "quit"},
	}
	var parts []string
	for _, kv := range shortcuts {
		k, label := kv[0], kv[1]
		parts = append(parts, t.BgSelected+bold+k+Reset+t.Muted+" "+label+Reset)
	}
	b.WriteString(" " + strings.Join(parts, "  "))
}

func (v *View) renderHelp(b *strings.Builder) {
	t := v.theme
	b.WriteString("\x1b[2J\x1b[H")
	centerRow := v.height / 2
	centerCol := v.width / 2
	helpLines := []string{
		"",
		"  ╔══════════════════════════════════════╗",
		"  ║          d9s  Keyboard Help           ║",
		"  ╠══════════════════════════════════════╣",
		"  ║  Tab          Switch panel            ║",
		"  ║  ↑ / ↓        Navigate list           ║",
		"  ║  Enter        Select / Open detail    ║",
		"  ║  /            Search containers       ║",
		"  ║  l            View Logs               ║",
		"  ║  e            View Events             ║",
		"  ║  i            Inspect container       ║",
		"  ║  s            Stats view              ║",
		"  ║  S (shift+s)  Open shell (exec)       ║",
		"  ║  r            Restart container       ║",
		"  ║  x            Stop container          ║",
		"  ║  R / Delete   Remove container        ║",
		"  ║  u            Compose up              ║",
		"  ║  d            Compose down            ║",
		"  ║  p            Compose pull            ║",
		"  ║  b            Compose build           ║",
		"  ║  ?            Toggle this help        ║",
		"  ║  q  /  Ctrl+C Quit                   ║",
		"  ╚══════════════════════════════════════╝",
		"",
	}
	startRow := centerRow - len(helpLines)/2
	for i, line := range helpLines {
		b.WriteString(MoveTo(startRow+i, centerCol-22))
		b.WriteString(t.Primary + line + Reset)
	}
	b.WriteString(MoveTo(startRow+len(helpLines), centerCol-10))
	b.WriteString(t.Muted + "Press any key to close" + Reset)
}

func (v *View) renderConfirm(b *strings.Builder) {
	t := v.theme
	centerRow := v.height / 2
	centerCol := v.width / 2
	box := []string{
		"  ┌────────────────────────────────┐",
		"  │           Confirm              │",
		"  │                                │",
		fmt.Sprintf("  │  %-30s  │", Pad(v.confirmMsg, 30)),
		"  │                                │",
		"  │    [y] Yes        [n] No       │",
		"  └────────────────────────────────┘",
	}
	startRow := centerRow - 4
	for i, line := range box {
		b.WriteString(MoveTo(startRow+i, centerCol-18))
		b.WriteString(t.Warning + line + Reset)
	}
}

// Navigation methods
func (v *View) MoveDown() {
	switch v.activePanel {
	case PanelContainers:
		filtered := v.filteredContainers()
		if v.containerIdx < len(filtered)-1 {
			v.containerIdx++
		}
	case PanelProjects:
		if v.projectIdx < len(v.projects)-1 {
			v.projectIdx++
		}
	case PanelContexts:
		if v.contextIdx < len(v.contexts)-1 {
			v.contextIdx++
		}
	}
}

func (v *View) MoveUp() {
	switch v.activePanel {
	case PanelContainers:
		if v.containerIdx > 0 {
			v.containerIdx--
		}
	case PanelProjects:
		if v.projectIdx > 0 {
			v.projectIdx--
		}
	case PanelContexts:
		if v.contextIdx > 0 {
			v.contextIdx--
		}
	}
}

func (v *View) TabNext() {
	switch v.activePanel {
	case PanelContexts:
		v.activePanel = PanelProjects
	case PanelProjects:
		v.activePanel = PanelContainers
	case PanelContainers:
		v.activePanel = PanelContexts
	}
}

func (v *View) SetActiveTab(tab DetailTab) { v.activeTab = tab }

func (v *View) StartSearch() {
	v.searching = true
	v.searchBuf = ""
}

func (v *View) AppendSearch(ch rune) {
	v.searchBuf += string(ch)
	v.containerIdx = 0
}

func (v *View) BackspaceSearch() {
	if len(v.searchBuf) > 0 {
		v.searchBuf = v.searchBuf[:len(v.searchBuf)-1]
	}
}

func (v *View) CommitSearch()  { v.searching = false }
func (v *View) ClearSearch()   { v.searching = false; v.searchBuf = ""; v.containerIdx = 0 }
func (v *View) ToggleHelp()    { v.showHelpFlag = !v.showHelpFlag }
func (v *View) ShowHelp() bool { return v.showHelpFlag }
func (v *View) SetStatus(msg string, isErr bool) {
	v.statusMsg = msg
	v.statusErr = isErr
}
func (v *View) ClearStatus()        { v.statusMsg = ""; v.statusErr = false }
func (v *View) ShowConfirm(msg string) { v.confirmMsg = msg; v.confirmActive = true }
func (v *View) HideConfirm()        { v.confirmActive = false }
func (v *View) ConfirmActive() bool { return v.confirmActive }
func (v *View) IsSearching() bool   { return v.searching }
func (v *View) Resize(w, h int)     { v.width = w; v.height = h }
