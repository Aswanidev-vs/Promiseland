package adult

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

type likeInput struct {
	Status string `json:"status"`
}

// handleLike records a like or pass. A match is created only after a positive
// action from BOTH users, inside one transaction. No public listing is involved.
func (m *module) handleLike(w http.ResponseWriter, r *http.Request) {
	actorID, ok := m.authorize(w, r)
	if !ok {
		return
	}
	targetID, valid := positiveURLID(r, "userId")
	if !valid {
		writeValidation(w, map[string]string{"userId": "must be a positive integer user id"})
		return
	}
	var body likeInput
	if !shared.DecodeJSON(w, r, &body) {
		return
	}
	status := strings.ToLower(strings.TrimSpace(body.Status))
	if status != "like" && status != "pass" {
		writeValidation(w, map[string]string{"status": "must be like or pass"})
		return
	}
	if targetID == actorID {
		writeValidation(w, map[string]string{"userId": "you cannot like your own profile"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	// The actor must have opted in too: ordinary fandom activity never implies
	// romantic or dating intent.
	_, found, err := m.loadProfile(ctx, actorID)
	if err != nil {
		internalError(w, m.d, "load actor adult profile", err)
		return
	}
	if !found {
		shared.WriteError(w, http.StatusForbidden, codeProfileMissing,
			"Opt in with an adults-only profile before using this area.")
		return
	}

	// A 404 is used for missing, non-opted-in and blocked targets so a user
	// cannot probe another account's participation in this feature.
	visible, err := m.visibleTarget(ctx, targetID)
	if err != nil {
		internalError(w, m.d, "check adult target", err)
		return
	}
	if !visible {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Profile not found.")
		return
	}
	blocked, err := blockedEitherWay(ctx, m.d.DB, actorID, targetID)
	if err != nil {
		internalError(w, m.d, "check adult blocks", err)
		return
	}
	if blocked {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Profile not found.")
		return
	}

	// Inspect the chat schema BEFORE opening the write transaction: the SQLite
	// pool has one connection, so querying the DB while the tx holds it would
	// block until timeout.
	schema := loadConversationSchema(ctx, m.d.DB)

	tx, err := m.d.DB.BeginTx(ctx, nil)
	if err != nil {
		internalError(w, m.d, "begin adult like", err)
		return
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO adult_likes (actor_id, target_id, status, created_at)
		VALUES (?, ?, ?, datetime('now'))
		ON CONFLICT(actor_id, target_id) DO UPDATE SET
			status = excluded.status, created_at = datetime('now')`,
		actorID, targetID, status); err != nil {
		internalError(w, m.d, "save adult like", err)
		return
	}

	if status == "like" {
		var reciprocal int
		err := tx.QueryRowContext(ctx, `
			SELECT 1 FROM adult_likes
			WHERE actor_id = ? AND target_id = ? AND status = 'like' LIMIT 1`,
			targetID, actorID).Scan(&reciprocal)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			internalError(w, m.d, "check mutual adult like", err)
			return
		}
		if err == nil {
			if err := m.createMatch(ctx, tx, schema, actorID, targetID); err != nil {
				internalError(w, m.d, "create adult match", err)
				return
			}
		}
	}
	if err := tx.Commit(); err != nil {
		internalError(w, m.d, "commit adult like", err)
		return
	}

	var like AdultLike
	var createdAt string
	if err := m.d.DB.QueryRowContext(ctx, `
		SELECT actor_id, target_id, status, created_at
		FROM adult_likes WHERE actor_id = ? AND target_id = ?`, actorID, targetID).
		Scan(&like.ActorID, &like.TargetID, &like.Status, &createdAt); err != nil {
		internalError(w, m.d, "reload adult like", err)
		return
	}
	like.CreatedAt = createdAt
	shared.WriteJSON(w, http.StatusOK, like)
}

// visibleTarget enforces independent opt-in: only adults with an 'opt_in'
// profile can be liked, and no listing endpoint exposes them.
func (m *module) visibleTarget(ctx context.Context, targetID int64) (bool, error) {
	var one int
	err := m.d.DB.QueryRowContext(ctx, `
		SELECT 1 FROM adult_profiles p
		JOIN users u ON u.id = p.user_id
		WHERE p.user_id = ? AND p.visibility = 'opt_in' AND u.is_active = 1
		LIMIT 1`, targetID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// createMatch inserts the canonical match row and opens (or reuses) the direct
// conversation for the pair. If the R3 chat schema differs or is absent, the
// match is still created and conversation_id stays null.
func (m *module) createMatch(ctx context.Context, tx *sql.Tx, schema conversationSchema, actorID, targetID int64) error {
	userA, userB := canonicalPair(actorID, targetID)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO adult_matches (user_a, user_b, created_at, unmatched_at)
		VALUES (?, ?, datetime('now'), NULL)
		ON CONFLICT(user_a, user_b) DO UPDATE SET
			created_at = datetime('now'), unmatched_at = NULL`, userA, userB); err != nil {
		return err
	}

	if !schema.usable() {
		if m.d.Log != nil {
			m.d.Log.Warn("adult: chat schema unavailable; match created without conversation",
				"user_a", userA, "user_b", userB)
		}
		return nil
	}
	key := pairKey(userA, userB)

	var conversationID int64
	err := tx.QueryRowContext(ctx,
		`SELECT id FROM conversations WHERE pair_key = ? LIMIT 1`, key).Scan(&conversationID)
	if errors.Is(err, sql.ErrNoRows) {
		result, insertErr := tx.ExecContext(ctx, `
			INSERT INTO conversations (type, name, created_by, pair_key, created_at)
			VALUES ('direct', '', ?, ?, datetime('now'))`, actorID, key)
		if insertErr != nil {
			// A concurrent transaction may have created the thread first.
			if lookupErr := tx.QueryRowContext(ctx,
				`SELECT id FROM conversations WHERE pair_key = ? LIMIT 1`, key).Scan(&conversationID); lookupErr == nil {
				return ensureConversationMembers(ctx, tx, conversationID, userA, userB)
			}
			return insertErr
		}
		conversationID, _ = result.LastInsertId()
	} else if err != nil {
		return err
	}
	return ensureConversationMembers(ctx, tx, conversationID, userA, userB)
}

func ensureConversationMembers(ctx context.Context, tx *sql.Tx, conversationID, a, b int64) error {
	for _, userID := range []int64{a, b} {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO conversation_members (conversation_id, user_id, joined_at)
			VALUES (?, ?, datetime('now'))`, conversationID, userID); err != nil {
			return err
		}
	}
	return nil
}
