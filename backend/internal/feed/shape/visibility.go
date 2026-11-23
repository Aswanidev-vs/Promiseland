package shape

import (
	"fmt"
	"strings"
)

// VisibilityClauses returns the shared post visibility SQL: no private
// community posts to non-members, no blocked authors, no suspended authors.
// The fragment is trusted SQL; viewerID is always bound as an argument where
// possible.
func VisibilityClauses(s *Schema, viewerID int64) (string, []any) {
	if s == nil {
		return "", nil
	}
	var clauses []string
	var args []any
	add := func(fragment string, values ...any) {
		clauses = append(clauses, fragment)
		args = append(args, values...)
	}

	if s.HasTable("communities") && s.HasCol("communities", "visibility") {
		clause := "(p.community_id IS NULL OR COALESCE(c.visibility, 'public') = 'public'"
		membership := s.MembershipTable()
		if membership != "" && s.HasCol(membership, "user_id") {
			clause += fmt.Sprintf(" OR EXISTS (SELECT 1 FROM %s cmv WHERE cmv.community_id = p.community_id AND cmv.user_id = ? AND %s)",
				membership, s.ActiveMembership("cmv"))
			args = append(args, viewerID)
		}
		if s.HasCol("communities", "owner_id") && viewerID > 0 {
			clause += fmt.Sprintf(" OR c.owner_id = %d", viewerID)
		}
		add(clause + ")")
	}

	if s.HasTable("blocks") {
		blocker := s.Col("blocks", "blocker_id", "user_id", "blocking_id")
		blocked := s.Col("blocks", "blocked_id", "blocking_user_id", "target_id")
		if blocker != "" && blocked != "" {
			add(fmt.Sprintf(`AND NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.%s = ? AND b.%s = p.author_id)
				   OR (b.%s = p.author_id AND b.%s = ?)
			)`, blocker, blocked, blocker, blocked), viewerID, viewerID)
		}
	}

	if s.HasTable("suspensions") {
		add(`AND NOT EXISTS (
			SELECT 1 FROM suspensions su
			WHERE su.user_id = p.author_id AND su.is_active = 1
			  AND (su.expires_at IS NULL OR su.expires_at > datetime('now'))
		)`)
	}

	return strings.Join(clauses, " "), args
}
