// KHLIFI HOUCEM / INGENIEUR DEVSECOPS && CLOUD
package tui

import (
	"fmt"
	"strings"
)

func (v *View) renderTrivyDetail(b *strings.Builder, col, width, startRow, rows int) {
	v.renderSecurityDetail(b, "Trivy", col, width, startRow, rows)
}

func (v *View) renderSnykDetail(b *strings.Builder, col, width, startRow, rows int) {
	v.renderSecurityDetail(b, "Snyk", col, width, startRow, rows)
}

func (v *View) renderSecurityDetail(b *strings.Builder, scanner string, col, width, startRow, rows int) {
	t := v.theme
	img := v.ActiveImage()
	if img == nil {
		b.WriteString(MoveTo(startRow, col))
		b.WriteString(t.Muted + " No image selected." + Reset + ClearLine())
		v.clearRemainingRows(b, col, startRow+1, startRow+rows)
		return
	}

	if v.scanInProgress[img.ID] {
		b.WriteString(MoveTo(startRow, col))
		b.WriteString(t.Warning + " Scan in progress... Please wait." + Reset + ClearLine())
		v.clearRemainingRows(b, col, startRow+1, startRow+rows)
		return
	}

	if msg, ok := v.scanErrors[img.ID]; ok {
		b.WriteString(MoveTo(startRow, col))
		b.WriteString(t.Danger + " Scan failed: " + Reset + ClearLine())
		b.WriteString(MoveTo(startRow+1, col))
		b.WriteString(t.Muted + truncate(msg, width-2) + Reset + ClearLine())
		v.clearRemainingRows(b, col, startRow+2, startRow+rows)
		return
	}

	res, ok := v.security[img.ID][scanner]
	if !ok {
		b.WriteString(MoveTo(startRow, col))
		b.WriteString(t.Muted + " No " + scanner + " results found. Press 's' to scan." + Reset + ClearLine())
		v.clearRemainingRows(b, col, startRow+1, startRow+rows)
		return
	}

	var lines []string

	// Summary
	s := res.Summary
	lines = append(lines, fmt.Sprintf("%sSummary: %s%d Critical%s, %s%d High%s, %d Medium, %d Low%s",
		bold,
		t.Danger, s.Critical, Reset+bold,
		t.Warning, s.High, Reset+bold,
		s.Medium, s.Low, ClearLine()))
	
	lines = append(lines, "")
	
	// Vulnerabilities
	if len(res.Vulnerabilities) > 0 {
		lines = append(lines, t.Primary + underline + "VULNERABILITIES" + Reset + ClearLine())
		for _, vuln := range res.Vulnerabilities {
			sevColor := v.severityColor(vuln.Severity)
			lines = append(lines, fmt.Sprintf("%s%-10s%s %-12s %s%s", 
				sevColor, vuln.Severity, Reset,
				truncate(vuln.ID, 12),
				truncate(vuln.Package, width-25),
				ClearLine()))
		}
	}

	// Misconfigs
	if len(res.Misconfigs) > 0 {
		lines = append(lines, "")
		lines = append(lines, t.Primary + underline + "MISCONFIGURATIONS" + Reset + ClearLine())
		for _, m := range res.Misconfigs {
			sevColor := v.severityColor(m.Severity)
			lines = append(lines, fmt.Sprintf("%s%-10s%s %s%s", 
				sevColor, m.Severity, Reset,
				truncate(m.Title, width-12),
				ClearLine()))
		}
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
		b.WriteString(" " + line)
	}
	
	if len(visibleLines) < rows {
		v.clearRemainingRows(b, col, startRow+len(visibleLines), startRow+rows)
	}

	v.renderScrollbar(b, totalLines, startRow, rows, col, width)
}

func (v *View) clearRemainingRows(b *strings.Builder, col, start, end int) {
	for i := start; i < end; i++ {
		b.WriteString(MoveTo(i, col) + ClearLine())
	}
}

func (v *View) severityColor(sev string) string {
	switch sev {
	case "CRITICAL":
		return v.theme.Danger + bold
	case "HIGH":
		return v.theme.Danger
	case "MEDIUM":
		return v.theme.Warning
	case "LOW":
		return v.theme.Info
	default:
		return v.theme.Muted
	}
}

func (v *View) renderRecommendationsDetail(b *strings.Builder, col, width, startRow, rows int) {
	t := v.theme
	img := v.ActiveImage()
	if img == nil {
		b.WriteString(MoveTo(startRow, col))
		b.WriteString(t.Muted + " No image selected." + Reset + ClearLine())
		v.clearRemainingRows(b, col, startRow+1, startRow+rows)
		return
	}

	recs, ok := v.recommendations[img.ID]
	if !ok || len(recs) == 0 {
		b.WriteString(MoveTo(startRow, col))
		b.WriteString(t.Muted + " No recommendations available. Run a scan first." + Reset + ClearLine())
		v.clearRemainingRows(b, col, startRow+1, startRow+rows)
		return
	}

	var lines []string

	for _, rec := range recs {
		sevColor := v.severityColor(rec.Severity)
		lines = append(lines, sevColor + "● " + Reset + bold + rec.Title + Reset + ClearLine())
		lines = append(lines, t.Muted + "  " + truncate(rec.Recommendation, width-4) + Reset + ClearLine())
		lines = append(lines, "")
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
		b.WriteString(" " + line)
	}
	
	if len(visibleLines) < rows {
		v.clearRemainingRows(b, col, startRow+len(visibleLines), startRow+rows)
	}

	v.renderScrollbar(b, totalLines, startRow, rows, col, width)
}
