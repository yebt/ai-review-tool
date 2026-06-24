package migrations

import "embed"

// Files contains the SQL migrations bundled into the server binary.
//
//go:embed *.sql
var Files embed.FS
