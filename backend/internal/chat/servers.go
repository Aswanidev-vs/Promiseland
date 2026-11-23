package chat

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

type serverInput struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	IconURL     *string `json:"icon_url"`
}

type channelInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func validateServer(in serverInput) map[string]string {
	details := map[string]string{}
	if n := utf8.RuneCountInString(strings.TrimSpace(in.Name)); n < 3 || n > 60 {
		details["name"] = "Name must be between 3 and 60 characters."
	}
	slug := strings.TrimSpace(in.Slug)
	if slug != "" && !slugPattern.MatchString(slug) {
		details["slug"] = "Slug must be 3-40 lowercase letters, digits or hyphens."
	}
	if utf8.RuneCountInString(in.Description) > 2000 {
		details["description"] = "Description must be at most 2000 characters."
	}
	if in.IconURL != nil && utf8.RuneCountInString(*in.IconURL) > 2048 {
		details["icon_url"] = "Icon URL must be at most 2048 characters."
	}
	return details
}

func validateChannel(in channelInput) map[string]string {
	details := map[string]string{}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 40 {
		details["name"] = "Channel name must be between 1 and 40 characters."
	}
	if utf8.RuneCountInString(in.Description) > 500 {
		details["description"] = "Description must be at most 500 characters."
	}
	return details
}

// slugify derives a URL-safe slug from a display name for clients that only
// send {name, description} (the R3 frontend contract).
func slugify(name string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if utf8.RuneCountInString(slug) > 40 {
		slug = strings.Trim(slug[:40], "-")
	}
	if len(slug) < 3 {
		if slug == "" {
			slug = "chat"
		}
		for len(slug) < 3 {
			slug += "-"
		}
	}
	return slug
}

func randomSuffix() string {
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return strconv.FormatInt(int64(len(raw)), 10)
	}
	return hex.EncodeToString(raw[:])
}

// resolveSlug enforces the slug regex and uniqueness. Explicit slugs conflict;
// derived slugs are suffixed automatically so name-only clients always work.
func resolveSlug(ctx context.Context, tx *sql.Tx, in serverInput) (string, bool) {
	base := strings.TrimSpace(in.Slug)
	explicit := base != ""
	if !explicit {
		base = slugify(in.Name)
	}
	exists := func(candidate string) bool {
		var one int
		err := tx.QueryRowContext(ctx, `SELECT 1 FROM chat_servers WHERE slug = ? LIMIT 1`, candidate).Scan(&one)
		return err == nil
	}
	if explicit {
		if exists(base) {
			return "", true
		}
		return base, false
	}
	if !exists(base) {
		return base, false
	}
	for i := 2; i < 100; i++ {
		suffix := "-" + strconv.Itoa(i)
		root := base
		if len(root)+len(suffix) > 40 {
			root = strings.Trim(root[:40-len(suffix)], "-")
		}
		candidate := root + suffix
		if !exists(candidate) {
			return candidate, false
		}
	}
	for {
		candidate := "chat-" + randomSuffix()
		if !exists(candidate) {
			return candidate, false
		}
	}
}

func (m *module) createServer(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.requireUser(w, r)
	if !ok {
		return
	}
	var in serverInput
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	if details := validateServer(in); len(details) > 0 {
		writeValidation(w, details)
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.TrimSpace(in.Slug)

	tx, err := m.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeInternal(w, m, "chat: begin create server", err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	slug, taken := resolveSlug(r.Context(), tx, in)
	if taken {
		shared.WriteError(w, http.StatusConflict, codeSlugTaken, "That slug is already in use.")
		return
	}
	result, err := tx.ExecContext(r.Context(), `
		INSERT INTO chat_servers(owner_id, name, slug, description, icon_url, created_at)
		VALUES(?,?,?,?,?,datetime('now'))`, userID, in.Name, slug, in.Description, in.IconURL)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			shared.WriteError(w, http.StatusConflict, codeSlugTaken, "That slug is already in use.")
			return
		}
		writeInternal(w, m, "chat: insert server", err)
		return
	}
	serverID, _ := result.LastInsertId()
	if _, err := tx.ExecContext(r.Context(), `
		INSERT INTO chat_server_members(server_id, user_id, role, joined_at)
		VALUES(?,?,'owner',datetime('now'))`, serverID, userID); err != nil {
		writeInternal(w, m, "chat: insert owner membership", err)
		return
	}
	if _, err := tx.ExecContext(r.Context(), `
		INSERT INTO chat_channels(server_id, name, description, type, created_at)
		VALUES(?, 'general', 'General discussion', 'text', datetime('now'))`, serverID); err != nil {
		writeInternal(w, m, "chat: insert default channel", err)
		return
	}
	if err := tx.Commit(); err != nil {
		writeInternal(w, m, "chat: commit create server", err)
		return
	}
	server, err := m.loadServer(r.Context(), serverID, userID)
	if err != nil {
		writeInternal(w, m, "chat: load created server", err)
		return
	}
	shared.WriteJSON(w, http.StatusCreated, server)
}

func (m *module) listServers(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.requireUser(w, r)
	if !ok {
		return
	}
	rows, err := m.d.DB.QueryContext(r.Context(), `
		SELECT id FROM chat_servers ORDER BY id DESC LIMIT ?`, shared.MaxLimit)
	if err != nil {
		writeInternal(w, m, "chat: list servers", err)
		return
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	servers := make([]ChatServer, 0, len(ids))
	for _, id := range ids {
		server, err := m.loadServer(r.Context(), id, userID)
		if err == nil {
			servers = append(servers, server)
		}
	}
	shared.WriteJSON(w, http.StatusOK, servers)
}

func (m *module) joinServer(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.requireUser(w, r)
	if !ok {
		return
	}
	serverID, err := parseIDParam(r, "id")
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidID, "Invalid server id.")
		return
	}
	var exists int
	if err := m.d.DB.QueryRowContext(r.Context(), `SELECT 1 FROM chat_servers WHERE id = ?`, serverID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Server not found.")
		} else {
			writeInternal(w, m, "chat: lookup server", err)
		}
		return
	}
	_, err = m.d.DB.ExecContext(r.Context(), `
		INSERT OR IGNORE INTO chat_server_members(server_id, user_id, role, joined_at)
		VALUES(?,?,'member',datetime('now'))`, serverID, userID)
	if err != nil {
		writeInternal(w, m, "chat: join server", err)
		return
	}
	server, err := m.loadServer(r.Context(), serverID, userID)
	if err != nil {
		writeInternal(w, m, "chat: load joined server", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, server)
}

func (m *module) listChannels(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.requireUser(w, r)
	if !ok {
		return
	}
	serverID, err := parseIDParam(r, "id")
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidID, "Invalid server id.")
		return
	}
	var exists int
	if err := m.d.DB.QueryRowContext(r.Context(), `SELECT 1 FROM chat_servers WHERE id = ?`, serverID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Server not found.")
		} else {
			writeInternal(w, m, "chat: lookup server channels", err)
		}
		return
	}
	role, err := m.serverRole(r.Context(), serverID, userID)
	if err != nil {
		writeInternal(w, m, "chat: server role", err)
		return
	}
	if role == "" {
		writeForbidden(w, "Join this server before reading its channels.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, m.loadChannels(r.Context(), serverID, userID))
}

func (m *module) createChannel(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.requireUser(w, r)
	if !ok {
		return
	}
	serverID, err := parseIDParam(r, "id")
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidID, "Invalid server id.")
		return
	}
	var exists int
	if err := m.d.DB.QueryRowContext(r.Context(), `SELECT 1 FROM chat_servers WHERE id = ?`, serverID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Server not found.")
		} else {
			writeInternal(w, m, "chat: lookup server for channel", err)
		}
		return
	}
	role, err := m.serverRole(r.Context(), serverID, userID)
	if err != nil {
		writeInternal(w, m, "chat: server role", err)
		return
	}
	if role != "owner" && role != "moderator" {
		writeForbidden(w, "Only server staff can create channels.")
		return
	}
	var in channelInput
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	if details := validateChannel(in); len(details) > 0 {
		writeValidation(w, details)
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	result, err := m.d.DB.ExecContext(r.Context(), `
		INSERT INTO chat_channels(server_id, name, description, type, created_at)
		VALUES(?,?,?,'text',datetime('now'))`, serverID, in.Name, in.Description)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			shared.WriteError(w, http.StatusConflict, codeChannelExists, "A channel with that name already exists.")
		} else {
			writeInternal(w, m, "chat: insert channel", err)
		}
		return
	}
	channelID, _ := result.LastInsertId()
	channels := m.loadChannels(r.Context(), serverID, userID)
	for _, channel := range channels {
		if channel.ID == channelID {
			shared.WriteJSON(w, http.StatusCreated, channel)
			return
		}
	}
	shared.WriteJSON(w, http.StatusCreated, ChatChannel{
		ID: channelID, ServerID: serverID, Name: in.Name,
		Description: in.Description, Type: "text", LastMessage: nil, UnreadCount: 0,
	})
}

// serverIDForChannel resolves the owning server without loading the channel
// DTO (used by moderation checks in messages.go).
func (m *module) serverIDForChannel(ctx context.Context, channelID int64) (int64, error) {
	var serverID int64
	err := m.d.DB.QueryRowContext(ctx, `SELECT server_id FROM chat_channels WHERE id = ?`, channelID).Scan(&serverID)
	return serverID, err
}
