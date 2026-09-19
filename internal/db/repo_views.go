package db

import (
	"database/sql"
	"time"
)

type PageView struct {
	Path      string    `json:"path"`
	Referrer  string    `json:"referrer"`
	UserAgent string    `json:"user_agent"`
	IPHash    string    `json:"ip_hash"`
	CreatedAt time.Time `json:"created_at"`
}

type PathCount struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

type ReferrerCount struct {
	Referrer string `json:"referrer"`
	Count    int    `json:"count"`
}

type PageViewRepo struct{ db *sql.DB }

func NewPageViewRepo(d *sql.DB) *PageViewRepo { return &PageViewRepo{db: d} }

func nullableText(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (r *PageViewRepo) Record(v *PageView) error {
	if v.CreatedAt.IsZero() {
		_, err := r.db.Exec(
			`INSERT INTO page_view (path, referrer, user_agent, ip_hash, created_at)
			 VALUES (?, ?, ?, ?, `+nowExpr+`)`,
			v.Path, nullableText(v.Referrer), nullableText(v.UserAgent), nullableText(v.IPHash),
		)
		return err
	}
	_, err := r.db.Exec(
		`INSERT INTO page_view (path, referrer, user_agent, ip_hash, created_at) VALUES (?, ?, ?, ?, ?)`,
		v.Path, nullableText(v.Referrer), nullableText(v.UserAgent), nullableText(v.IPHash),
		v.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (r *PageViewRepo) CountSince(since string) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM page_view WHERE created_at >= ?`, since).Scan(&n)
	return n, err
}

func (r *PageViewRepo) CountBetween(from, to string) (int, error) {
	var n int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM page_view WHERE created_at >= ? AND created_at < ?`, from, to,
	).Scan(&n)
	return n, err
}

func (r *PageViewRepo) DailyUniqueCount(day string) (int, error) {
	var n int
	err := r.db.QueryRow(
		`SELECT COUNT(DISTINCT ip_hash) FROM page_view
		 WHERE substr(created_at, 1, 10) = ? AND ip_hash IS NOT NULL AND ip_hash <> ''`, day,
	).Scan(&n)
	return n, err
}

func (r *PageViewRepo) TopPaths(since string, limit int) ([]PathCount, error) {
	rows, err := r.db.Query(
		`SELECT path, COUNT(*) AS c FROM page_view WHERE created_at >= ?
		 GROUP BY path ORDER BY c DESC, path LIMIT ?`, since, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PathCount
	for rows.Next() {
		var p PathCount
		if err := rows.Scan(&p.Path, &p.Count); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PageViewRepo) TopReferrers(since string, limit int) ([]ReferrerCount, error) {
	rows, err := r.db.Query(
		`SELECT referrer, COUNT(*) AS c FROM page_view
		 WHERE created_at >= ? AND referrer IS NOT NULL AND referrer <> ''
		 GROUP BY referrer ORDER BY c DESC, referrer LIMIT ?`, since, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReferrerCount
	for rows.Next() {
		var rc ReferrerCount
		if err := rows.Scan(&rc.Referrer, &rc.Count); err != nil {
			return nil, err
		}
		out = append(out, rc)
	}
	return out, rows.Err()
}

func (r *PageViewRepo) Recent(limit int) ([]PageView, error) {
	rows, err := r.db.Query(
		`SELECT path, COALESCE(referrer, ''), COALESCE(user_agent, ''), COALESCE(ip_hash, ''), created_at
		 FROM page_view ORDER BY created_at DESC, id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PageView
	for rows.Next() {
		var v PageView
		var created string
		if err := rows.Scan(&v.Path, &v.Referrer, &v.UserAgent, &v.IPHash, &created); err != nil {
			return nil, err
		}
		if parsed, err := time.Parse(time.RFC3339, created); err == nil {
			v.CreatedAt = parsed
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *PageViewRepo) Prune(before string) (int64, error) {
	res, err := r.db.Exec(`DELETE FROM page_view WHERE created_at < ?`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *PageViewRepo) Clear() error {
	_, err := r.db.Exec(`DELETE FROM page_view`)
	return err
}
