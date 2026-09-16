package db

import (
	"database/sql"
	"errors"
	"time"
)

var ErrSessionNotFound = errors.New("session not found")

type Session struct {
	ID        string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SessionRepo struct{ db *sql.DB }

func NewSessionRepo(d *sql.DB) *SessionRepo { return &SessionRepo{db: d} }

func (r *SessionRepo) Create(id string, expiresAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO session (id, created_at, expires_at)
		 VALUES (?, strftime('%Y-%m-%dT%H:%M:%SZ','now'), ?)`,
		id, expiresAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (r *SessionRepo) Get(id string) (*Session, error) {
	var s Session
	var created, expires string
	err := r.db.QueryRow(
		`SELECT id, created_at, expires_at FROM session WHERE id = ?`, id,
	).Scan(&s.ID, &created, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	if s.CreatedAt, err = time.Parse(time.RFC3339, created); err != nil {
		return nil, err
	}
	if s.ExpiresAt, err = time.Parse(time.RFC3339, expires); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepo) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM session WHERE id = ?`, id)
	return err
}

func (r *SessionRepo) DeleteExpired() (int64, error) {
	res, err := r.db.Exec(
		`DELETE FROM session WHERE expires_at <= strftime('%Y-%m-%dT%H:%M:%SZ','now')`,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
