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
