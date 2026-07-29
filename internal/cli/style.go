package cli

import (
	"io"
	"os"
)

// SGR sequences. Bold and dim only, deliberately: a hue that reads well on a
// dark background is often invisible on a light one, and the renderer has no way
// to know which it is facing. Weight and intensity are safe on both.
const (
	sgrReset = "\x1b[0m"
	sgrBold  = "\x1b[1m"
	sgrDim   = "\x1b[2m"
)

// style decides whether the renderer may emit SGR sequences. Its zero value is
// plain text, which is what every non-terminal writer gets.
type style struct{ on bool }

func (s style) bold(v string) string { return s.wrap(sgrBold, v) }
func (s style) dim(v string) string  { return s.wrap(sgrDim, v) }

// wrap resets with 0m rather than the narrower 22m: the renderer never nests
// styles, so the blunt reset is correct and is understood everywhere. The reset
// is emitted before the newline, so a line never carries styling past its end.
func (s style) wrap(code, v string) string {
	if !s.on || v == "" {
		return v
	}
	return code + v + sgrReset
}

// Option adjusts presentation only. No option changes a number, a label, or
// whether a line appears.
type Option func(*renderOpts)

type renderOpts struct{ noColor bool }

// NoColor suppresses every escape sequence regardless of what the writer is; it
// is what --no-color selects. NoColor(false) leaves the decision to the writer,
// which is the default and the right answer for every caller but the flag
// parser.
func NoColor(v bool) Option { return func(o *renderOpts) { o.noColor = v } }

// styleFor resolves the options against the writer. Callers that pass no
// options get auto-detection, which is why a test writing to a buffer never has
// to ask for plain text.
func styleFor(w io.Writer, opts []Option) style {
	var o renderOpts
	for _, apply := range opts {
		apply(&o)
	}
	return style{on: colorEnabled(w, o.noColor)}
}

// colorEnabled reports whether w should receive SGR sequences.
//
// Precedence, strongest first:
//  1. --no-color, an explicit instruction from the person running the command;
//  2. NO_COLOR set to a non-empty value (https://no-color.org);
//  3. TERM=dumb, the terminal saying it cannot render them;
//  4. w being a character device — a pipe, a file or a test buffer must get
//     bytes a later reader can use, not escape sequences it has to strip.
//
// There is deliberately no way to force styling on. --no-color can only
// subtract, so the auto-detection is never wrong in the direction that produces
// unreadable output.
func colorEnabled(w io.Writer, noColor bool) bool {
	if noColor {
		return false
	}
	// Non-empty, per the NO_COLOR spec: an empty value does not disable.
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return isTerminal(w)
}

// isTerminal reports whether w is a character device. Anything that is not an
// *os.File — a bytes.Buffer in a test, an http.ResponseWriter — is not, which is
// why the tests never see an escape sequence and never have to ask not to.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
