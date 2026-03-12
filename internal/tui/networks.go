// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
package tui

import (
	"fmt"
	"strings"
)

func (v *View) renderNetworks(b *strings.Builder, startCol, width, contentHeight int) {
	t := v.theme
	
	col1 := 14 // ID
	col2 := 30 // NAME
	col3 := 10 // DRIVER
	col4 := 10 // SCOPE
	
	// Column headers
	b.WriteString(MoveTo(3, startCol))
	header := t.Muted + bold +
		Pad("ID", col1) +
		Pad("NAME", col2) +
		Pad("DRIVER", col3) +
		Pad("SCOPE", col4) + Reset
	b.WriteString(header)
	
	b.WriteString(MoveTo(4, startCol))
	b.WriteString(t.Muted + strings.Repeat("─", width) + Reset)
	
	row := 5
	for i, net := range v.networks {
		if row >= 3+contentHeight {
			break
		}
		
		b.WriteString(MoveTo(row, startCol))
		style := ""
		if v.activePanel == PanelNetworks && i == v.networkIdx {
			style = t.BgSelected
		}
		
		id := net.ID
		if len(id) > 12 {
			id = id[:12]
		}
		
		line := fmt.Sprintf("%-13s %-29s %-9s %-9s",
			id,
			truncate(net.Name, col2-1),
			net.Driver,
			net.Scope,
		)
		
		b.WriteString(style + Pad(line, width) + Reset)
		row++
	}
	
	if len(v.networks) == 0 {
		b.WriteString(MoveTo(row, startCol))
		b.WriteString(t.Muted + "  No networks found" + Reset)
	}
}
