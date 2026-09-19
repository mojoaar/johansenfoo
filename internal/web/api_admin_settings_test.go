package web

import (
	"net/http"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAdminAPIPostsSetting(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/settings/posts", `{"enabled":false}`, withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got, err := db.NewSettingsRepo(d).GetBool("posts_enabled")
	if err != nil {
		t.Fatalf("GetBool: %v", err)
	}
	if got {
		t.Error("posts_enabled = true, want false")
	}
	if rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/settings/posts", `{}`, withSession); rec.Code != http.StatusBadRequest {
		t.Fatalf("missing enabled status = %d, want 400", rec.Code)
	}
}

func TestAdminAPIThemeSetting(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	if rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/settings/theme", `{"slug":"nope"}`, withSession); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown theme status = %d, want 400", rec.Code)
	}
	if rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/settings/theme", `{"slug":"johansen"}`, withSession); rec.Code != http.StatusOK {
		t.Fatalf("known theme status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got, err := db.NewSettingsRepo(d).Get("active_theme")
	if err != nil || got != "johansen" {
		t.Fatalf("active_theme = %q, err=%v", got, err)
	}
}

func TestAdminAPIExportImportRoundTrip(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/export", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("export status = %d, want 200", rec.Code)
	}
	snap := decodeBody[db.Snapshot](t, rec)
	if snap.Version != db.SnapshotVersion || snap.Profile.Name == "" {
		t.Fatalf("bad snapshot: %+v", snap)
	}

	snap.Profile.Name = "Imported Name"
	body := string(mustJSON(t, snap))
	rec = apiDo(t, h, http.MethodPost, "/api/v1/admin/import", body, withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("import status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	rec = apiDo(t, h, http.MethodGet, "/api/v1/profile", "", nil)
	got := decodeBody[db.Profile](t, rec)
	if got.Name != "Imported Name" {
		t.Errorf("public profile name = %q, want Imported Name (import did not reload)", got.Name)
	}
}
