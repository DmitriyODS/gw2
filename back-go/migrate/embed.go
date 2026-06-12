// Package migrate — embed SQL-миграций схемы Groove Work (goose).
// Сами миграции — migrations/*.sql; раннер — cmd/migrate.
package migrate

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
