// Package api assembles the HTTP router. Domain modules are mounted here;
// each exposes Register(r chi.Router, d *shared.Deps) and, where it has
// authenticated operations, RegisterProtected as well.
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/adult"
	"github.com/Aswanidev-vs/promiseland/backend/internal/anime"
	"github.com/Aswanidev-vs/promiseland/backend/internal/auth"
	"github.com/Aswanidev-vs/promiseland/backend/internal/chat"
	"github.com/Aswanidev-vs/promiseland/backend/internal/communities"
	"github.com/Aswanidev-vs/promiseland/backend/internal/creators"
	"github.com/Aswanidev-vs/promiseland/backend/internal/discovery"
	"github.com/Aswanidev-vs/promiseland/backend/internal/events"
	"github.com/Aswanidev-vs/promiseland/backend/internal/feed"
	"github.com/Aswanidev-vs/promiseland/backend/internal/media"
	"github.com/Aswanidev-vs/promiseland/backend/internal/moderation"
	"github.com/Aswanidev-vs/promiseland/backend/internal/notifications"
	"github.com/Aswanidev-vs/promiseland/backend/internal/posts"
	"github.com/Aswanidev-vs/promiseland/backend/internal/realtime"
	"github.com/Aswanidev-vs/promiseland/backend/internal/search"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/Aswanidev-vs/promiseland/backend/internal/teamups"
	"github.com/Aswanidev-vs/promiseland/backend/internal/users"
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

	// Uploaded images are served from a persistent disk directory, never from
	// SQLite (PRD §10.6). Mounted before /api so it cannot be shadowed.
	media.RegisterMedia(r, d)

	// Authenticated WebSocket endpoint. Chat handlers publish events through
	// the same singleton; chat.Register installs its access checker.
	realtime.Default().SetDeps(d)
	r.Handle("/ws", realtime.Default().Handler())

	r.Route("/api/v1", func(r chi.Router) {
		// Public routes. Handlers that need to know the viewer call
		// d.SessionUserID(r) directly for privacy filtering.
		auth.Register(r, d)
		users.Register(r, d)
		anime.Register(r, d)
		communities.Register(r, d)
		posts.Register(r, d)
		feed.Register(r, d)
		search.Register(r, d)
		events.Register(r, d)
		creators.Register(r, d)
		teamups.Register(r, d)
		chat.Register(r, d)
		// R5 is a gated scaffold: every handler 404s unless both the feature
		// flag and a real age-assurance provider are configured.
		adult.Register(r, d)

		// Authenticated routes. With() shares chi's route tree, so these
		// cannot shadow public routes; auth is also enforced in handlers.
		protected := r.With(shared.RequireAuth(d))
		auth.RegisterProtected(protected, d)
		users.RegisterProtected(protected, d)
		communities.RegisterProtected(protected, d)
		posts.RegisterProtected(protected, d)
		media.RegisterProtected(protected, d)
		notifications.RegisterProtected(protected, d)
		moderation.RegisterProtected(protected, d)
		events.RegisterProtected(protected, d)
		creators.RegisterProtected(protected, d)
		teamups.RegisterProtected(protected, d)
		chat.RegisterProtected(protected, d)
		discovery.RegisterProtected(protected, d)

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
