package internal

import (
	_ "github.com/glebarez/go-sqlite"
	"github.com/jmoiron/sqlx"
)

func GetDB() *sqlx.DB {
	return sqlx.MustOpen(
		"sqlite",
		"meta.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)",
	)
}
