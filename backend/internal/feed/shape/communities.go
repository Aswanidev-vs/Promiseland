package shape

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// LoadCommunities builds the exact R1 Community wire shape for communities
// matching a trusted SQL fragment.
func LoadCommunities(ctx context.Context, db *sql.DB, s *Schema, where string, args []any, viewerID int64, limit int) ([]Community, error) {
	if db == nil || s == nil || !s.HasTable("communities") {
		return []Community{}, nil
	}
	viewer := strconv.FormatInt(viewerID, 10)

	memberCount, postCount := "0", "0"
	role, isMember := "CAST(NULL AS TEXT)", "0"
	membership := s.MembershipTable()
	if membership != "" {
		active := s.ActiveMembership("cmc")
		memberCount = fmt.Sprintf("(SELECT COUNT(*) FROM %s cmc WHERE cmc.community_id = c.id AND %s)", membership, active)
		if s.HasCol(membership, "user_id") && s.HasCol(membership, "role") {
			role = fmt.Sprintf(`(SELECT cmr.role FROM %s cmr
				WHERE cmr.community_id = c.id AND cmr.user_id = %s AND %s LIMIT 1)`,
				membership, viewer, s.ActiveMembership("cmr"))
		}
		if s.HasCol(membership, "user_id") {
			isMember = fmt.Sprintf(`CASE WHEN EXISTS (
				SELECT 1 FROM %s cmm WHERE cmm.community_id = c.id AND cmm.user_id = %s AND %s
			) THEN 1 ELSE 0 END`, membership, viewer, s.ActiveMembership("cmm"))
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
	if s.HasTable("posts") && s.HasCol("posts", "community_id") {
		deleted := "1 = 1"
		if s.HasCol("posts", "deleted_at") {
			deleted = "pc.deleted_at IS NULL"
		}
		postCount = fmt.Sprintf("(SELECT COUNT(*) FROM posts pc WHERE pc.community_id = c.id AND %s)", deleted)
	}

	visibility := "'public'"
	if s.HasCol("communities", "visibility") {
		visibility = "COALESCE(c.visibility, 'public')"
	}
	q := fmt.Sprintf(`SELECT c.id,
		%s, %s, %s, %s, %s, %s,
		%s, %s, %s, %s, %s
		FROM communities c
		WHERE 1 = 1 %s
		ORDER BY c.id DESC LIMIT %d`,
		s.Text("communities", "c", "name"),
		s.Text("communities", "c", "slug"),
		s.Text("communities", "c", "description"),
		nullable("communities", "icon_url", "c.icon_url", s),
		nullable("communities", "banner_url", "c.banner_url", s),
		visibility,
		s.Text("communities", "c", "created_at"),
		memberCount, postCount, role, isMember,
		where, limit,
	)
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("select communities: %w", err)
	}
	defer rows.Close()

	out := []Community{}
	for rows.Next() {
		var c Community
		var icon, banner sql.NullString
		var roleValue sql.NullString
		var isMemberInt int
		err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.Description, &icon, &banner,
			&c.Visibility, &c.CreatedAt, &c.MemberCount, &c.PostCount, &roleValue, &isMemberInt)
		if err != nil {
			return nil, fmt.Errorf("scan community: %w", err)
		}
		c.IconURL = nullStringPtr(icon)
		c.BannerURL = nullStringPtr(banner)
		c.Role = nullStringPtr(roleValue)
		c.IsMember = isMemberInt != 0
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate communities: %w", err)
	}
	return out, nil
}

// CommunityVisibilityWhere excludes private communities the viewer cannot see.
func CommunityVisibilityWhere(s *Schema, viewerID int64) (string, []any) {
	if s == nil || !s.HasTable("communities") || !s.HasCol("communities", "visibility") {
		return "", nil
	}
	clause := " AND (COALESCE(c.visibility, 'public') = 'public'"
	var args []any
	membership := s.MembershipTable()
	if membership != "" && s.HasCol(membership, "user_id") {
		clause += fmt.Sprintf(" OR EXISTS (SELECT 1 FROM %s cmx WHERE cmx.community_id = c.id AND cmx.user_id = ? AND %s)",
			membership, s.ActiveMembership("cmx"))
		args = append(args, viewerID)
	}
	if s.HasCol("communities", "owner_id") && viewerID > 0 {
		clause += fmt.Sprintf(" OR c.owner_id = %d", viewerID)
	}
	return clause + ")", args
}

// EscapeLike escapes LIKE wildcard characters so user input is treated
// literally. Use with `LIKE ? ESCAPE '\'`.
func EscapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}
