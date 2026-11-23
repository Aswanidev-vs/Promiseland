// Package api assembles the HTTP router. Domain modules are mounted here in
// later releases; each exposes Register(r chi.Router, d *shared.Deps).
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

const readyTimeout = 2 * time.Second

// New builds the root HTTP handler. Middleware order is RequestID ->
// Recoverer -> CORS -> routes.
func New(d *shared.Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(shared.RequestID)
	r.Use(shared.Recoverer)
	r.Use(shared.CORS(d.Config.CORSOrigin))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		shared.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()
		if err := d.DB.PingContext(ctx); err != nil {
			if d.Log != nil {
				d.Log.Warn("readiness check failed", "error", err)
			}
			shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "Database is not ready.")
			return
		}
		shared.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Public routes are registered directly on r.
		// [R1] auth.Register(r, d)
		// [R1] users.Register(r, d)
		// [R1] anime.Register(r, d)
		// [R1] communities.Register(r, d)
		// [R1] posts.Register(r, d)
		// [R1] comments.Register(r, d)
		// [R1] search.Register(r, d)
		// [R1] notifications.Register(r, d)
		// [R1] moderation.Register(r, d)
		// [R2] events.Register(r, d)
		// [R2] teamups.Register(r, d)
		// [R3] chat.Register(r, d)
		// [R4] discovery.Register(r, d)
		// [R5] adult.Register(r, d) // gated by FEATURE_ADULT_DATING

		// Protected routes go through RequireAuth; with() shares the parent
		// routing tree, so adding routes here cannot shadow public ones.
		protected := r.With(shared.RequireAuth(d))
		_ = protected // reserved for release 1 registration
		// [R1] auth.RegisterProtected(protected, d)
		// [R1] users.RegisterProtected(protected, d)
		// [R1] posts.RegisterProtected(protected, d)
		// [R1] moderation.RegisterProtected(protected, d)

		r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
			shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Endpoint not found.")
		})
		r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
			shared.WriteError(w, http.StatusMethodNotAllowed, shared.CodeMethodNotAllowed, "Method not allowed.")
		})
	})

	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Endpoint not found.")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		shared.WriteError(w, http.StatusMethodNotAllowed, shared.CodeMethodNotAllowed, "Method not allowed.")
	})

	return r
}
