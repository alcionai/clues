package ctats

import "github.com/alcionai/clues/cluerr"

var (
	// errNoNodeInCtx is used to indicate that ctats attempted to retrieve a populated
	// node out of the provided ctx, but found none.  This means no consuming client
	// (ie: OTEL.metrics) exists in the context, so no metrics production can occur.
	errNoNodeInCtx = cluerr.New("no node in ctx - ctats and clues might not be initialized").NoTrace()
)
