package web

import (
	"encoding/xml"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
)

type rssDoc struct {
	Channel struct {
		Title string `xml:"title"`
		Items []struct {
			Title   string `xml:"title"`
			Link    string `xml:"link"`
			PubDate string `xml:"pubDate"`
			GUID    string `xml:"guid"`
		} `xml:"item"`
	} `xml:"channel"`
}

func TestFeedIsValidRSS(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	publishPost(t, db.NewPostRepo(d), "Feed Post", "feed-post", "published", "body", nil)

	rec := apiDo(t, h, http.MethodGet, "/feed.xml", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/rss+xml; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `<rss version="2.0"`) || !strings.Contains(body, "<channel>") {
		t.Fatalf("not RSS: %s", body)
	}
	var doc rssDoc
	if err := xml.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("invalid XML: %v", err)
	}
	if len(doc.Channel.Items) != 1 || doc.Channel.Items[0].Title != "Feed Post" {
		t.Fatalf("items = %+v", doc.Channel.Items)
	}
	if !strings.Contains(doc.Channel.Items[0].Link, "/posts/feed-post") {
		t.Errorf("item link = %q", doc.Channel.Items[0].Link)
	}
}

func TestFeedExcludesDrafts(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	repo := db.NewPostRepo(d)
	publishPost(t, repo, "Live Post", "live-post", "published", "body", nil)
	publishPost(t, repo, "Draft Post", "draft-post", "draft", "body", nil)

	rec := apiDo(t, h, http.MethodGet, "/feed.xml", "", nil)
	if strings.Contains(rec.Body.String(), "Draft Post") {
		t.Error("draft leaked into the feed")
	}
}

func TestFeedPubDateIsRFC1123(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	publishPost(t, db.NewPostRepo(d), "Dated Post", "dated-post", "published", "body", nil)

	rec := apiDo(t, h, http.MethodGet, "/feed.xml", "", nil)
	var doc rssDoc
	if err := xml.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("invalid XML: %v", err)
	}
	if len(doc.Channel.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(doc.Channel.Items))
	}
	got, err := time.Parse(time.RFC1123Z, doc.Channel.Items[0].PubDate)
	if err != nil {
		t.Fatalf("pubDate %q is not RFC 1123: %v", doc.Channel.Items[0].PubDate, err)
	}
	want := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("pubDate = %v, want %v", got, want)
	}
}

func TestFeedDisabledWhenPostsDisabled(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	if err := db.NewSettingsRepo(d).Set("posts_enabled", "false"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	rec := apiDo(t, h, http.MethodGet, "/feed.xml", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
