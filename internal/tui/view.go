// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
package tui

import (
	"fmt"
	"strings"

	"github.com/houcemdevops007/d9s/internal/domain"
	"github.com/houcemdevops007/d9s/internal/store"
	"github.com/houcemdevops007/d9s/pkg/version"
)

// Panel represents a named display panel.
type Panel int

const (
	PanelContainers Panel = iota
	PanelProjects
	PanelContexts
	PanelImages
	PanelVolumes
	PanelNetworks
)

// DetailTab represents the active detail tab.
type DetailTab int

const (
	TabLogs DetailTab = iota
	TabEvents
	TabStats
	TabInspect
	TabTrivy
	TabSnyk
	TabRecommendations
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
	images         []domain.Image
	imageIdx       int
	volumes        []domain.Volume
	volumeIdx      int
	networks       []domain.Network
	networkIdx     int
	logs           []string
	events         []domain.RuntimeEvent
	stats          map[string]domain.RuntimeStats
	security       map[string]map[string]domain.SecurityScanResult
	recommendations map[string][]domain.BestPracticeRecommendation
	scanInProgress map[string]bool
	scanErrors     map[string]string
	inspect        string
	detailScroll   int
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
	v.images = st.Images
	v.volumes = st.Volumes
	v.networks = st.Networks
	v.events = st.Events
	v.stats = st.Stats
	v.security = st.SecurityResults
	v.recommendations = st.Recommendations
	v.scanInProgress = st.ScanInProgress
	v.scanErrors = st.ScanningErrors

	if v.containerIdx >= len(v.containers) && len(v.containers) > 0 {
		v.containerIdx = len(v.containers) - 1
	}
	if v.projectIdx >= len(v.projects) && len(v.projects) > 0 {
		v.projectIdx = len(v.projects) - 1
	}
	if v.contextIdx >= len(v.contexts) && len(v.contexts) > 0 {
		v.contextIdx = len(v.contexts) - 1
	}
	if v.imageIdx >= len(v.images) && len(v.images) > 0 {
		v.imageIdx = len(v.images) - 1
	}
	if v.volumeIdx >= len(v.volumes) && len(v.volumes) > 0 {
		v.volumeIdx = len(v.volumes) - 1
	}
	if v.networkIdx >= len(v.networks) && len(v.networks) > 0 {
		v.networkIdx = len(v.networks) - 1
	}
}

// SetInspect sets the inspect panel content.
func (v *View) SetInspect(s string) {
	v.inspect = s
	v.detailScroll = 0
}

func (v *View) ScrollDetail(delta int) {
	totalLines := 0
	switch v.activeTab {
	case TabLogs:
		totalLines = len(v.logs)
	case TabEvents:
		totalLines = len(v.events)
	case TabStats:
		totalLines = len(v.stats) + 1
	case TabInspect:
		totalLines = len(strings.Split(v.inspect, "\n"))
	case TabTrivy, TabSnyk:
		if img := v.ActiveImage(); img != nil {
			scanner := "Trivy"
			if v.activeTab == TabSnyk {
				scanner = "Snyk"
			}
			if res, ok := v.security[img.ID][scanner]; ok {
				totalLines = 4 + len(res.Vulnerabilities) + len(res.Misconfigs) + 2
			}
		}
	case TabRecommendations:
		if img := v.ActiveImage(); img != nil {
			if recs, ok := v.recommendations[img.ID]; ok {
				totalLines = len(recs) * 2
			}
		}
	}

	maxScroll := totalLines - (v.height - 8)
	if maxScroll < 0 {
		maxScroll = 0
	}
	v.detailScroll += delta
	if v.detailScroll < 0 {
		v.detailScroll = 0
	}
	if v.detailScroll > maxScroll {
		v.detailScroll = maxScroll
	}
}

// ResetScroll resets the detail scroll offset.
func (v *View) ResetScroll() {
	v.detailScroll = 0
}

// SetLogs sets the log lines.
func (v *View) SetLogs(lines []string) { v.logs = lines }

// SetActivePanel changes the current focus panel.
func (v *View) SetActivePanel(p Panel) { v.activePanel = p }

// ActivePanel returns the currently active panel.
func (v *View) ActivePanel() Panel { return v.activePanel }

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

// ActiveImage returns the currently selected image or nil.
func (v *View) ActiveImage() *domain.Image {
	if len(v.images) == 0 {
		return nil
	}
	if v.imageIdx >= len(v.images) {
		v.imageIdx = len(v.images) - 1
	}
	return &v.images[v.imageIdx]
}

// ActiveVolume returns the currently selected volume or nil.
func (v *View) ActiveVolume() *domain.Volume {
	if len(v.volumes) == 0 {
		return nil
	}
	if v.volumeIdx >= len(v.volumes) {
		v.volumeIdx = len(v.volumes) - 1
	}
	return &v.volumes[v.volumeIdx]
}

// ActiveNetwork returns the currently selected network or nil.
func (v *View) ActiveNetwork() *domain.Network {
	if len(v.networks) == 0 {
		return nil
	}
	if v.networkIdx >= len(v.networks) {
		v.networkIdx = len(v.networks) - 1
	}
	return &v.networks[v.networkIdx]
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
	
	switch v.activePanel {
	case PanelContainers, PanelContexts, PanelProjects:
		v.renderCenterPanel(&b, leftW+2, centerW, contentHeight)
	case PanelImages:
		v.renderImages(&b, leftW+2, centerW, contentHeight)
	case PanelVolumes:
		v.renderVolumes(&b, leftW+2, centerW, contentHeight)
	case PanelNetworks:
		v.renderNetworks(&b, leftW+2, centerW, contentHeight)
	}
	
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
	headerText := " ⬡ d9s  Docker TUI"
	authorText := "by " + version.Author
	
	// Show author in header - adjust padding based on width
	fullHeader := fmt.Sprintf("%-20s %s", headerText, authorText)
	
	header := Pad(fullHeader, v.width)
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
		{TabTrivy, "Trivy Scan"},
		{TabSnyk, "Snyk Scan"},
		{TabRecommendations, "Best Practices"},
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
	case TabTrivy:
		v.renderTrivyDetail(b, startCol, width, contentStart, contentRows)
	case TabSnyk:
		v.renderSnykDetail(b, startCol, width, contentStart, contentRows)
	case TabRecommendations:
		v.renderRecommendationsDetail(b, startCol, width, contentStart, contentRows)
	}
}

func (v *View) renderLogsDetail(b *strings.Builder, col, width, startRow, rows int) {
	t := v.theme
	if len(v.logs) == 0 {
		b.WriteString(MoveTo(startRow, col))
		b.WriteString(t.Muted + " No logs. Select container, press 'l'." + Reset + ClearLine())
		v.clearRemainingRows(b, col, startRow+1, startRow+rows)
		return
	}
	lines := v.logs
	totalLines := len(lines)
	
	startIdx := v.detailScroll
	if startIdx > totalLines {
		startIdx = totalLines
	}
	
	visibleLines := lines[startIdx:]

	for i, line := range visibleLines {
		if i >= rows {
			break
		}
		b.WriteString(MoveTo(startRow+i, col))
		runes := []rune(line)
		if len(runes) > width-3 {
			runes = runes[:width-3]
		}
		b.WriteString(" " + string(runes) + ClearLine())
	}
	
	if len(visibleLines) < rows {
		v.clearRemainingRows(b, col, startRow+len(visibleLines), startRow+rows)
	}

	v.renderScrollbar(b, totalLines, startRow, rows, col, width)
}

func (v *View) renderEventsDetail(b *strings.Builder, col, width, startRow, rows int) {
	t := v.theme
	if len(v.events) == 0 {
		b.WriteString(MoveTo(startRow, col))
		b.WriteString(t.Muted + " Waiting for events..." + Reset + ClearLine())
		v.clearRemainingRows(b, col, startRow+1, startRow+rows)
		return
	}
	events := v.events
	totalLines := len(events)
	
	startIdx := v.detailScroll
	if startIdx > totalLines {
		startIdx = totalLines
	}
	
	visibleEvents := events[startIdx:]

	for i, ev := range visibleEvents {
		if i >= rows {
			break
		}
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
		if len(runes) > width-3 {
			line = string(runes[:width-3])
		}
		b.WriteString(line + ClearLine())
	}
	
	if len(visibleEvents) < rows {
		v.clearRemainingRows(b, col, startRow+len(visibleEvents), startRow+rows)
	}

	v.renderScrollbar(b, totalLines, startRow, rows, col, width)
}

func (v *View) renderStatsDetail(b *strings.Builder, col, width, startRow, rows int) {
	t := v.theme
	if len(v.stats) == 0 {
		b.WriteString(MoveTo(startRow, col))
		b.WriteString(t.Muted + " No stats. Select a container and press 's'." + Reset + ClearLine())
		v.clearRemainingRows(b, col, startRow+1, startRow+rows)
		return
	}
	var lines []string
	lines = append(lines, t.Muted + fmt.Sprintf(" %-20s %8s %12s %8s %6s", "NAME", "CPU%", "MEM", "MEM%", "PIDS") + Reset + ClearLine())
	for _, s := range v.stats {
		cpuColor := ColorizePercent(t, s.CPUPercent)
		memColor := ColorizePercent(t, s.MemPercent)
		lines = append(lines, fmt.Sprintf(" %-20s %s%7.1f%%%s %s%12s%s %s%7.1f%%%s %6d",
			Pad(s.Name, 20),
			cpuColor, s.CPUPercent, Reset,
			memColor, FormatBytes(s.MemUsage)+"/"+FormatBytes(s.MemLimit), Reset,
			memColor, s.MemPercent, Reset,
			s.PidsCount,
		)+ClearLine())
	}
	
	totalLines := len(lines)
	startIdx := v.detailScroll
	if startIdx > totalLines {
		startIdx = totalLines
	}
	
	visibleLines := lines[startIdx:]

	for i, line := range visibleLines {
		if i >= rows {
			break
		}
		b.WriteString(MoveTo(startRow+i, col))
		b.WriteString(line)
	}
	
	if len(visibleLines) < rows {
		v.clearRemainingRows(b, col, startRow+len(visibleLines), startRow+rows)
	}

	v.renderScrollbar(b, totalLines, startRow, rows, col, width)
}

func (v *View) renderInspectDetail(b *strings.Builder, col, width, startRow, rows int) {
	t := v.theme
	if v.inspect == "" {
		b.WriteString(MoveTo(startRow, col))
		b.WriteString(t.Muted + " Select a container, press 'i' to inspect." + Reset + ClearLine())
		v.clearRemainingRows(b, col, startRow+1, startRow+rows)
		return
	}
	lines := strings.Split(v.inspect, "\n")
	totalLines := len(lines)
	
	startIdx := v.detailScroll
	if startIdx > totalLines {
		startIdx = totalLines
	}
	
	visibleLines := lines[startIdx:]

	for i, line := range visibleLines {
		if i >= rows {
			break
		}
		b.WriteString(MoveTo(startRow+i, col))
		runes := []rune(line)
		if len(runes) > width-3 {
			runes = runes[:width-3]
		}
		b.WriteString(" " + string(runes) + ClearLine())
	}
	
	if len(visibleLines) < rows {
		v.clearRemainingRows(b, col, startRow+len(visibleLines), startRow+rows)
	}

	v.renderScrollbar(b, totalLines, startRow, rows, col, width)
}

func (v *View) renderScrollbar(b *strings.Builder, totalLines, startRow, rows, col, width int) {
	if totalLines > rows {
		trackChar := "│"
		thumbChar := "█"
		
		scrollPercent := float64(v.detailScroll) / float64(totalLines-rows)
		if totalLines-rows <= 0 {
			scrollPercent = 0
		}
		
		thumbHeight := int(float64(rows) * (float64(rows) / float64(totalLines)))
		if thumbHeight < 1 {
			thumbHeight = 1
		}
		
		trackHeight := rows
		thumbPos := int(scrollPercent * float64(trackHeight-thumbHeight))
		
		for i := 0; i < trackHeight; i++ {
			b.WriteString(MoveTo(startRow+i, col+width-1))
			if i >= thumbPos && i < thumbPos+thumbHeight {
				b.WriteString(v.theme.Primary + thumbChar + Reset)
			} else {
				b.WriteString(v.theme.Muted + trackChar + Reset)
			}
		}
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
		{"Tab", "panel"}, {"c", "conts"}, {"g", "imgs"}, {"v", "vols"}, {"n", "nets"},
		{"l", "logs"}, {"e", "events"}, {"s", "stats/scan"}, {"i", "inspect"},
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
		"  ║  ← / →        Switch Details Tab      ║",
		"  ║  Enter        Select / Open detail    ║",
		"  ║  /            Search containers       ║",
		"  ║  l            View Logs               ║",
		"  ║  e            View Events             ║",
		"  ║  i            Inspect container       ║",
		"  ║  s            Stats view              ║",
		"  ║  S (shift+s)  Open shell (exec)       ║",
		"  ║  r            Restart container       ║",
		"  ║  x            Stop container          ║",
		"  ║  R / Delete   Remove resource         ║",
		"  ║                                       ║",
		"  ║  Views:                               ║",
		"  ║  c            Containers (default)    ║",
		"  ║  g            Images                  ║",
		"  ║  v            Volumes                 ║",
		"  ║  n            Networks                ║",
		"  ║                                       ║",
		"  ║  Compose:                             ║",
		"  ║  u            Compose up              ║",
		"  ║  d            Compose down            ║",
		"  ║  p            Compose pull            ║",
		"  ║  b            Compose build           ║",
		"  ║  ?            Toggle this help        ║",
		"  ║  q  /  Ctrl+C Quit                   ║",
		"  ╚══════════════════════════════════════╝",
		"",
		"  " + version.Author,
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
	case PanelImages:
		if v.imageIdx < len(v.images)-1 {
			v.imageIdx++
		}
	case PanelVolumes:
		if v.volumeIdx < len(v.volumes)-1 {
			v.volumeIdx++
		}
	case PanelNetworks:
		if v.networkIdx < len(v.networks)-1 {
			v.networkIdx++
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
	case PanelImages:
		if v.imageIdx > 0 {
			v.imageIdx--
		}
	case PanelVolumes:
		if v.volumeIdx > 0 {
			v.volumeIdx--
		}
	case PanelNetworks:
		if v.networkIdx > 0 {
			v.networkIdx--
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

func (v *View) SetActiveTab(tab DetailTab) {
	v.activeTab = tab
	v.ResetScroll()
}

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
func (v *View) ActiveTab() DetailTab { return v.activeTab }

func (v *View) NextTab() {
	if v.activeTab < TabRecommendations {
		v.activeTab++
	} else {
		v.activeTab = TabLogs
	}
}

func (v *View) PrevTab() {
	if v.activeTab > TabLogs {
		v.activeTab--
	} else {
		v.activeTab = TabRecommendations
	}
}
