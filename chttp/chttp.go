package chttp

import (
	"context"
	"net/http"

	"github.com/alcionai/clues"
	"github.com/alcionai/clues/clog"
	"github.com/alcionai/clues/ctats"
)

// NewInheritorHTTPMiddleware builds a http middleware which automatically
// inherits initialized clients from the clues ecosystem and embeds them in
// the request context.  Since clues prefers context-bound client propagation
// over global singletons, this behavior is necessary to ensure request contexts
// maintain scope of initialized values.
func NewInheritorHTTPMiddleware(
	ctx context.Context,
	handler http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rctx := r.Context()

		// always assume the request context needs to be clobbered.
		rctx = clues.Inherit(ctx, rctx, true)
		rctx = clog.Inherit(ctx, rctx, true)
		rctx = ctats.Inherit(ctx, rctx, true)

		r = r.WithContext(rctx)

		handler.ServeHTTP(w, r)
	})
}
