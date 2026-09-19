package db

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

var (
	ErrPostNotFound = errors.New("post not found")
	ErrTagNotFound  = errors.New("tag not found")
)

type PostRepo struct{ db *sql.DB }

func NewPostRepo(d *sql.DB) *PostRepo { return &PostRepo{db: d} }

func IsUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

const postColumns = `id, slug, title, summary, body_md, status, published_at, created_at, updated_at,
	hero_image_url, hero_image_alt, seo_title, seo_description, og_image_url, canonical_url, noindex`

const nowExpr = `strftime('%Y-%m-%dT%H:%M:%SZ','now')`

type rowScanner interface{ Scan(dest ...any) error }

func scanPost(row rowScanner) (*Post, error) {
	var p Post
	var published, heroURL, heroAlt sql.NullString
	var created, updated string
	var noindex int
	if err := row.Scan(&p.ID, &p.Slug, &p.Title, &p.Summary, &p.BodyMD, &p.Status, &published,
		&created, &updated, &heroURL, &heroAlt, &p.SEOTitle, &p.SEODescription, &p.OGImageURL,
		&p.CanonicalURL, &noindex); err != nil {
		return nil, err
	}
	if published.Valid && published.String != "" {
		v, err := time.Parse(time.RFC3339, published.String)
		if err != nil {
			return nil, err
		}
		p.PublishedAt = &v
	}
	var err error
	if p.CreatedAt, err = time.Parse(time.RFC3339, created); err != nil {
		return nil, err
	}
	if p.UpdatedAt, err = time.Parse(time.RFC3339, updated); err != nil {
		return nil, err
	}
	p.HeroImageURL = heroURL.String
	p.HeroImageAlt = heroAlt.String
	p.NoIndex = noindex != 0
	return &p, nil
}

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	dash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		default:
			if !dash {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func (r *PostRepo) queryPosts(query string, args ...any) ([]Post, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Post
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return r.attachTags(out)
}

func (r *PostRepo) attachTags(posts []Post) ([]Post, error) {
	if len(posts) == 0 {
		return posts, nil
	}
	index := make(map[int64]int, len(posts))
	for i := range posts {
		index[posts[i].ID] = i
		posts[i].Tags = []Tag{}
	}
	rows, err := r.db.Query(`
		SELECT pt.post_id, t.id, t.name, t.slug
		FROM post_tag pt JOIN tag t ON t.id = pt.tag_id
		ORDER BY t.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		var tag Tag
		if err := rows.Scan(&postID, &tag.ID, &tag.Name, &tag.Slug); err != nil {
			return nil, err
		}
		if i, ok := index[postID]; ok {
			posts[i].Tags = append(posts[i].Tags, tag)
		}
	}
	return posts, rows.Err()
}

func (r *PostRepo) one(query string, args ...any) (*Post, error) {
	p, err := scanPost(r.db.QueryRow(query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPostNotFound
	}
	if err != nil {
		return nil, err
	}
	posts, err := r.attachTags([]Post{*p})
	if err != nil {
		return nil, err
	}
	return &posts[0], nil
}

func (r *PostRepo) All() ([]Post, error) {
	return r.queryPosts(`SELECT ` + postColumns + ` FROM post ORDER BY created_at DESC, id DESC`)
}

func (r *PostRepo) Published(limit, offset int) ([]Post, error) {
	return r.queryPosts(`SELECT `+postColumns+` FROM post
		WHERE status = 'published' AND published_at IS NOT NULL
		ORDER BY published_at DESC, id DESC LIMIT ? OFFSET ?`, limit, offset)
}

func (r *PostRepo) CountPublished() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM post WHERE status = 'published' AND published_at IS NOT NULL`).Scan(&n)
	return n, err
}

func (r *PostRepo) PublishedBySlug(slug string) (*Post, error) {
	return r.one(`SELECT `+postColumns+` FROM post WHERE slug = ? AND status = 'published' AND published_at IS NOT NULL`, slug)
}

func (r *PostRepo) ByID(id int64) (*Post, error) {
	return r.one(`SELECT `+postColumns+` FROM post WHERE id = ?`, id)
}

func (r *PostRepo) ByTag(slug string, limit, offset int) ([]Post, error) {
	return r.queryPosts(`SELECT `+postColumns+` FROM post
		WHERE status = 'published' AND published_at IS NOT NULL
		AND id IN (SELECT pt.post_id FROM post_tag pt JOIN tag t ON t.id = pt.tag_id WHERE t.slug = ?)
		ORDER BY published_at DESC, id DESC LIMIT ? OFFSET ?`, slug, limit, offset)
}

func (r *PostRepo) CountByTag(slug string) (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM post
		WHERE status = 'published' AND published_at IS NOT NULL
		AND id IN (SELECT pt.post_id FROM post_tag pt JOIN tag t ON t.id = pt.tag_id WHERE t.slug = ?)`, slug).Scan(&n)
	return n, err
}

func (r *PostRepo) setTags(tx *sql.Tx, postID int64, tags []Tag) error {
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		slug := strings.TrimSpace(tag.Slug)
		if slug == "" {
			slug = Slugify(name)
		}
		if slug == "" {
			continue
		}
		if name == "" {
			name = slug
		}
		var tagID int64
		if err := tx.QueryRow(
			`INSERT INTO tag (name, slug) VALUES (?, ?)
			 ON CONFLICT(slug) DO UPDATE SET name = excluded.name
			 RETURNING id`, name, slug).Scan(&tagID); err != nil {
			return err
		}
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO post_tag (post_id, tag_id) VALUES (?, ?)`, postID, tagID); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostRepo) Create(p *Post) (int64, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	status := p.Status
	if status == "" {
		status = "draft"
	}
	var published any
	if p.PublishedAt != nil {
		published = p.PublishedAt.UTC().Format(time.RFC3339)
	}
	res, err := tx.Exec(`INSERT INTO post
		(slug, title, summary, body_md, status, published_at, created_at, updated_at,
		 hero_image_url, hero_image_alt, seo_title, seo_description, og_image_url, canonical_url, noindex)
		VALUES (?, ?, ?, ?, ?, ?, `+nowExpr+`, `+nowExpr+`, ?, ?, ?, ?, ?, ?, ?)`,
		p.Slug, p.Title, p.Summary, p.BodyMD, status, published,
		p.HeroImageURL, p.HeroImageAlt, p.SEOTitle, p.SEODescription, p.OGImageURL, p.CanonicalURL, boolToInt(p.NoIndex))
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := r.setTags(tx, id, p.Tags); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (r *PostRepo) Update(p *Post) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	status := p.Status
	if status == "" {
		status = "draft"
	}
	var published any
	if p.PublishedAt != nil {
		published = p.PublishedAt.UTC().Format(time.RFC3339)
	}
	if _, err := tx.Exec(`UPDATE post SET
		slug = ?, title = ?, summary = ?, body_md = ?, status = ?,
		published_at = COALESCE(?, published_at), updated_at = `+nowExpr+`,
		hero_image_url = ?, hero_image_alt = ?, seo_title = ?, seo_description = ?,
		og_image_url = ?, canonical_url = ?, noindex = ?
		WHERE id = ?`,
		p.Slug, p.Title, p.Summary, p.BodyMD, status, published,
		p.HeroImageURL, p.HeroImageAlt, p.SEOTitle, p.SEODescription, p.OGImageURL, p.CanonicalURL,
		boolToInt(p.NoIndex), p.ID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM post_tag WHERE post_id = ?`, p.ID); err != nil {
		return err
	}
	if err := r.setTags(tx, p.ID, p.Tags); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *PostRepo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM post WHERE id = ?`, id)
	return err
}

func (r *PostRepo) SetStatus(id int64, status string) error {
	if status == "published" {
		_, err := r.db.Exec(`UPDATE post SET status = 'published',
			published_at = COALESCE(published_at, `+nowExpr+`), updated_at = `+nowExpr+`
			WHERE id = ?`, id)
		return err
	}
	_, err := r.db.Exec(`UPDATE post SET status = 'draft', updated_at = `+nowExpr+` WHERE id = ?`, id)
	return err
}

func (r *PostRepo) Tags() ([]Tag, error) {
	rows, err := r.db.Query(`SELECT id, name, slug FROM tag ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *PostRepo) TagBySlug(slug string) (*Tag, error) {
	var t Tag
	err := r.db.QueryRow(`SELECT id, name, slug FROM tag WHERE slug = ?`, slug).Scan(&t.ID, &t.Name, &t.Slug)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTagNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}
