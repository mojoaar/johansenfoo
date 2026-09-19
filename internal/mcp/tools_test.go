package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func testBackend(t *testing.T) (Backend, *int) {
	t.Helper()
	reloads := 0
	return Backend{
		DB:     testDB(t),
		Reload: func() error { reloads++; return nil },
	}, &reloads
}

func callTool(t *testing.T, fn func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := fn(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
	if err != nil {
		t.Fatalf("tool returned a Go error: %v", err)
	}
	if res == nil {
		t.Fatal("tool returned nil result")
	}
	return res
}

func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool returned an error result: %s", textOf(res))
	}
	return textOf(res)
}

func textOf(res *mcp.CallToolResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

func decode[T any](t *testing.T, res *mcp.CallToolResult) T {
	t.Helper()
	var v T
	body := resultText(t, res)
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return v
}

func TestGetProfileTool(t *testing.T) {
	b, _ := testBackend(t)
	got := decode[db.Profile](t, callTool(t, b.getProfile, nil))
	if got.Name != "Morten Johansen" {
		t.Errorf("name = %q", got.Name)
	}
}

func TestUpdateProfileMergesAndReloads(t *testing.T) {
	b, reloads := testBackend(t)
	callTool(t, b.updateProfile, map[string]any{"tagline": "new tagline"})

	got := decode[db.Profile](t, callTool(t, b.getProfile, nil))
	if got.Name != "Morten Johansen" {
		t.Errorf("name = %q, want unchanged", got.Name)
	}
	if got.Tagline != "new tagline" {
		t.Errorf("tagline = %q, want new tagline", got.Tagline)
	}
	if *reloads != 1 {
		t.Errorf("reloads = %d, want 1", *reloads)
	}
}

func TestMissingRequiredArgIsToolError(t *testing.T) {
	b, _ := testBackend(t)
	res := callTool(t, b.createProject, nil)
	if !res.IsError {
		t.Fatal("expected a tool error for a missing name")
	}
}

func TestProjectLifecycleTools(t *testing.T) {
	b, reloads := testBackend(t)
	created := decode[db.Project](t, callTool(t, b.createProject,
		map[string]any{"name": "MCP Project", "description": "d", "icon": "globe", "visible": true}))
	if created.ID == 0 {
		t.Fatal("create_project returned no id")
	}

	list := decode[[]db.Project](t, callTool(t, b.listProjects, nil))
	found := false
	for _, p := range list {
		if p.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("created project not listed")
	}

	updated := decode[db.Project](t, callTool(t, b.updateProject,
		map[string]any{"id": float64(created.ID), "name": "MCP Project Two"}))
	if updated.Name != "MCP Project Two" || updated.Description != "d" {
		t.Fatalf("updated = %+v", updated)
	}

	callTool(t, b.deleteProject, map[string]any{"id": float64(created.ID)})
	list = decode[[]db.Project](t, callTool(t, b.listProjects, nil))
	for _, p := range list {
		if p.ID == created.ID {
			t.Fatal("project still listed after delete")
		}
	}
	if *reloads != 3 {
		t.Errorf("reloads = %d, want 3", *reloads)
	}
}

func TestExperienceAndSkillTools(t *testing.T) {
	b, _ := testBackend(t)
	exp := decode[db.Experience](t, callTool(t, b.createExperience,
		map[string]any{"years": "2026", "role": "MCP Role", "company": "Co", "icon": "briefcase", "visible": true}))
	if exp.ID == 0 {
		t.Fatal("create_experience returned no id")
	}
	callTool(t, b.deleteExperience, map[string]any{"id": float64(exp.ID)})

	skill := decode[db.Skill](t, callTool(t, b.createSkill, map[string]any{"name": "MCP Skill", "visible": true}))
	if skill.ID == 0 {
		t.Fatal("create_skill returned no id")
	}
	updated := decode[db.Skill](t, callTool(t, b.updateSkill, map[string]any{"id": float64(skill.ID), "name": "MCP Skill Two"}))
	if updated.Name != "MCP Skill Two" {
		t.Fatalf("updated skill = %+v", updated)
	}
	callTool(t, b.deleteSkill, map[string]any{"id": float64(skill.ID)})
}

func findPost(posts []db.Post, id int64) bool {
	for _, p := range posts {
		if p.ID == id {
			return true
		}
	}
	return false
}

func TestPostToolsLifecycle(t *testing.T) {
	b, reloads := testBackend(t)
	created := decode[db.Post](t, callTool(t, b.createPost, map[string]any{
		"title": "MCP Post", "slug": "mcp-post", "summary": "s", "body_md": "# hi",
		"status": "published", "tags": "go, testing",
	}))
	if created.ID == 0 || created.PublishedAt == nil {
		t.Fatalf("created = %+v", created)
	}
	if len(created.Tags) != 2 {
		t.Fatalf("tags = %+v", created.Tags)
	}

	if posts := decode[[]db.Post](t, callTool(t, b.listPosts, nil)); !findPost(posts, created.ID) {
		t.Error("published post not listed")
	}
	if tags := decode[[]db.Tag](t, callTool(t, b.listTags, nil)); len(tags) != 2 {
		t.Errorf("tags = %+v", tags)
	}

	got := decode[db.Post](t, callTool(t, b.getPost, map[string]any{"slug": "mcp-post"}))
	if got.Title != "MCP Post" {
		t.Errorf("get_post = %+v", got)
	}

	updated := decode[db.Post](t, callTool(t, b.updatePost, map[string]any{"id": float64(created.ID), "title": "MCP Post Two"}))
	if updated.Title != "MCP Post Two" || updated.BodyMD != "# hi" {
		t.Fatalf("updated = %+v", updated)
	}

	callTool(t, b.unpublishPost, map[string]any{"id": float64(created.ID)})
	if posts := decode[[]db.Post](t, callTool(t, b.listPosts, nil)); findPost(posts, created.ID) {
		t.Error("unpublished post still listed")
	}
	if posts := decode[[]db.Post](t, callTool(t, b.listPosts, map[string]any{"include_drafts": true})); !findPost(posts, created.ID) {
		t.Error("draft not listed with include_drafts")
	}

	callTool(t, b.publishPost, map[string]any{"id": float64(created.ID)})
	callTool(t, b.deletePost, map[string]any{"id": float64(created.ID)})
	if posts := decode[[]db.Post](t, callTool(t, b.listPosts, map[string]any{"include_drafts": true})); findPost(posts, created.ID) {
		t.Error("post still listed after delete")
	}
	if *reloads != 5 {
		t.Errorf("reloads = %d, want 5", *reloads)
	}
}

func TestPostDuplicateSlugIsToolError(t *testing.T) {
	b, _ := testBackend(t)
	callTool(t, b.createPost, map[string]any{"title": "One", "slug": "dup", "status": "draft"})
	res := callTool(t, b.createPost, map[string]any{"title": "Two", "slug": "dup", "status": "draft"})
	if !res.IsError {
		t.Fatal("expected a tool error for a duplicate slug")
	}
}

func TestSeoTools(t *testing.T) {
	b, _ := testBackend(t)
	if err := db.NewPageSeoRepo(b.DB).Upsert(&db.PageSeo{Route: "/posts", Title: "Keep"}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	out := decode[map[string]any](t, callTool(t, b.getSeoSettings, nil))
	if out["site_title"] != "Morten Johansen | johansen.foo" {
		t.Errorf("site_title = %v", out["site_title"])
	}

	callTool(t, b.updateSeoSettings, map[string]any{"twitter_site": "@mcp"})
	settings := db.NewSettingsRepo(b.DB)
	if v, _ := settings.Get("twitter_site"); v != "@mcp" {
		t.Errorf("twitter_site = %q", v)
	}
	page, err := db.NewPageSeoRepo(b.DB).Get("/posts")
	if err != nil || page.Title != "Keep" {
		t.Errorf("page_seo not preserved: %+v, err=%v", page, err)
	}
}

func TestPostsToggleTools(t *testing.T) {
	b, reloads := testBackend(t)
	callTool(t, b.disablePosts, nil)
	if v, _ := db.NewSettingsRepo(b.DB).Get("posts_enabled"); v != "false" {
		t.Errorf("posts_enabled = %q, want false", v)
	}
	callTool(t, b.enablePosts, nil)
	if v, _ := db.NewSettingsRepo(b.DB).Get("posts_enabled"); v != "true" {
		t.Errorf("posts_enabled = %q, want true", v)
	}
	if *reloads != 2 {
		t.Errorf("reloads = %d, want 2", *reloads)
	}
}

func TestExportImportTools(t *testing.T) {
	b, _ := testBackend(t)
	snap := decode[db.Snapshot](t, callTool(t, b.exportContent, nil))
	if snap.Version != db.SnapshotVersion || snap.Profile.Name == "" {
		t.Fatalf("snapshot = %+v", snap)
	}

	snap.Profile.Name = "MCP Imported"
	callTool(t, b.importContent, map[string]any{"snapshot": snap})
	got := decode[db.Profile](t, callTool(t, b.getProfile, nil))
	if got.Name != "MCP Imported" {
		t.Errorf("profile name = %q, want MCP Imported", got.Name)
	}
}

func TestImportContentRejectsMalformed(t *testing.T) {
	b, _ := testBackend(t)
	res := callTool(t, b.importContent, map[string]any{"snapshot": map[string]any{"version": 999}})
	if !res.IsError {
		t.Fatal("expected a tool error for an unsupported snapshot")
	}
}
