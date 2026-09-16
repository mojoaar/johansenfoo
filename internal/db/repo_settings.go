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
