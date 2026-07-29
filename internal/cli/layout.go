package cli

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

const (
	rowIndent  = "  "   // every row sits under its section heading
	noteIndent = "    " // a note sits under the row it qualifies
	colGap     = "  "   // between label, value and unit
)

type rowKind uint8

const (
	rowNum     rowKind = iota // right-aligned value, so decimal points line up
	rowText                   // left-aligned value: names, timestamps, "unavailable …"
	rowNote                   // dim, deeper indent, no columns
	rowSection                // bold heading, blank line before
)

type row struct {
	kind  rowKind
	label string
	value string
	unit  string // unit, or a short annotation such as "(2 pauses)"; always dimmed
}

// table accumulates the whole report before writing a byte of it, so the column
// widths come from the data rather than from hand-counted literals. A track
// without elevation never pays for the 36-character effort labels, and a label
// can be reworded without re-counting spaces on twenty other lines.
type table struct {
	st   style
	rows []row
}

// section starts a new group. Every group is preceded by a blank line, which is
// what lets a heading carry no rule of its own.
func (t *table) section(name string) {
	t.rows = append(t.rows, row{kind: rowSection, label: name})
}

// num records a metric whose value is a number: right-aligned in the shared
// value column, with its unit — or a short annotation — in the column to its
// right.
func (t *table) num(label, value, unit string) {
	t.rows = append(t.rows, row{kind: rowNum, label: label, value: value, unit: unit})
}

// text records a metric whose value is prose. It starts where the numeric column
// starts and runs rightward; nothing follows it, so nothing needs it padded and
// its width never affects the layout.
func (t *table) text(label, value string) {
	t.rows = append(t.rows, row{kind: rowText, label: label, value: value})
}

// note records a subordinate line — an effort legend, the segment explanation.
// Dim and indented past the rows, so it reads as belonging to the row above
// rather than as a metric of its own.
func (t *table) note(s string) {
	t.rows = append(t.rows, row{kind: rowNote, label: s})
}

// flush writes the accumulated rows.
//
// Widths are measured with len because every string it measures is ASCII by
// construction: labels are constants, and numeric values are digits, dots and
// duration letters. The one user-controlled string — an activity name — is a
// text value, which is last on its line and therefore never measured.
func (t *table) flush(w io.Writer) {
	labelW, valueW := 0, 0
	for _, r := range t.rows {
		switch r.kind {
		case rowNum:
			labelW = max(labelW, len(r.label))
			valueW = max(valueW, len(r.value))
		case rowText:
			labelW = max(labelW, len(r.label))
		case rowNote, rowSection:
		}
	}

	for _, r := range t.rows {
		switch r.kind {
		case rowSection:
			writeSection(w, t.st, r.label)
		case rowNote:
			fmt.Fprintf(w, "%s%s\n", noteIndent, t.st.dim(r.label))
		case rowNum:
			fmt.Fprintf(w, "%s%-*s%s%*s", rowIndent, labelW, r.label, colGap, valueW, r.value)
			if r.unit != "" {
				fmt.Fprintf(w, "%s%s", colGap, t.st.dim(r.unit))
			}
			fmt.Fprintln(w)
		case rowText:
			fmt.Fprintf(w, "%s%-*s%s%s\n", rowIndent, labelW, r.label, colGap, r.value)
		}
	}
}

// writeSection is shared with the splits table and the charts, which render
// after the main table has flushed and therefore cannot go through it.
func writeSection(w io.Writer, st style, name string) {
	fmt.Fprintf(w, "\n%s\n", st.bold(name))
}

// writeTitle underlines the report title with a rule the width of it.
// RuneCountInString, not len: "═" is three bytes and a len-based rule would be
// three times too long.
func writeTitle(w io.Writer, st style, s string) {
	fmt.Fprintf(w, "%s\n%s\n", st.bold(s), st.bold(strings.Repeat("═", utf8.RuneCountInString(s))))
}

// grid renders a small matrix whose column widths come from the data. It is what
// replaces the "%2d … %-8s" splits format, which mis-aligned the moment a track
// passed 99 kilometers or a split passed nine characters of duration.
//
// Every cell is right-aligned under its header, so no line carries trailing
// whitespace.
type grid struct {
	head []string
	rows [][]string
}

// add appends a row. Callers build the cells in header order; a row with a
// different length than the header is a programming error, and the width pass
// below tolerates a short one rather than panicking on a long one.
func (g *grid) add(cells ...string) { g.rows = append(g.rows, cells) }

func (g *grid) flush(w io.Writer, st style) {
	widths := make([]int, len(g.head))
	for i, h := range g.head {
		widths[i] = len(h)
	}
	for _, r := range g.rows {
		for i, c := range r {
			if i < len(widths) {
				widths[i] = max(widths[i], len(c))
			}
		}
	}

	fmt.Fprintln(w, st.dim(gridLine(g.head, widths)))
	for _, r := range g.rows {
		fmt.Fprintln(w, gridLine(r, widths))
	}
}

func gridLine(cells []string, widths []int) string {
	var b strings.Builder
	b.WriteString(rowIndent)
	for i, c := range cells {
		if i > 0 {
			b.WriteString(colGap)
		}
		fmt.Fprintf(&b, "%*s", widths[i], c)
	}
	return b.String()
}
