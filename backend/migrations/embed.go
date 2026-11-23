// Package migrations embeds the SQL migration files so the compiled server
// binary can migrate a fresh database without external files.
package migrations

import "embed"

// FS holds the NNN_name.sql migration files applied in filename order.
//
//go:embed *.sql
var FS embed.FS
