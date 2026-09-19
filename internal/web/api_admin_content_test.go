package web

import (
	"net/http"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAdminAPIContentCRUDLifecycle(t *testing.T) {
	cases := []struct {
		name       string
		path       string
		create     string
		update     string
		assertName func(t *testing.T, body []byte) string
	}{
		{
			name:   "projects",
			path:   "/api/v1/admin/projects",
			create: `{"name":"crud-project","description":"d","icon":"globe","visible":false,"sort":99}`,
			update: `{"name":"crud-project-2","description":"d","icon":"globe","visible":true,"sort":99}`,
			assertName: func(t *testing.T, body []byte) string {
				return decodeJSONBytes[db.Project](t, body).Name
			},
		},
		{
			name:   "experience",
			path:   "/api/v1/admin/experience",
			create: `{"years":"2026","role":"crud-role","company":"c","icon":"briefcase","visible":false,"sort":99}`,
			update: `{"years":"2026","role":"crud-role-2","company":"c","icon":"briefcase","visible":true,"sort":99}`,
			assertName: func(t *testing.T, body []byte) string {
				return decodeJSONBytes[db.Experience](t, body).Role
			},
		},
		{
			name:   "skills",
			path:   "/api/v1/admin/skills",
			create: `{"name":"crud-skill","visible":false,"sort":99}`,
			update: `{"name":"crud-skill-2","visible":true,"sort":99}`,
			assertName: func(t *testing.T, body []byte) string {
				return decodeJSONBytes[db.Skill](t, body).Name
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newTestDB(t)
			store, _ := NewContentStore(d)
			h := loggedInHandler(t, d, store)

			rec := apiDo(t, h, http.MethodPost, tc.path, tc.create, withSession)
			if rec.Code != http.StatusCreated {
				t.Fatalf("create status = %d, want 201; body=%s", rec.Code, rec.Body.String())
			}
			created := decodeBody[struct {
				ID int64 `json:"id"`
			}](t, rec)
			if created.ID == 0 {
				t.Fatal("create returned id 0")
			}
			path := tc.path + "/" + itoa(created.ID)

			rec = apiDo(t, h, http.MethodGet, path, "", withSession)
			if rec.Code != http.StatusOK {
				t.Fatalf("get status = %d, want 200", rec.Code)
			}

			rec = apiDo(t, h, http.MethodPut, path, tc.update, withSession)
			if rec.Code != http.StatusOK {
				t.Fatalf("update status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
			if name := tc.assertName(t, rec.Body.Bytes()); name == "" {
				t.Fatal("update returned an empty name")
			}

			rec = apiDo(t, h, http.MethodDelete, path, "", withSession)
			if rec.Code != http.StatusNoContent {
				t.Fatalf("delete status = %d, want 204", rec.Code)
			}
			if rec = apiDo(t, h, http.MethodDelete, path, "", withSession); rec.Code != http.StatusNotFound {
				t.Fatalf("second delete status = %d, want 404", rec.Code)
			}
		})
	}
}

func TestAdminAPIContentRequiresName(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	paths := []string{
		"/api/v1/admin/projects",
		"/api/v1/admin/experience",
		"/api/v1/admin/skills",
	}
	for _, path := range paths {
		rec := apiDo(t, h, http.MethodPost, path, `{}`, withSession)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", path, rec.Code)
		}
	}
}

func TestAdminAPIContentListIncludesHidden(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/projects", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := decodeBody[[]db.Project](t, rec); len(got) != 9 {
		t.Errorf("projects = %d, want 9 (all rows, including hidden)", len(got))
	}
}
