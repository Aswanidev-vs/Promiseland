// Package adult implements the Release 5 optional, adults-only scaffold.
//
// SAFETY FIRST (PRD §7.14): this module is disabled by default. When
// FEATURE_ADULT_DATING is false every /adult/* route answers 404 and reveals
// no eligibility data. When it is enabled, access is still denied by the
// default AgeAssurance implementation. A real age-assurance provider, legal
// review, safety policy and moderation capacity are required before launch.
//
// There is no public discovery listing: adult profiles are private and are
// only reachable by their owner or by a user who has mutually matched.
package adult

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

const (
	codeEligibility    = "adult_eligibility_required"
	codeValidation     = "validation_error"
	codeProfileMissing = "adult_profile_required"
	codeAlreadyReport  = "already_reported"

	disabledMessage = "This feature is not available."
)

// module holds the request-scoped dependencies. Handlers resolve and verify
// the viewer themselves, so both registration entry points are safe.
type module struct {
	d         *shared.Deps
	assurance AgeAssurance
}

func newModule(d *shared.Deps) *module {
	m := &module{d: d}
	m.assurance = newAgeAssurance(d)
	return m
}

// Register mounts the /adult/* routes on a public router. Every handler
// resolves the viewer from shared.UserID (auth middleware) with a session
// cookie fallback, so the routes are protected even without RequireAuth.
func Register(r chi.Router, d *shared.Deps) { mount(r, d) }

// RegisterProtected mounts the same routes on a router already wrapped with
// shared.RequireAuth. Eligibility and the feature gate are re-checked by the
// handlers themselves, so both entry points may be wired safely.
func RegisterProtected(r chi.Router, d *shared.Deps) { mount(r, d) }

func mount(r chi.Router, d *shared.Deps) {
	if r == nil || d == nil {
		return
	}
	m := newModule(d)

	// status answers its own gate so it can return the exact safe disabled
	// shape; every other handler gets the generic 404 gate wrapper.
	r.Get("/adult/status", m.handleStatus)
	r.Get("/adult/profile", m.gate(m.handleGetProfile))
	r.Put("/adult/profile", m.gate(m.handlePutProfile))
	r.Post("/adult/like/{userId}", m.gate(m.handleLike))
	r.Get("/adult/matches", m.gate(m.handleMatches))
	r.Post("/adult/matches/{id}/unmatch", m.gate(m.handleUnmatch))
	r.Post("/adult/reports", m.gate(m.handleReport))
	r.Post("/adult/dev/verify", m.gate(m.handleDevVerify))

	// Unknown /adult/* paths stay inside the gate too: no route in this
	// namespace can be used to probe the feature while it is disabled.
	r.Handle("/adult/*", m.gate(m.handleCatchAll))
}

func (m *module) enabled() bool {
	return m != nil && m.d != nil && m.d.Config.FeatureAdultDating
}

func (m *module) gate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled() {
			shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, disabledMessage)
			return
		}
		next(w, r)
	}
}

func (m *module) handleCatchAll(w http.ResponseWriter, _ *http.Request) {
	shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Endpoint not found.")
}

func (m *module) viewer(r *http.Request) (int64, bool) {
	if id, ok := shared.UserID(r); ok {
		return id, true
	}
	if m == nil || m.d == nil {
		return 0, false
	}
	return m.d.SessionUserID(r)
}

// authorize enforces authentication, database availability and age assurance
// in that order. It never infers eligibility from ordinary profile interests.
func (m *module) authorize(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := m.viewer(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return 0, false
	}
	if m.d.DB == nil {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable,
			"The adults-only area is temporarily unavailable.")
		return 0, false
	}
	eligible, _ := m.assurance.Eligible(userID)
	if !eligible {
		shared.WriteError(w, http.StatusForbidden, codeEligibility,
			"Age assurance is required before you can use this area.")
		return 0, false
	}
	return userID, true
}

func internalError(w http.ResponseWriter, d *shared.Deps, action string, err error) {
	if d != nil && d.Log != nil {
		d.Log.Error("adult: "+action, "error", err)
	}
	shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError,
		"The request could not be completed.")
}

func writeValidation(w http.ResponseWriter, details map[string]string) {
	shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation,
		"Invalid adults-only request.", details)
}

// positiveURLID parses a positive integer URL parameter.
func positiveURLID(r *http.Request, param string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, param)), 10, 64)
	return id, err == nil && id > 0
}

// pairKey mirrors the existing chat module's stable unordered direct-chat key.
func pairKey(a, b int64) string {
	if a > b {
		a, b = b, a
	}
	return strconv.FormatInt(a, 10) + ":" + strconv.FormatInt(b, 10)
}

// canonicalPair orders a pair so the UNIQUE(user_a,user_b) constraint in
// adult_matches has a single representation per unordered pair.
func canonicalPair(a, b int64) (int64, int64) {
	if a > b {
		return b, a
	}
	return a, b
}
