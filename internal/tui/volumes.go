// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
package tui

import (
	"fmt"
	"strings"
)

func (v *View) renderVolumes(b *strings.Builder, startCol, width, contentHeight int) {
	t := v.theme
	
	col1 := 30 // NAME
	col2 := 10 // DRIVER
	col3 := 10 // SCOPE
	
	// Column headers
	// Column headers
	b.WriteString(MoveTo(3, startCol))
	header := t.Muted + bold +
		Pad("NAME", col1) +
		Pad("DRIVER", col2) +
		Pad("SCOPE", col3) + Reset
	b.WriteString(header)
	
	b.WriteString(MoveTo(4, startCol))
	b.WriteString(t.Muted + strings.Repeat("─", width) + Reset)
	
	row := 5
	for i, vol := range v.volumes {
		if row >= 3+contentHeight {
			break
		}
		
		b.WriteString(MoveTo(row, startCol))
		style := ""
		if v.activePanel == PanelVolumes && i == v.volumeIdx {
			style = t.BgSelected
		}
		
		line := fmt.Sprintf("%-29s %-9s %-9s",
			truncate(vol.Name, col1-1),
			vol.Driver,
			vol.Scope,
		)
		
		b.WriteString(style + Pad(line, width) + Reset)
		row++
	}
	
	if len(v.volumes) == 0 {
		b.WriteString(MoveTo(row, startCol))
		b.WriteString(t.Muted + "  No volumes found" + Reset)
	}
}
