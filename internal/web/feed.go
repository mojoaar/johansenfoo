package web

import (
	"bytes"
	"encoding/xml"
	"net/http"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func formatRSSDate(t time.Time) string {
	return t.UTC().Format(time.RFC1123Z)
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func writeElem(buf *bytes.Buffer, name, value string) {
	buf.WriteByte('<')
	buf.WriteString(name)
	buf.WriteByte('>')
	buf.WriteString(xmlEscape(value))
	buf.WriteString("</")
	buf.WriteString(name)
	buf.WriteString(">\n")
}

func feedHandler(d Deps) http.HandlerFunc {
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
		base := c.Settings["canonical_base_url"]
		if base == "" {
			base = "https://johansen.foo"
		}
		posts, err := db.NewPostRepo(d.DB).Published(50, 0)
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}

		var buf bytes.Buffer
		buf.WriteString(xml.Header)
		buf.WriteString(`<rss version="2.0">` + "\n")
		buf.WriteString("<channel>\n")
		writeElem(&buf, "title", c.Profile.Name+" — Blog")
		writeElem(&buf, "link", base+"/posts")
		writeElem(&buf, "description", c.Settings["seo_description"])
		writeElem(&buf, "lastBuildDate", formatRSSDate(time.Now().UTC()))
		for _, p := range posts {
			link := base + "/posts/" + p.Slug
			buf.WriteString("<item>\n")
			writeElem(&buf, "title", p.Title)
			writeElem(&buf, "link", link)
			writeElem(&buf, "guid", link)
			if p.PublishedAt != nil {
				writeElem(&buf, "pubDate", formatRSSDate(*p.PublishedAt))
			}
			writeElem(&buf, "description", p.Summary)
			buf.WriteString("</item>\n")
		}
		buf.WriteString("</channel>\n</rss>\n")

		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		_, _ = w.Write(buf.Bytes())
	}
}
