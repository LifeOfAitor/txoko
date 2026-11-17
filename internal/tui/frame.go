package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// span is a piece of text with its style. Keeping the plain text around
// lets the layout measure widths without parsing escape codes.
type span struct {
	text  string
	style lipgloss.Style
}

func (s span) width() int     { return lipgloss.Width(s.text) }
func (s span) render() string { return s.style.Render(s.text) }

func spansWidth(ss []span) int {
	n := 0
	for _, s := range ss {
		n += s.width()
	}
	return n
}

func renderSpans(ss []span) string {
	var b strings.Builder
	for _, s := range ss {
		b.WriteString(s.render())
	}
	return b.String()
}

// frame draws a rounded box around body, btop-style, with a title in the top
// border and a footer row separated from the body:
//
//	╭─ title ──────────╮
//	│ body             │
//	├──────────────────┤
//	│ footer           │
//	╰──────────────────╯
//
// Every body line must already be exactly inner cells wide.
func frame(inner int, title []span, body []string, footer []string, t Theme) string {
	b := t.Border.Render
	var out strings.Builder

	// Top border, with the title if it fits.
	if tw := spansWidth(title); tw > 0 && inner >= tw+3 {
		out.WriteString(b("╭─ ") + renderSpans(title) + b(" "+strings.Repeat("─", inner-tw-3)+"╮"))
	} else {
		out.WriteString(b("╭" + strings.Repeat("─", inner) + "╮"))
	}
	out.WriteByte('\n')

	for _, line := range body {
		out.WriteString(b("│") + line + b("│") + "\n")
	}

	out.WriteString(b("├"+strings.Repeat("─", inner)+"┤") + "\n")
	for _, line := range footer {
		out.WriteString(b("│") + line + b("│") + "\n")
	}
	out.WriteString(b("╰" + strings.Repeat("─", inner) + "╯"))
	return out.String()
}

// layout is one way to arrange the footer: a left and a right group.
type layout struct{ left, right []span }

// footerLine renders the first layout that fits in exactly inner cells, with
// the left group flush left and the right group flush right. Callers list
// layouts from richest to most compact; if none fits, the line is blank
// rather than overflowing the frame.
func footerLine(inner int, layouts ...layout) string {
	for _, l := range layouts {
		lw, rw := spansWidth(l.left), spansWidth(l.right)
		need := lw + rw + 2 // one space of padding on each side
		if len(l.right) > 0 {
			need++ // keep at least one space between the groups
		}
		if need > inner {
			continue
		}
		gap := inner - lw - rw - 2
		return " " + renderSpans(l.left) + strings.Repeat(" ", gap) + renderSpans(l.right) + " "
	}
	return strings.Repeat(" ", max(inner, 0))
}

// footerBlock lays out the footer: a status on the left and the key hints.
// When everything fits on one line it uses one. Otherwise the status gets a
// line of its own and the hints wrap onto as many lines as they need, so
// every key is always listed. compact is a shorter status for very narrow
// frames. Every returned line is exactly inner cells wide.
func footerBlock(inner int, status, compact []span, hints [][]span) []string {
	sep := span{text: "  "}
	var all []span
	for i, h := range hints {
		if i > 0 {
			all = append(all, sep)
		}
		all = append(all, h...)
	}
	if spansWidth(status)+spansWidth(all)+3 <= inner {
		return []string{footerLine(inner, layout{status, all})}
	}

	lines := []string{footerLine(inner, layout{status, nil}, layout{compact, nil})}
	var cur []span
	flush := func() {
		if len(cur) > 0 {
			lines = append(lines, footerLine(inner, layout{cur, nil}))
			cur = nil
		}
	}
	for _, h := range hints {
		if len(cur) > 0 && spansWidth(cur)+spansWidth(h)+len(sep.text)+2 > inner {
			flush()
		}
		if len(cur) > 0 {
			cur = append(cur, sep)
		}
		cur = append(cur, h...)
	}
	flush()
	return lines
}

// progressBar renders a ▰▰▰▱▱ bar of n cells for a value between 0 and 1.
func progressBar(v float64, n int) (filled, empty string) {
	k := int(v*float64(n) + 0.5)
	k = min(max(k, 0), n)
	return strings.Repeat("▰", k), strings.Repeat("▱", n-k)
}
