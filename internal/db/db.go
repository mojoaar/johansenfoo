package db

import (
	"database/sql"
	"net/url"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	dsn := url.URL{Scheme: "file", Path: path, OmitHost: true}
	dsn.RawQuery = "_pragma=foreign_keys(1)" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=journal_mode(WAL)"

	d, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, err
	}

	if err := d.Ping(); err != nil {
		_ = d.Close()
		return nil, err
	}
	return d, nil
}
