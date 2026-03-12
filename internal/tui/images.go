// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
package tui

import (
	"fmt"
	"strings"

	"github.com/houcemdevops007/d9s/internal/domain"
)

func (v *View) renderImages(b *strings.Builder, startCol, width, contentHeight int) {
	t := v.theme
	
	col1 := 14 // ID
	col2 := 30 // REPOSITORY
	col3 := 12 // TAG
	col4 := 10 // SIZE
	col5 := 10 // CONTAINERS
	col6 := 20 // SECURITY
	
	// Column headers
	b.WriteString(MoveTo(3, startCol))
	header := t.Muted + bold +
		Pad("ID", col1) +
		Pad("REPOSITORY", col2) +
		Pad("TAG", col3) +
		Pad("SIZE", col4) +
		Pad("CONTAINERS", col5) +
		Pad("SECURITY (C/H/M/L)", col6) + Reset
	b.WriteString(header)
	
	b.WriteString(MoveTo(4, startCol))
	b.WriteString(t.Muted + strings.Repeat("─", width) + Reset)
	
	row := 5
	for i, img := range v.images {
		if row >= 3+contentHeight {
			break
		}
		
		b.WriteString(MoveTo(row, startCol))
		style := ""
		if v.activePanel == PanelImages && i == v.imageIdx {
			style = t.BgSelected
		}
		
		sizeStr := formatSize(img.Size)
		
		line := fmt.Sprintf("%-13s %-29s %-11s %-9s %-9d",
			img.ID,
			truncate(img.Repository, col2-1),
			truncate(img.Tag, col3-1),
			sizeStr,
			img.Containers,
		)
		
		// Add security summary (aggregate from all available scanners)
		secStr := ""
		if v.scanInProgress[img.ID] {
			secStr = t.Warning + "Scanning..." + Reset
		} else if scans, ok := v.security[img.ID]; ok && len(scans) > 0 {
			var agg domain.ScanSummary
			for _, res := range scans {
				agg.Critical += res.Summary.Critical
				agg.High += res.Summary.High
				agg.Medium += res.Summary.Medium
				agg.Low += res.Summary.Low
			}
			secStr = fmt.Sprintf("%s%d%s/%s%d%s/%d/%d", 
				t.Danger, agg.Critical, Reset,
				t.Warning, agg.High, Reset,
				agg.Medium, agg.Low)
		} else if _, ok := v.scanErrors[img.ID]; ok {
			secStr = t.Danger + "Error" + Reset
		} else {
			secStr = t.Muted + "-" + Reset
		}

		b.WriteString(style + Pad(line, width-col6) + secStr + Pad("", col6-10) + Reset)
		row++
	}
	
	if len(v.images) == 0 {
		b.WriteString(MoveTo(row, startCol))
		b.WriteString(t.Muted + "  No images found" + Reset)
	}
}

func formatSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	kb := float64(size) / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1f KB", kb)
	}
	mb := kb / 1024
	if mb < 1024 {
		return fmt.Sprintf("%.1f MB", mb)
	}
	gb := mb / 1024
	return fmt.Sprintf("%.1f GB", gb)
}

func truncate(s string, l int) string {
	if len(s) > l {
		return s[:l-1] + "…"
	}
	return s
}
