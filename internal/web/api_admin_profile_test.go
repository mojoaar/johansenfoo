package web

import (
	"net/http"
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
