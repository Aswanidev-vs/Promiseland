package shape

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// Loader reads posts from SQLite and fills the R1 Post wire shape. It is used
// by both feed and search so their Post JSON is identical.
type Loader struct {
	DB *sql.DB
	S  *Schema
}

type baseRow struct {
	id                                      int64
	pType, title, body, linkURL             string
	spoiler                                 bool
	spoilerNote                             string
	season, episode, voteScore              int
	createdAt, updatedAt, editedAt          string
	authorID                                int64
	authorUsername, authorDisplayName       string
	authorAvatar                            sql.NullString
	communityID                             sql.NullInt64
	cName, cSlug, cDescription, cVisibility string
	cIcon, cBanner                          sql.NullString
	cCreatedAt                              string
	cMemberCount, cPostCount                int
	cRole                                   sql.NullString
	cIsMember                               bool
}

// baseSelect builds the single SELECT joining posts, authors and communities.
func (l *Loader) baseSelect(viewerID int64) (string, []any) {
	s := l.S
	viewer := strconv.FormatInt(viewerID, 10)

	username := "''"
	switch {
	case s.HasCol("users", "username"):
		username = "COALESCE(NULLIF(u.username, ''), u.display_name, '')"
	case s.HasCol("user_profiles", "username"):
		username = "COALESCE(up.username, '')"
	}

	displayName := s.NullableText("users", "u", "display_name")
	if !s.HasCol("users", "display_name") {
		displayName = s.NullableText("user_profiles", "up", "display_name")
	}
	avatar := "CAST(NULL AS TEXT)"
	if s.HasCol("user_profiles", "avatar_url") {
		avatar = "up.avatar_url"
	} else if s.HasCol("users", "avatar_url") {
		avatar = "u.avatar_url"
	}

	pType := s.NullableText("posts", "p", "type")
	if !s.HasCol("posts", "type") {
		pType = "'text'"
	}
	spoiler := "0"
	if c := s.Col("posts", "spoiler", "spoiler_flag", "is_spoiler"); c != "" {
		spoiler = "CASE WHEN COALESCE(p." + c + ", 0) != 0 THEN 1 ELSE 0 END"
	}
	communityID := "CAST(NULL AS INTEGER)"
	if s.HasCol("posts", "community_id") {
		communityID = "p.community_id"
	}
	cVisibility := "'public'"
	if s.HasCol("communities", "visibility") {
		cVisibility = "COALESCE(c.visibility, 'public')"
	}

	updatedAt := s.NullableText("posts", "p", "updated_at")
	if s.HasCol("posts", "updated_at") && s.HasCol("posts", "created_at") {
		updatedAt = "COALESCE(CAST(p.updated_at AS TEXT), CAST(p.created_at AS TEXT))"
	}

	// Community membership uses the table created by the communities migration.
	membership := s.MembershipTable()
	active := "1 = 1"
	if membership != "" {
		active = s.ActiveMembership("m")
	}
	memberCount, postCount := "0", "0"
	role, isMember := "CAST(NULL AS TEXT)", "0"
	if membership != "" {
		memberCount = fmt.Sprintf(`(SELECT COUNT(*) FROM %s m WHERE m.community_id = c.id AND %s)`, membership, active)
		if s.HasCol(membership, "user_id") && s.HasCol(membership, "role") {
			role = fmt.Sprintf(`(SELECT m.role FROM %s m
				WHERE m.community_id = c.id AND m.user_id = %s AND %s LIMIT 1)`, membership, viewer, active)
		}
		if s.HasCol(membership, "user_id") {
			isMember = fmt.Sprintf(`CASE WHEN EXISTS (
				SELECT 1 FROM %s m WHERE m.community_id = c.id AND m.user_id = %s AND %s
			) THEN 1 ELSE 0 END`, membership, viewer, active)
		}
	}
	if s.HasCol("communities", "owner_id") {
		ownerRole := fmt.Sprintf(`CASE WHEN c.owner_id = %s THEN 'owner' END`, viewer)
		if role != "CAST(NULL AS TEXT)" {
			role = "COALESCE(" + role + ", " + ownerRole + ")"
		} else {
			role = ownerRole
		}
		isMember = fmt.Sprintf(`CASE WHEN (%s) = 1 OR c.owner_id = %s THEN 1 ELSE 0 END`, isMember, viewer)
	}
	if s.HasCol("posts", "community_id") {
		postCount = "(SELECT COUNT(*) FROM posts pc WHERE pc.community_id = c.id AND " + l.notDeleted("pc") + ")"
	}

	selectSQL := fmt.Sprintf(`
		SELECT p.id,
			%s AS p_type,
			%s AS title,
			%s AS body,
			%s AS link_url,
			%s AS spoiler,
			%s AS spoiler_note,
			%s AS season,
			%s AS episode,
			%s AS vote_score,
			%s AS created_at,
			%s AS updated_at,
			%s AS edited_at,
			u.id AS author_id,
			%s AS author_username,
			%s AS author_display_name,
			%s AS author_avatar,
			%s AS community_id,
			%s AS c_name,
			%s AS c_slug,
			%s AS c_description,
			%s AS c_visibility,
			%s AS c_icon,
			%s AS c_banner,
			%s AS c_created_at,
			%s AS c_member_count,
			%s AS c_post_count,
			%s AS c_role,
			%s AS c_is_member
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN user_profiles up ON up.user_id = u.id
		LEFT JOIN communities c ON c.id = p.community_id
		WHERE 1 = 1 AND %s`,
		pType,
		s.NullableText("posts", "p", "title"),
		s.NullableText("posts", "p", "body"),
		s.NullableText("posts", "p", "link_url"),
		spoiler,
		s.NullableText("posts", "p", "spoiler_note"),
		s.Int("posts", "p", "season"),
		s.Int("posts", "p", "episode"),
		s.Int("posts", "p", "vote_score"),
		s.NullableText("posts", "p", "created_at"),
		updatedAt,
		s.NullableText("posts", "p", "edited_at"),
		username,
		displayName,
		avatar,
		communityID,
		s.NullableText("communities", "c", "name"),
		s.NullableText("communities", "c", "slug"),
		s.NullableText("communities", "c", "description"),
		cVisibility,
		nullable("communities", "icon_url", "c.icon_url", s),
		nullable("communities", "banner_url", "c.banner_url", s),
		s.NullableText("communities", "c", "created_at"),
		memberCount,
		postCount,
		role,
		isMember,
		l.notDeleted("p"),
	)
	return selectSQL, nil
}

func (l *Loader) notDeleted(alias string) string {
	if l.S.HasCol("posts", "deleted_at") {
		return alias + ".deleted_at IS NULL"
	}
	return "1 = 1"
}

func nullable(table, col, expr string, s *Schema) string {
	if s.HasCol(table, col) {
		return expr
	}
	return "CAST(NULL AS TEXT)"
}

// Load returns posts matching extraWhere (a trusted SQL fragment beginning
// with AND, with placeholders bound by extraArgs), newest first.
func (l *Loader) Load(ctx context.Context, extraWhere string, extraArgs []any, viewerID int64, limit int) ([]Post, error) {
	if l == nil || l.DB == nil || l.S == nil || !l.S.HasTable("posts") {
		return []Post{}, nil
	}
	base, _ := l.baseSelect(viewerID)
	extra := strings.TrimSpace(extraWhere)
	if extra != "" {
		lower := strings.ToLower(extra)
		if !strings.HasPrefix(lower, "and ") && !strings.HasPrefix(lower, "or ") {
			extra = "AND " + extra
		}
	}
	query := base + " " + extra + " ORDER BY p.id DESC LIMIT " + strconv.Itoa(limit)
	rows, err := l.DB.QueryContext(ctx, query, extraArgs...)
	if err != nil {
		return nil, fmt.Errorf("select posts: %w", err)
	}
	defer rows.Close()

	posts := make([]Post, 0, limit)
	ids := make([]int64, 0, limit)
	byID := map[int64]*Post{}
	for rows.Next() {
		var r baseRow
		err := rows.Scan(
			&r.id, &r.pType, &r.title, &r.body, &r.linkURL, &r.spoiler,
			&r.spoilerNote, &r.season, &r.episode, &r.voteScore, &r.createdAt, &r.updatedAt, &r.editedAt,
			&r.authorID, &r.authorUsername, &r.authorDisplayName, &r.authorAvatar,
			&r.communityID, &r.cName, &r.cSlug, &r.cDescription, &r.cVisibility,
			&r.cIcon, &r.cBanner, &r.cCreatedAt, &r.cMemberCount, &r.cPostCount,
			&r.cRole, &r.cIsMember,
		)
		if err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		p := newPost()
		p.ID = r.id
		p.Type = r.pType
		p.Title = r.title
		p.Body = r.body
		p.LinkURL = r.linkURL
		p.Spoiler = r.spoiler
		p.SpoilerNote = r.spoilerNote
		p.Season = r.season
		p.Episode = r.episode
		p.VoteScore = r.voteScore
		p.CreatedAt = r.createdAt
		p.UpdatedAt = r.updatedAt
		p.EditedAt = r.editedAt
		p.Author = UserSummary{
			ID:          r.authorID,
			Username:    r.authorUsername,
			DisplayName: r.authorDisplayName,
			AvatarURL:   nullStringPtr(r.authorAvatar),
		}
		if r.communityID.Valid {
			p.Community = &Community{
				ID:          r.communityID.Int64,
				Name:        r.cName,
				Slug:        r.cSlug,
				Description: r.cDescription,
				IconURL:     nullStringPtr(r.cIcon),
				BannerURL:   nullStringPtr(r.cBanner),
				Visibility:  r.cVisibility,
				MemberCount: r.cMemberCount,
				PostCount:   r.cPostCount,
				Role:        nullStringPtr(r.cRole),
				IsMember:    r.cIsMember,
				CreatedAt:   r.cCreatedAt,
			}
		}
		posts = append(posts, p)
		ids = append(ids, r.id)
		byID[r.id] = &posts[len(posts)-1]
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate posts: %w", err)
	}
	if len(ids) == 0 {
		return posts, nil
	}

	l.fillMedia(ctx, byID, ids)
	l.fillAnime(ctx, byID, ids)
	l.fillTags(ctx, byID, ids)
	l.fillVoteData(ctx, byID, ids, viewerID)
	l.fillCommentCounts(ctx, byID, ids)
	l.fillBookmarks(ctx, byID, ids, viewerID)
	l.fillPolls(ctx, byID, ids, viewerID)
	return posts, nil
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func placeholders(n int) string {
	if n <= 0 {
		return "NULL"
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func argsFromIDs(ids []int64) []any {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return args
}

func (l *Loader) fillMedia(ctx context.Context, byID map[int64]*Post, ids []int64) {
	s := l.S
	if !s.HasTable("post_media") {
		return
	}
	in := placeholders(len(ids))
	var q string
	if s.HasCol("post_media", "media_id") && s.HasTable("media") {
		// R1 schema: post_media is a link table and media stores the metadata.
		keyCol := s.Col("media", "storage_key", "url", "storage_url")
		urlExpr := "''"
		if keyCol != "" {
			urlExpr = "'/media/' || COALESCE(m." + keyCol + ", '')"
		}
		q = fmt.Sprintf(`SELECT pm.post_id, m.id, %s,
			COALESCE(m.mime_type, ''), COALESCE(m.size_bytes, 0),
			COALESCE(m.width, 0), COALESCE(m.height, 0)
			FROM post_media pm JOIN media m ON m.id = pm.media_id
			WHERE pm.post_id IN (%s) ORDER BY m.id ASC`, urlExpr, in)
	} else if urlCol := s.Col("post_media", "url", "storage_key", "media_url"); urlCol != "" {
		mime, size, width, height := "''", "0", "0", "0"
		if s.HasCol("post_media", "mime_type") {
			mime = "COALESCE(pm.mime_type, '')"
		}
		if s.HasCol("post_media", "size_bytes") {
			size = "COALESCE(pm.size_bytes, 0)"
		}
		if s.HasCol("post_media", "width") {
			width = "COALESCE(pm.width, 0)"
		}
		if s.HasCol("post_media", "height") {
			height = "COALESCE(pm.height, 0)"
		}
		q = fmt.Sprintf(`SELECT pm.post_id, pm.id, COALESCE(pm.%s, ''), %s, %s, %s, %s
			FROM post_media pm WHERE pm.post_id IN (%s) ORDER BY pm.id ASC`,
			urlCol, mime, size, width, height, in)
	} else {
		return
	}
	rows, err := l.DB.QueryContext(ctx, q, argsFromIDs(ids)...)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var postID, id int64
		var url, mimeType string
		var sizeBytes int64
		var w, h int
		if rows.Scan(&postID, &id, &url, &mimeType, &sizeBytes, &w, &h) != nil {
			continue
		}
		if p := byID[postID]; p != nil {
			p.Media = append(p.Media, Media{ID: id, URL: url, MimeType: mimeType, SizeBytes: sizeBytes, Width: w, Height: h})
		}
	}
}

func (l *Loader) fillAnime(ctx context.Context, byID map[int64]*Post, ids []int64) {
	s := l.S
	linkTable := ""
	for _, candidate := range []string{"post_anime", "posts_anime", "anime_posts"} {
		if s.HasTable(candidate) {
			linkTable = candidate
			break
		}
	}
	animeTable := ""
	for _, candidate := range []string{"anime_titles", "anime"} {
		if s.HasTable(candidate) {
			animeTable = candidate
			break
		}
	}
	if linkTable == "" || animeTable == "" {
		return
	}
	postCol := s.Col(linkTable, "post_id")
	animeCol := s.Col(linkTable, "anime_title_id", "anime_id")
	if postCol == "" || animeCol == "" {
		return
	}
	externalID := "''"
	if s.HasCol(animeTable, "external_id") {
		externalID = "COALESCE(CAST(a.external_id AS TEXT), '')"
	}
	title := "''"
	if s.HasCol(animeTable, "title") {
		title = "COALESCE(a.title, '')"
	}
	alt := "CAST(NULL AS TEXT)"
	if s.HasCol(animeTable, "alternate_title") {
		alt = "a.alternate_title"
	}
	format := "''"
	if c := s.Col(animeTable, "format", "media_type"); c != "" {
		format = "COALESCE(a." + c + ", '')"
	}
	q := fmt.Sprintf(`SELECT l.post_id, a.id, %s, %s, %s, %s
		FROM %s l JOIN %s a ON a.id = l.%s
		WHERE l.%s IN (%s) ORDER BY a.id ASC`,
		externalID, title, alt, format, linkTable, animeTable, animeCol, postCol, placeholders(len(ids)))
	rows, err := l.DB.QueryContext(ctx, q, argsFromIDs(ids)...)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var postID, id int64
		var external, titleText string
		var altTitle sql.NullString
		var formatText string
		if rows.Scan(&postID, &id, &external, &titleText, &altTitle, &formatText) != nil {
			continue
		}
		if p := byID[postID]; p != nil {
			p.Anime = append(p.Anime, Anime{ID: id, ExternalID: external, Title: titleText, AlternateTitle: nullStringPtr(altTitle), Format: formatText})
		}
	}
}

func (l *Loader) fillTags(ctx context.Context, byID map[int64]*Post, ids []int64) {
	s := l.S
	if !s.HasTable("post_tags") {
		return
	}
	postCol := s.Col("post_tags", "post_id")
	tagCol := s.Col("post_tags", "tag", "tag_name", "name")
	if postCol == "" || tagCol == "" {
		return
	}
	q := fmt.Sprintf(`SELECT %s, COALESCE(%s, '') FROM post_tags WHERE %s IN (%s) ORDER BY %s ASC`,
		postCol, tagCol, postCol, placeholders(len(ids)), tagCol)
	rows, err := l.DB.QueryContext(ctx, q, argsFromIDs(ids)...)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		var tag string
		if rows.Scan(&postID, &tag) == nil {
			if p := byID[postID]; p != nil && tag != "" {
				p.Tags = append(p.Tags, tag)
			}
		}
	}
}

func (l *Loader) fillVoteData(ctx context.Context, byID map[int64]*Post, ids []int64, viewerID int64) {
	s := l.S
	table := ""
	for _, candidate := range []string{"votes", "post_votes"} {
		if s.HasTable(candidate) {
			table = candidate
			break
		}
	}
	if table == "" {
		return
	}
	postCol := s.Col(table, "post_id")
	valueCol := s.Col(table, "value", "vote", "score")
	if postCol == "" || valueCol == "" {
		return
	}
	q := fmt.Sprintf(`SELECT %s, COALESCE(SUM(%s), 0) FROM %s WHERE %s IN (%s) GROUP BY %s`,
		postCol, valueCol, table, postCol, placeholders(len(ids)), postCol)
	rows, err := l.DB.QueryContext(ctx, q, argsFromIDs(ids)...)
	if err != nil {
		return
	}
	for rows.Next() {
		var postID int64
		var score int
		if rows.Scan(&postID, &score) == nil {
			if p := byID[postID]; p != nil {
				p.VoteScore = score
			}
		}
	}
	_ = rows.Close()
	if viewerID <= 0 || !s.HasCol(table, "user_id") {
		return
	}
	args := append([]any{viewerID}, argsFromIDs(ids)...)
	q = fmt.Sprintf(`SELECT %s, COALESCE(%s, 0) FROM %s WHERE user_id = ? AND %s IN (%s)`,
		postCol, valueCol, table, postCol, placeholders(len(ids)))
	rows, err = l.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		var vote int
		if rows.Scan(&postID, &vote) == nil {
			if p := byID[postID]; p != nil {
				p.MyVote = vote
			}
		}
	}
}

func (l *Loader) fillCommentCounts(ctx context.Context, byID map[int64]*Post, ids []int64) {
	s := l.S
	if !s.HasTable("comments") || !s.HasCol("comments", "post_id") {
		return
	}
	filter := ""
	if s.HasCol("comments", "deleted_at") {
		filter = " AND deleted_at IS NULL"
	} else if s.HasCol("comments", "is_deleted") {
		filter = " AND COALESCE(is_deleted, 0) = 0"
	}
	q := fmt.Sprintf(`SELECT post_id, COUNT(*) FROM comments WHERE post_id IN (%s)%s GROUP BY post_id`,
		placeholders(len(ids)), filter)
	rows, err := l.DB.QueryContext(ctx, q, argsFromIDs(ids)...)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		var count int
		if rows.Scan(&postID, &count) == nil {
			if p := byID[postID]; p != nil {
				p.CommentCount = count
			}
		}
	}
}

func (l *Loader) fillBookmarks(ctx context.Context, byID map[int64]*Post, ids []int64, viewerID int64) {
	s := l.S
	if viewerID <= 0 || !s.HasTable("bookmarks") || !s.HasCol("bookmarks", "post_id") || !s.HasCol("bookmarks", "user_id") {
		return
	}
	args := append([]any{viewerID}, argsFromIDs(ids)...)
	q := fmt.Sprintf(`SELECT post_id FROM bookmarks WHERE user_id = ? AND post_id IN (%s)`, placeholders(len(ids)))
	rows, err := l.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		if rows.Scan(&postID) == nil {
			if p := byID[postID]; p != nil {
				p.Bookmarked = true
			}
		}
	}
}

func (l *Loader) fillPolls(ctx context.Context, byID map[int64]*Post, ids []int64, viewerID int64) {
	s := l.S
	if !s.HasTable("poll_options") || !s.HasCol("poll_options", "post_id") {
		return
	}
	// R1 stores the question on the post itself (poll_question, falling back
	// to the title) and the options in poll_options.
	questionExpr := "COALESCE(p.title, '')"
	if s.HasCol("posts", "poll_question") {
		questionExpr = "COALESCE(NULLIF(p.poll_question, ''), p.title, '')"
	}
	voteCount := "0"
	if s.HasTable("poll_votes") && s.HasCol("poll_votes", "option_id") {
		voteCount = "(SELECT COUNT(*) FROM poll_votes v WHERE v.option_id = o.id)"
	}
	order := "p.id, o.id"
	if s.HasCol("poll_options", "sort_order") {
		order = "p.id, o.sort_order, o.id"
	}
	q := fmt.Sprintf(`SELECT p.id, %s, o.id, COALESCE(o.text, ''), %s
		FROM posts p JOIN poll_options o ON o.post_id = p.id
		WHERE p.id IN (%s) ORDER BY %s`, questionExpr, voteCount, placeholders(len(ids)), order)
	rows, err := l.DB.QueryContext(ctx, q, argsFromIDs(ids)...)
	if err != nil {
		return
	}
	polls := map[int64]*Poll{}
	for rows.Next() {
		var postID, optID int64
		var question, text string
		var votes int
		if rows.Scan(&postID, &question, &optID, &text, &votes) != nil {
			continue
		}
		poll := polls[postID]
		if poll == nil {
			poll = &Poll{Question: question, Options: []PollOption{}}
			polls[postID] = poll
		}
		poll.Options = append(poll.Options, PollOption{ID: optID, Text: text, Votes: votes})
		poll.TotalVotes += votes
	}
	_ = rows.Close()
	if s.HasTable("poll_votes") && viewerID > 0 && s.HasCol("poll_votes", "user_id") {
		optCol := s.Col("poll_votes", "option_id", "poll_option_id")
		if optCol != "" {
			optionIDs := make([]any, 0)
			for _, poll := range polls {
				for _, opt := range poll.Options {
					optionIDs = append(optionIDs, opt.ID)
				}
			}
			if len(optionIDs) > 0 {
				args := append([]any{viewerID}, optionIDs...)
				q := fmt.Sprintf(`SELECT %s FROM poll_votes WHERE user_id = ? AND %s IN (%s)`, optCol, optCol, placeholders(len(optionIDs)))
				vrows, err := l.DB.QueryContext(ctx, q, args...)
				if err == nil {
					chosen := map[int64]bool{}
					for vrows.Next() {
						var optID int64
						if vrows.Scan(&optID) == nil {
							chosen[optID] = true
						}
					}
					_ = vrows.Close()
					for _, poll := range polls {
						for _, opt := range poll.Options {
							if chosen[opt.ID] {
								id := opt.ID
								poll.MyVote = &id
								break
							}
						}
					}
				}
			}
		}
	}
	for postID, poll := range polls {
		if p := byID[postID]; p != nil && p.Type == "poll" {
			p.Poll = poll
		}
	}
}
