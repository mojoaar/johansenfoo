package web

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mojoaar/johansenfoo/internal/db"
	"github.com/mojoaar/johansenfoo/internal/markdown"
	"github.com/mojoaar/johansenfoo/internal/theme"
)

const postsPerPage = 10

type postView struct {
	db.Post
	Date string
}

func toView(p db.Post, loc *time.Location) postView {
	v := postView{Post: p}
	if p.PublishedAt != nil {
		v.Date = formatPostDate(*p.PublishedAt, loc)
	}
	return v
}

func postsEnabled(c *db.SiteContent) bool {
	if c == nil {
		return false
	}
	return c.Settings["posts_enabled"] != "false"
}

func siteLocation(c *db.SiteContent) *time.Location {
	if c != nil {
		if name := c.Settings["timezone"]; name != "" {
			if loc, err := time.LoadLocation(name); err == nil {
				return loc
			}
		}
	}
	return time.UTC
}

func formatPostDate(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("2006-01-02 15:04")
}

func pageParam(r *http.Request) int {
	n, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || n < 1 {
		return 1
	}
	return n
}

func totalPages(total int) int {
	if total <= 0 {
		return 1
	}
	return (total + postsPerPage - 1) / postsPerPage
}

func withPage(base string, n int) string {
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + "page=" + strconv.Itoa(n)
}

func paginationURLs(base string, pageNum, total int) (string, string) {
	last := totalPages(total)
	var prev, next string
	if pageNum > 1 {
		prev = withPage(base, pageNum-1)
	}
	if pageNum < last {
		next = withPage(base, pageNum+1)
	}
	return prev, next
}

func newPostPage(d Deps, route string, ov metaOverride) page {
	c := d.Content.Current()
	meta := resolveMeta(c, route, ov)
	p := page{
		Title:          meta.Title,
		PostsEnabled:   postsEnabled(c),
		StructuredData: personSchema(c),
		Profile:        c.Profile,
		Social:         visibleSocial(c.Social),
		ThemeSlug:      c.Theme.Slug,
		ThemeCSS:       template.CSS(theme.CSS(themeFromRow(c.Theme))),
		Meta:           meta,
	}
	return p
}

func postIndexHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			http.Error(w, "content unavailable", http.StatusInternalServerError)
			return
		}
		if !postsEnabled(c) {
			http.NotFound(w, r)
			return
		}
		repo := db.NewPostRepo(d.DB)
		pageNum := pageParam(r)
		offset := (pageNum - 1) * postsPerPage
		tagSlug := r.URL.Query().Get("tag")

		var (
			posts []db.Post
			total int
			err   error
		)
		if tagSlug != "" {
			total, err = repo.CountByTag(tagSlug)
			if err == nil {
				posts, err = repo.ByTag(tagSlug, postsPerPage, offset)
			}
		} else {
			total, err = repo.CountPublished()
			if err == nil {
				posts, err = repo.Published(postsPerPage, offset)
			}
		}
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}

		loc := siteLocation(c)
		views := make([]postView, 0, len(posts))
		for _, p := range posts {
			views = append(views, toView(p, loc))
		}
		data := newPostPage(d, "/posts", metaOverride{title: "Posts"})
		data.Posts = views
		data.PageNum = pageNum
		data.TotalPages = totalPages(total)
		base := "/posts"
		if tagSlug != "" {
			base = "/posts?tag=" + tagSlug
		}
		data.PrevURL, data.NextURL = paginationURLs(base, pageNum, total)
		renderPage(w, "posts", data)
	}
}

func tagArchiveHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			http.Error(w, "content unavailable", http.StatusInternalServerError)
			return
		}
		if !postsEnabled(c) {
			http.NotFound(w, r)
			return
		}
		slug := chi.URLParam(r, "slug")
		repo := db.NewPostRepo(d.DB)
		tag, err := repo.TagBySlug(slug)
		if errors.Is(err, db.ErrTagNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		pageNum := pageParam(r)
		offset := (pageNum - 1) * postsPerPage
		total, err := repo.CountByTag(slug)
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		posts, err := repo.ByTag(slug, postsPerPage, offset)
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		loc := siteLocation(c)
		views := make([]postView, 0, len(posts))
		for _, p := range posts {
			views = append(views, toView(p, loc))
		}
		data := newPostPage(d, "/tags/"+tag.Slug, metaOverride{title: "Posts tagged " + tag.Name})
		data.Tag = *tag
		data.Posts = views
		data.PageNum = pageNum
		data.TotalPages = totalPages(total)
		data.PrevURL, data.NextURL = paginationURLs("/tags/"+tag.Slug, pageNum, total)
		renderPage(w, "tag", data)
	}
}

func postHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			http.Error(w, "content unavailable", http.StatusInternalServerError)
			return
		}
		if !postsEnabled(c) {
			http.NotFound(w, r)
			return
		}
		slug := chi.URLParam(r, "slug")
		post, err := db.NewPostRepo(d.DB).PublishedBySlug(slug)
		if errors.Is(err, db.ErrPostNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		data := newPostPage(d, "/posts/"+post.Slug, metaOverride{
			title:       firstNonEmpty(post.SEOTitle, post.Title),
			description: firstNonEmpty(post.SEODescription, post.Summary),
			image:       firstNonEmpty(post.OGImageURL, post.HeroImageURL),
			canonical:   post.CanonicalURL,
			noindex:     post.NoIndex,
		})
		data.Post = toView(*post, siteLocation(c))
		data.PostBody = markdown.Render(post.BodyMD)
		renderPage(w, "post", data)
	}
}
