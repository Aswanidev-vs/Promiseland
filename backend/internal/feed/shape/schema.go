// Package shape holds the R1 wire DTOs and the local SQL used to build them.
//
// It lives under feed but is deliberately dependency-free: both the feed and
// search modules import it so their Post/UserSummary/Community JSON is byte-for-
// byte identical. It does not import any other domain package.
package shape

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Schema caches the tables and columns that actually exist in the database.
// R1 modules are developed in parallel, so queries must adapt to the schema
// that is present instead of assuming a fixed column list.
type Schema struct {
	tables map[string]bool
	cols   map[string]map[string]bool
}

// LoadSchema inspects sqlite_master and PRAGMA table_info.
func LoadSchema(ctx context.Context, db *sql.DB) *Schema {
	s := &Schema{tables: map[string]bool{}, cols: map[string]map[string]bool{}}
	if db == nil {
		return s
	}
	rows, err := db.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type IN ('table','view')`)
	if err != nil {
		return s
	}
	var names []string
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil {
			names = append(names, name)
		}
	}
	_ = rows.Close()

	for _, name := range names {
		s.tables[name] = true
		cols := map[string]bool{}
		info, err := db.QueryContext(ctx, "PRAGMA table_info("+quoteIdent(name)+")")
		if err != nil {
			s.cols[name] = cols
			continue
		}
		for info.Next() {
			var cid, notNull, pk int
			var colName, colType string
			var dflt any
			if info.Scan(&cid, &colName, &colType, &notNull, &dflt, &pk) == nil {
				cols[strings.ToLower(colName)] = true
			}
		}
		_ = info.Close()
		s.cols[name] = cols
	}
	return s
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// HasTable reports whether a table or view exists.
func (s *Schema) HasTable(name string) bool {
	return s != nil && s.tables[name]
}

// HasCol reports whether a table has a column (case-insensitive).
func (s *Schema) HasCol(table, col string) bool {
	if s == nil {
		return false
	}
	return s.cols[table][strings.ToLower(col)]
}

// Col returns the first candidate column that exists, or "".
func (s *Schema) Col(table string, candidates ...string) string {
	for _, c := range candidates {
		if s.HasCol(table, c) {
			return c
		}
	}
	return ""
}

// Text yields a text-valued SQL expression for alias.col, or a constant when
// the column is absent.
func (s *Schema) Text(table, alias, col string) string {
	if s.HasCol(table, col) {
		return fmt.Sprintf("CAST(%s.%s AS TEXT)", alias, col)
	}
	return "''"
}

// NullableText yields a non-null text expression for alias.col.
func (s *Schema) NullableText(table, alias, col string) string {
	if s.HasCol(table, col) {
		return fmt.Sprintf("COALESCE(CAST(%s.%s AS TEXT), '')", alias, col)
	}
	return "''"
}

// Int yields a numeric SQL expression for alias.col, or 0 when absent.
func (s *Schema) Int(table, alias, col string) string {
	if s.HasCol(table, col) {
		return fmt.Sprintf("COALESCE(%s.%s, 0)", alias, col)
	}
	return "0"
}

// MembershipTable returns the community membership table name used by the R1
// communities migration, with a fallback for an alternate name.
func (s *Schema) MembershipTable() string {
	for _, candidate := range []string{"community_memberships", "community_members"} {
		if s.HasTable(candidate) {
			return candidate
		}
	}
	return ""
}

// ActiveMembership returns a predicate restricting a membership alias to
// active memberships when the table has a status column.
func (s *Schema) ActiveMembership(alias string) string {
	if s.HasCol("community_memberships", "status") || s.HasCol("community_members", "status") {
		return alias + ".status = 'active'"
	}
	return "1 = 1"
}
