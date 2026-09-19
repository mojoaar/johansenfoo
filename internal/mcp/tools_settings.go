package mcp

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/mojoaar/johansenfoo/internal/db"
)

var seoStringKeys = []string{
	"site_title", "title_template", "seo_description", "og_image_url",
	"og_type", "twitter_card", "twitter_site", "canonical_base_url", "robots_txt",
}

func (b Backend) seoPayload() (map[string]any, error) {
	settings, err := db.NewSettingsRepo(b.DB).All()
	if err != nil {
		return nil, err
	}
	pages, err := db.NewPageSeoRepo(b.DB).List()
	if err != nil {
		return nil, err
	}
	if pages == nil {
		pages = []db.PageSeo{}
	}
	out := map[string]any{
		"noindex":         settings["noindex"] == "true",
		"sitemap_enabled": settings["sitemap_enabled"] != "false",
		"pages":           pages,
	}
	for _, key := range seoStringKeys {
		out[key] = settings[key]
	}
	return out, nil
}

func (b Backend) getSeoSettings(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	out, err := b.seoPayload()
	return jsonResult(out, err)
}

func (b Backend) updateSeoSettings(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repo := db.NewSettingsRepo(b.DB)
	for _, key := range seoStringKeys {
		if hasArg(req, key) {
			if err := repo.Set(key, argString(req, key)); err != nil {
				return jsonResult(nil, err)
			}
		}
	}
	for _, key := range []string{"noindex", "sitemap_enabled"} {
		if hasArg(req, key) {
			if err := repo.Set(key, strconv.FormatBool(argBool(req, key))); err != nil {
				return jsonResult(nil, err)
			}
		}
	}
	if hasArg(req, "pages") {
		body, err := json.Marshal(req.GetArguments()["pages"])
		if err != nil {
			return mcp.NewToolResultError("invalid pages"), nil
		}
		var pages []db.PageSeo
		if err := json.Unmarshal(body, &pages); err != nil {
			return mcp.NewToolResultError("invalid pages"), nil
		}
		if err := db.NewPageSeoRepo(b.DB).ReplaceAll(pages); err != nil {
			return mcp.NewToolResultError("invalid pages"), nil
		}
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	out, err := b.seoPayload()
	return jsonResult(out, err)
}

func (b Backend) setPostsEnabled(enabled bool) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if err := db.NewSettingsRepo(b.DB).Set("posts_enabled", strconv.FormatBool(enabled)); err != nil {
			return jsonResult(nil, err)
		}
		if err := b.reload(); err != nil {
			return jsonResult(nil, err)
		}
		return jsonResult(map[string]bool{"enabled": enabled}, nil)
	}
}

func (b Backend) enablePosts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return b.setPostsEnabled(true)(ctx, req)
}

func (b Backend) disablePosts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return b.setPostsEnabled(false)(ctx, req)
}

func (b Backend) exportContent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	snap, err := db.Export(b.DB)
	return jsonResult(snap, err)
}

func (b Backend) importContent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	raw, ok := req.GetArguments()["snapshot"]
	if !ok {
		return mcp.NewToolResultError("snapshot is required"), nil
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return mcp.NewToolResultError("invalid snapshot"), nil
	}
	var snap db.Snapshot
	if err := json.Unmarshal(body, &snap); err != nil {
		return mcp.NewToolResultError("invalid snapshot"), nil
	}
	if err := db.Import(b.DB, &snap); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	return jsonResult(map[string]bool{"imported": true}, nil)
}

func registerSettingTools(s *server.MCPServer, b Backend) {
	s.AddTool(mcp.NewTool("get_seo_settings",
		mcp.WithDescription("Return the global SEO settings and per-page overrides."),
	), b.getSeoSettings)
	s.AddTool(mcp.NewTool("update_seo_settings",
		mcp.WithDescription("Update SEO settings. Only the fields you pass are changed; pages replaces the whole override set when provided."),
		mcp.WithString("site_title"), mcp.WithString("title_template"), mcp.WithString("seo_description"),
		mcp.WithString("og_image_url"), mcp.WithString("og_type"), mcp.WithString("twitter_card"),
		mcp.WithString("twitter_site"), mcp.WithString("canonical_base_url"), mcp.WithString("robots_txt"),
		mcp.WithBoolean("noindex"), mcp.WithBoolean("sitemap_enabled"),
	), b.updateSeoSettings)
	s.AddTool(mcp.NewTool("enable_posts",
		mcp.WithDescription("Turn the public posts surface back on."),
	), b.enablePosts)
	s.AddTool(mcp.NewTool("disable_posts",
		mcp.WithDescription("Turn the public posts surface off; posts are kept in the database."),
	), b.disablePosts)
	s.AddTool(mcp.NewTool("export_content",
		mcp.WithDescription("Return a whole-content snapshot (secrets excluded)."),
	), b.exportContent)
	s.AddTool(mcp.NewTool("import_content",
		mcp.WithDescription("Restore content from a snapshot previously returned by export_content."),
		mcp.WithObject("snapshot", mcp.Required()),
	), b.importContent)
}
