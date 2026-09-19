package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAdminAPIProfileGet(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/profile", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	got := decodeBody[db.Profile](t, rec)
	if got.Name != "Morten Johansen" {
		t.Errorf("name = %q", got.Name)
	}
}

func TestAdminAPIProfilePutPersistsAndReloads(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	body := `{"name":"New Name","handle":"mojoaar","location":"Denmark","dob":"1980-08-13","tagline":"t","hero_bio":"h","bio":"b","about_para_1":"a","about_para_2":"b","avatar":"avatar.png"}`
	rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/profile", body, withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	stored, err := db.NewProfileRepo(d).Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.Name != "New Name" {
		t.Errorf("stored name = %q, want New Name", stored.Name)
	}
	if current := store.Current(); current.Profile.Name != "New Name" {
		t.Errorf("snapshot name = %q, want New Name (Reload missing)", current.Profile.Name)
	}
}

func TestAdminAPIProfilePutRequiresName(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/profile", `{"name":""}`, withSession)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestAdminAPISocialCRUD(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodPost, "/api/v1/admin/social",
		`{"platform":"signal","url":"https://signal.me/x","label":"Signal","sort":9,"visible":true}`, withSession)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	created := decodeBody[db.SocialLink](t, rec)
	if created.ID == 0 || created.Platform != "signal" {
		t.Fatalf("created = %+v", created)
	}

	rec = apiDo(t, h, http.MethodPut, "/api/v1/admin/social/"+itoa(created.ID),
		`{"platform":"signal","url":"https://signal.me/y","label":"Signal","sort":9,"visible":false}`, withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	updated := decodeBody[db.SocialLink](t, rec)
	if updated.URL != "https://signal.me/y" || updated.Visible {
		t.Fatalf("updated = %+v", updated)
	}
	if current := store.Current(); !socialHidden(current.Social, created.ID) {
		t.Fatal("hidden social link still visible in the refreshed snapshot")
	}

	rec = apiDo(t, h, http.MethodDelete, "/api/v1/admin/social/"+itoa(created.ID), "", withSession)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", rec.Code, rec.Body.String())
	}
	if rec = apiDo(t, h, http.MethodDelete, "/api/v1/admin/social/"+itoa(created.ID), "", withSession); rec.Code != http.StatusNotFound {
		t.Fatalf("second delete status = %d, want 404", rec.Code)
	}
}

func socialHidden(links []db.SocialLink, id int64) bool {
	for _, l := range links {
		if l.ID == id {
			return !l.Visible
		}
	}
	return false
}

func TestAdminAPISocialListIncludesHidden(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/social", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := decodeBody[[]db.SocialLink](t, rec); len(got) != 4 {
		t.Errorf("social = %d, want 4", len(got))
	}
}

func TestAdminAPISocialRejectsBadPayload(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodPost, "/api/v1/admin/social", `not json`, withSession)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad json status = %d, want 400", rec.Code)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/social", strings.NewReader("platform=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withSession(req)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("form-post status = %d, want 415", rec.Code)
	}
}
