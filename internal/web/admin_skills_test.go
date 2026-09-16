package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAdminSkillsCreateUpdateDelete(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodGet, "/admin/skills", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Terraform") {
		t.Error("skills list is missing a seeded skill")
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/skills", url.Values{
		"name": {"Zig"}, "sort": {"99"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d, want 303", rec.Code)
	}
	if len(store.Current().Skills) != 31 {
		t.Fatalf("snapshot skills = %d, want 31", len(store.Current().Skills))
	}

	var id int64
	for _, s := range store.Current().Skills {
		if s.Name == "Zig" {
			id = s.ID
		}
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/skills/"+itoa(id), url.Values{
		"name": {"Ziglang"}, "sort": {"5"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("update status = %d, want 303", rec.Code)
	}
	sk, err := db.NewContentRepo(d).Skill(id)
	if err != nil {
		t.Fatalf("Skill: %v", err)
	}
	if sk.Name != "Ziglang" {
		t.Errorf("Name = %q, want %q", sk.Name, "Ziglang")
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/skills/"+itoa(id)+"/delete", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want 303", rec.Code)
	}
	if len(store.Current().Skills) != 30 {
		t.Errorf("snapshot skills = %d, want 30", len(store.Current().Skills))
	}
}

func skillFromSnapshot(t *testing.T, store *ContentStore, id int64) db.Skill {
	t.Helper()
	for _, s := range store.Current().Skills {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("skill %d is missing from the content snapshot", id)
	return db.Skill{}
}

func TestAdminSkillUpdateRefreshesContentSnapshot(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	id, err := db.NewContentRepo(d).CreateSkill(&db.Skill{Name: "snapshotold", Sort: 99, Visible: true})
	if err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if got := skillFromSnapshot(t, store, id); got.Name != "snapshotold" {
		t.Fatalf("precondition: snapshot Name = %q, want %q", got.Name, "snapshotold")
	}

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/skills/"+itoa(id), url.Values{
		"name": {"snapshotnew"}, "sort": {"5"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}

	got := skillFromSnapshot(t, store, id)
	if got.Name != "snapshotnew" {
		t.Errorf("snapshot Name = %q, want %q", got.Name, "snapshotnew")
	}
	if got.Sort != 5 {
		t.Errorf("snapshot Sort = %d, want %d", got.Sort, 5)
	}
}

func TestAdminSkillsHiddenRowsStayOutOfPublicSite(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	id, err := db.NewContentRepo(d).CreateSkill(&db.Skill{Name: "zzzhiddenskill", Sort: 50, Visible: false})
	if err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	found := false
	for _, s := range store.Current().Skills {
		if s.ID == id {
			found = true
		}
	}
	if !found {
		t.Fatal("hidden skill is missing from the admin content snapshot")
	}

	adminGet := httptest.NewRequest(http.MethodGet, "/admin/skills", nil)
	adminGet.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	adminRec := httptest.NewRecorder()
	h.ServeHTTP(adminRec, adminGet)
	if !strings.Contains(adminRec.Body.String(), "zzzhiddenskill") {
		t.Error("admin skills list does not show the hidden row for re-enabling")
	}

	landing := httptest.NewRequest(http.MethodGet, "/", nil)
	landingRec := httptest.NewRecorder()
	h.ServeHTTP(landingRec, landing)
	if strings.Contains(landingRec.Body.String(), "zzzhiddenskill") {
		t.Error("hidden skill leaked onto /")
	}

	me := httptest.NewRequest(http.MethodGet, "/me", nil)
	meRec := httptest.NewRecorder()
	h.ServeHTTP(meRec, me)
	if strings.Contains(meRec.Body.String(), "zzzhiddenskill") {
		t.Error("hidden skill leaked into /me")
	}

	ld := landingRec.Body.String()
	const marker = `<script type="application/ld+json">`
	i := strings.Index(ld, marker)
	if i < 0 {
		t.Fatal("landing page has no JSON-LD block")
	}
	rest := ld[i+len(marker):]
	j := strings.Index(rest, `</script>`)
	if j < 0 {
		t.Fatal("landing page JSON-LD block is not terminated")
	}
	if strings.Contains(rest[:j], "zzzhiddenskill") {
		t.Error("hidden skill leaked into the JSON-LD structured data")
	}
}

func TestAdminSkillsVisibilityReachesPublicSite(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	id, err := db.NewContentRepo(d).CreateSkill(&db.Skill{Name: "zzztoggleskill", Sort: 50, Visible: false})
	if err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if skillFromSnapshot(t, store, id).Visible {
		t.Fatal("precondition: snapshot Visible = true, want false")
	}
	if strings.Contains(publicLandingBody(t, h), "zzztoggleskill") {
		t.Fatal("precondition: hidden skill already appears on /")
	}

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/skills/"+itoa(id), url.Values{
		"name": {"zzztoggleskill"}, "sort": {"50"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("enable status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	if !skillFromSnapshot(t, store, id).Visible {
		t.Error("snapshot Visible = false after enabling, want true")
	}
	if !strings.Contains(publicLandingBody(t, h), "zzztoggleskill") {
		t.Error("newly visible skill does not appear on /")
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/skills/"+itoa(id), url.Values{
		"name": {"zzztoggleskill"}, "sort": {"50"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("disable status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	if skillFromSnapshot(t, store, id).Visible {
		t.Error("snapshot Visible = true after disabling, want false")
	}
	if strings.Contains(publicLandingBody(t, h), "zzztoggleskill") {
		t.Error("hidden skill still appears on /")
	}
}

func TestAdminSkillsFormTemplatesRenderCSRFToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	cases := []struct {
		path  string
		forms int
	}{
		{path: "/admin/skills", forms: 2 + 2*len(store.Current().Skills)},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
			token := findCookie(t, rec.Result().Cookies(), csrfCookieName).Value
			want := `name="csrf_token" value="` + token + `"`
			if got := strings.Count(rec.Body.String(), want); got != tc.forms {
				t.Errorf("%s renders %d csrf-token hidden fields, want %d", tc.path, got, tc.forms)
			}
		})
	}
}

func TestAdminSkillCreateAcceptsRenderedCSRFToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	get := httptest.NewRequest(http.MethodGet, "/admin/skills", nil)
	get.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, get)
	token := csrfTokenForAction(t, getRec.Body.String(), "/admin/skills")

	form := url.Values{"csrf_token": {token}, "name": {"csrfskill"}, "sort": {"99"}, "visible": {"1"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/skills", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	found := false
	for _, s := range store.Current().Skills {
		if s.Name == "csrfskill" {
			found = true
		}
	}
	if !found {
		t.Error("created skill is missing from the content snapshot")
	}
}

func TestAdminSkillPostRejectsMissingCSRF(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	form := url.Values{"name": {"no-csrf"}, "sort": {"99"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/skills", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; a skills POST without a CSRF token was accepted", rec.Code)
	}
}
