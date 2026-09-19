package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/mojoaar/johansenfoo/internal/db"
	"github.com/mojoaar/johansenfoo/internal/theme"
)

func themeFromArgs(req mcp.CallToolRequest, slug, name, description string, existing *db.Theme) db.Theme {
	t := db.Theme{Slug: slug, Name: name, Description: description}
	if existing != nil {
		t = *existing
		t.Slug = slug
		if name != "" {
			t.Name = name
		}
		if description != "" {
			t.Description = description
		}
	}
	if m, ok := argStringMap(req, "tokens_base"); ok {
		t.TokensBase = m
	}
	if m, ok := argStringMap(req, "tokens_light"); ok {
		t.TokensLight = m
	}
	if m, ok := argStringMap(req, "tokens_dark"); ok {
		t.TokensDark = m
	}
	return t
}

func themeSlug(name, slug string) string {
	if slug != "" {
		return slug
	}
	return db.Slugify(name)
}

func (b Backend) listThemes(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	themes, err := db.NewThemeRepo(b.DB).List()
	return jsonResult(themes, err)
}

func (b Backend) getTheme(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	slug := argString(req, "slug")
	if slug == "" {
		return mcp.NewToolResultError("slug is required"), nil
	}
	th, err := db.NewThemeRepo(b.DB).GetBySlug(slug)
	if err != nil {
		return mcp.NewToolResultError("theme not found"), nil
	}
	return jsonResult(th, nil)
}

func (b Backend) createTheme(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := argString(req, "name")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	t := themeFromArgs(req, themeSlug(name, argString(req, "slug")), name, argString(req, "description"), nil)
	if err := theme.Validate(theme.Theme{Slug: t.Slug, Base: t.TokensBase, Light: t.TokensLight, Dark: t.TokensDark}); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	repo := db.NewThemeRepo(b.DB)
	id, err := repo.Create(&t)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return mcp.NewToolResultError("slug already in use"), nil
		}
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	created, err := repo.GetByID(id)
	return jsonResult(created, err)
}

func (b Backend) updateTheme(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	slug := argString(req, "slug")
	if slug == "" {
		return mcp.NewToolResultError("slug is required"), nil
	}
	repo := db.NewThemeRepo(b.DB)
	existing, err := repo.GetBySlug(slug)
	if err != nil {
		return mcp.NewToolResultError("theme not found"), nil
	}
	merged := themeFromArgs(req, slug, argString(req, "name"), argString(req, "description"), existing)
	if err := theme.Validate(theme.Theme{Slug: merged.Slug, Base: merged.TokensBase, Light: merged.TokensLight, Dark: merged.TokensDark}); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := repo.Update(&merged); err != nil {
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	updated, err := repo.GetByID(merged.ID)
	return jsonResult(updated, err)
}

func (b Backend) deleteTheme(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	slug := argString(req, "slug")
	if slug == "" {
		return mcp.NewToolResultError("slug is required"), nil
	}
	repo := db.NewThemeRepo(b.DB)
	th, err := repo.GetBySlug(slug)
	if err != nil {
		return mcp.NewToolResultError("theme not found"), nil
	}
	if err := repo.Delete(th.ID); err != nil {
		if err == db.ErrThemeProtected {
			return mcp.NewToolResultError("the base and active themes cannot be deleted"), nil
		}
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	return jsonResult(map[string]string{"deleted": slug}, nil)
}

func (b Backend) setActiveTheme(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	slug := argString(req, "slug")
	if slug == "" {
		return mcp.NewToolResultError("slug is required"), nil
	}
	if err := db.NewThemeRepo(b.DB).SetActive(slug); err != nil {
		return mcp.NewToolResultError("theme not found"), nil
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	return jsonResult(map[string]string{"active": slug}, nil)
}

func seedByFlavour(flavour string) (theme.Theme, bool) {
	want := "catppuccin-" + flavour
	for _, s := range theme.Seeds() {
		if s.Slug == want {
			return s, true
		}
	}
	return theme.Theme{}, false
}

func (b Backend) importTheme(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := argString(req, "name")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	slug := themeSlug(name, argString(req, "slug"))

	var t db.Theme
	if flavour := argString(req, "flavour"); flavour != "" {
		seed, ok := seedByFlavour(flavour)
		if !ok {
			return mcp.NewToolResultError("unknown catppuccin flavour; use latte, frappe, macchiato or mocha"), nil
		}
		t = db.Theme{
			Slug:        slug,
			Name:        name,
			Description: "Imported Catppuccin " + flavour,
			TokensBase:  seed.Base,
			TokensLight: seed.Light,
			TokensDark:  seed.Dark,
		}
	} else {
		t = themeFromArgs(req, slug, name, argString(req, "description"), nil)
		if len(t.TokensLight) == 0 && len(t.TokensDark) == 0 {
			return mcp.NewToolResultError("pass a flavour or token maps"), nil
		}
	}

	if err := theme.Validate(theme.Theme{Slug: t.Slug, Base: t.TokensBase, Light: t.TokensLight, Dark: t.TokensDark}); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	repo := db.NewThemeRepo(b.DB)
	id, err := repo.Create(&t)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return mcp.NewToolResultError("slug already in use"), nil
		}
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	created, err := repo.GetByID(id)
	return jsonResult(created, err)
}

func registerThemeTools(s *server.MCPServer, b Backend) {
	s.AddTool(mcp.NewTool("list_themes",
		mcp.WithDescription("List every theme with its full token maps."),
	), b.listThemes)
	s.AddTool(mcp.NewTool("get_theme",
		mcp.WithDescription("Return one theme by slug."),
		mcp.WithString("slug", mcp.Required()),
	), b.getTheme)
	s.AddTool(mcp.NewTool("create_theme",
		mcp.WithDescription("Create a theme from token maps. Every base token is required in light and dark."),
		mcp.WithString("name", mcp.Required()), mcp.WithString("slug"), mcp.WithString("description"),
		mcp.WithObject("tokens_base"), mcp.WithObject("tokens_light"), mcp.WithObject("tokens_dark"),
	), b.createTheme)
	s.AddTool(mcp.NewTool("update_theme",
		mcp.WithDescription("Update a theme by slug; token maps merge over the stored values."),
		mcp.WithString("slug", mcp.Required()), mcp.WithString("name"), mcp.WithString("description"),
		mcp.WithObject("tokens_base"), mcp.WithObject("tokens_light"), mcp.WithObject("tokens_dark"),
	), b.updateTheme)
	s.AddTool(mcp.NewTool("delete_theme",
		mcp.WithDescription("Delete a theme by slug. The base and active themes are protected."),
		mcp.WithString("slug", mcp.Required()),
	), b.deleteTheme)
	s.AddTool(mcp.NewTool("set_active_theme",
		mcp.WithDescription("Make a theme the site-wide active theme."),
		mcp.WithString("slug", mcp.Required()),
	), b.setActiveTheme)
	s.AddTool(mcp.NewTool("import_theme",
		mcp.WithDescription("Create a theme from a Catppuccin flavour (latte/frappe/macchiato/mocha) or from supplied token maps."),
		mcp.WithString("name", mcp.Required()), mcp.WithString("slug"), mcp.WithString("description"),
		mcp.WithString("flavour"),
		mcp.WithObject("tokens_base"), mcp.WithObject("tokens_light"), mcp.WithObject("tokens_dark"),
	), b.importTheme)
}
