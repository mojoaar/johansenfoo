package web

import (
	"database/sql"
	"sync/atomic"

	"github.com/mojoaar/johansenfoo/internal/db"
)

type ContentStore struct {
	db  *sql.DB
	cur atomic.Pointer[db.SiteContent]
}

func NewContentStore(d *sql.DB) (*ContentStore, error) {
	s := &ContentStore{db: d}
	if err := s.Reload(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *ContentStore) Current() *db.SiteContent {
	return s.cur.Load()
}

func (s *ContentStore) Reload() error {
	c, err := LoadContent(s.db)
	if err != nil {
		return err
	}
	s.cur.Store(c)
	return nil
}
