package cli

import (
	"io"

	"github.com/sgaunet/gpx-stats/internal/stats"
)

// Exported aliases so black-box (cli_test) tests can exercise the unexported
// styling and layout helpers directly. Compiled only during testing.
var (
	ColorEnabled   = colorEnabled
	IsTerminal     = isTerminal
	FormatDuration = formatDuration
)

// StyledBold and StyledDim expose the SGR wrappers with styling forced on, which
// is the one state no exported entry point can reach: --no-color can only
// subtract, and a test writer is never a terminal.
func StyledBold(s string) string { return style{on: true}.bold(s) }
func StyledDim(s string) string  { return style{on: true}.dim(s) }

// RenderSplitsPlain renders just the splits table, unstyled, so its width
// computation can be asserted without the surrounding statistics.
func RenderSplitsPlain(w io.Writer, r stats.Result) { writeSplits(w, style{}, r) }
