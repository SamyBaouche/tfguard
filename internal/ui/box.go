package ui

import (
	"fmt"
	"io"
	"strings"
)

// Inner width between box borders (exclusive of ╭╮│╰╯).
const boxInner = 58

// BoxTitle writes a rounded section header.
func BoxTitle(w io.Writer, style Style, title string) {
	label := "─ " + title + " "
	fill := boxInner - len([]rune(label))
	if fill < 0 {
		fill = 0
	}
	line := "╭" + label + strings.Repeat("─", fill) + "╮"
	fmt.Fprintln(w, style.Dim(line))
}

// BoxEnd closes a section box.
func BoxEnd(w io.Writer, style Style) {
	fmt.Fprintln(w, style.Dim("╰"+strings.Repeat("─", boxInner)+"╯"))
}

// BoxLine writes one content line inside a box.
func BoxLine(w io.Writer, style Style, content string) {
	// "│ " + content + padding + "│"
	visible := stripANSILen(content)
	pad := boxInner - 1 - visible // 1 = leading space after │
	if pad < 0 {
		pad = 0
	}
	fmt.Fprintf(w, "%s %s%s%s\n",
		style.Dim("│"),
		content,
		strings.Repeat(" ", pad),
		style.Dim("│"),
	)
}

// Section writes a lightweight section label (no box).
func Section(w io.Writer, style Style, title string) {
	fmt.Fprintln(w, style.Bold("▸ "+title))
	fmt.Fprintln(w, style.Dim("  "+strings.Repeat("─", boxInner-2)))
}

func stripANSILen(s string) int {
	n := 0
	inEsc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inEsc {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				inEsc = false
			}
			continue
		}
		if c == '\033' {
			inEsc = true
			continue
		}
		n++
	}
	return n
}
