package db

import (
	"database/sql"
	"errors"
	"strconv"
)

var ErrSettingNotFound = errors.New("setting not found")

type SettingsRepo struct{ db *sql.DB }

func NewSettingsRepo(d *sql.DB) *SettingsRepo { return &SettingsRepo{db: d} }

func (r *SettingsRepo) Get(key string) (string, error) {
	var v string
	err := r.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrSettingNotFound
	}
	return v, err
}

func (r *SettingsRepo) GetBool(key string) (bool, error) {
	v, err := r.Get(key)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}

func (r *SettingsRepo) Set(key, value string) error {
	_, err := r.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}

func (r *SettingsRepo) All() (map[string]string, error) {
	rows, err := r.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}
