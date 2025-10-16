package chttp

import (
	"context"
	"net/http"

	"github.com/alcionai/clues"
)

// InheritorMiddleware builds a http middleware which automatically
// inherits initialized clients from the clues ecosystem and embeds them in
// the request context.  Since clues prefers context-bound client propagation
// over global singletons, this behavior is necessary to ensure request contexts
// maintain scope of initialized values.
func InheritorMiddleware(
	ctx context.Context,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			rctx := r.Context()

			// always assume the request context needs to be clobbered.
			rctx = clues.Inherit(ctx, rctx, true)

			r = r.WithContext(rctx)

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
