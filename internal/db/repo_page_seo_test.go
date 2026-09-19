package db

import (
	"errors"
	"testing"
)

func TestPageSeoUpsertAndGet(t *testing.T) {
	d := seeded(t)
	r := NewPageSeoRepo(d)

	if err := r.Upsert(&PageSeo{Route: "/posts", Title: "Writing", Description: "desc"}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := r.Upsert(&PageSeo{Route: "/posts", Title: "Writing Two", NoIndex: true}); err != nil {
		t.Fatalf("Upsert 2: %v", err)
	}

	got, err := r.Get("/posts")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Writing Two" || !got.NoIndex {
		t.Errorf("page = %+v", got)
	}

	list, err := r.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list = %d, want 1 (upsert on route)", len(list))
	}
}

func TestPageSeoGetMissing(t *testing.T) {
	d := seeded(t)
	if _, err := NewPageSeoRepo(d).Get("/nope"); !errors.Is(err, ErrPageSeoNotFound) {
		t.Fatalf("err = %v, want ErrPageSeoNotFound", err)
	}
}

func TestPageSeoListOrdered(t *testing.T) {
	d := seeded(t)
	r := NewPageSeoRepo(d)
	for _, route := range []string{"/posts", "/", "/tags/go"} {
		if err := r.Upsert(&PageSeo{Route: route, Title: route}); err != nil {
			t.Fatalf("Upsert %s: %v", route, err)
		}
	}
	list, err := r.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 || list[0].Route != "/" || list[1].Route != "/posts" || list[2].Route != "/tags/go" {
		t.Fatalf("order = %+v", list)
	}
}

func TestPageSeoDelete(t *testing.T) {
	d := seeded(t)
	r := NewPageSeoRepo(d)
	if err := r.Upsert(&PageSeo{Route: "/", Title: "Home"}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	got, err := r.Get("/")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if err := r.Delete(got.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := r.Get("/"); !errors.Is(err, ErrPageSeoNotFound) {
		t.Fatalf("after delete err = %v, want ErrPageSeoNotFound", err)
	}
}

func TestPageSeoReplaceAll(t *testing.T) {
	d := seeded(t)
	r := NewPageSeoRepo(d)
	if err := r.Upsert(&PageSeo{Route: "/old", Title: "Old"}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := r.ReplaceAll([]PageSeo{{Route: "/", Title: "Home"}, {Route: "/posts", Title: "Writing"}}); err != nil {
		t.Fatalf("ReplaceAll: %v", err)
	}
	list, err := r.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("list = %d, want 2", len(list))
	}
	if _, err := r.Get("/old"); !errors.Is(err, ErrPageSeoNotFound) {
		t.Fatalf("old route survived ReplaceAll: %v", err)
	}
}
