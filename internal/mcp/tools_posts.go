package mcp

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func tagsFromArgs(req mcp.CallToolRequest) []db.Tag {
	var tags []db.Tag
	for _, name := range argStrings(req, "tags") {
		tags = append(tags, db.Tag{Name: name, Slug: db.Slugify(name)})
	}
	return tags
}

func validStatus(status string) bool {
	return status == "" || status == "draft" || status == "published"
}

func ensurePublished(p *db.Post) {
	if p.Status == "published" && p.PublishedAt == nil {
		now := time.Now().UTC()
		p.PublishedAt = &now
	}
}

func (b Backend) listPosts(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	repo := db.NewPostRepo(b.DB)
	limit := argInt(req, "limit")
	if limit <= 0 {
		limit = 100
	}
	var (
		posts []db.Post
		err   error
	)
	if argBool(req, "include_drafts") {
		posts, err = repo.All()
	} else if tag := argString(req, "tag"); tag != "" {
		posts, err = repo.ByTag(tag, limit, 0)
	} else {
		posts, err = repo.Published(limit, 0)
	}
	return jsonResult(posts, err)
}

func (b Backend) getPost(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	slug := argString(req, "slug")
	if slug == "" {
		return mcp.NewToolResultError("slug is required"), nil
	}
	posts, err := db.NewPostRepo(b.DB).All()
	if err != nil {
		return jsonResult(nil, err)
	}
	for _, p := range posts {
		if p.Slug == slug {
			return jsonResult(p, nil)
		}
	}
	return mcp.NewToolResultError("post not found"), nil
}

func (b Backend) createPost(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title := argString(req, "title")
	if title == "" {
		return mcp.NewToolResultError("title is required"), nil
	}
	status := argString(req, "status")
	if !validStatus(status) {
		return mcp.NewToolResultError("status must be draft or published"), nil
	}
	if status == "" {
		status = "draft"
	}
	slug := db.Slugify(argString(req, "slug"))
	if slug == "" {
		slug = db.Slugify(title)
	}
	if slug == "" {
		return mcp.NewToolResultError("slug could not be derived from the title; pass a slug"), nil
	}
	p := &db.Post{
		Slug:           slug,
		Title:          title,
		Summary:        argString(req, "summary"),
		BodyMD:         argString(req, "body_md"),
		Status:         status,
		HeroImageURL:   argString(req, "hero_image_url"),
		HeroImageAlt:   argString(req, "hero_image_alt"),
		SEOTitle:       argString(req, "seo_title"),
		SEODescription: argString(req, "seo_description"),
		OGImageURL:     argString(req, "og_image_url"),
		CanonicalURL:   argString(req, "canonical_url"),
		NoIndex:        argBool(req, "noindex"),
		Tags:           tagsFromArgs(req),
	}
	ensurePublished(p)
	repo := db.NewPostRepo(b.DB)
	id, err := repo.Create(p)
	if err != nil {
		if db.IsUniqueViolation(err) {
			return mcp.NewToolResultError("slug already in use"), nil
		}
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	created, err := repo.ByID(id)
	return jsonResult(created, err)
}

func (b Backend) updatePost(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := requiredID(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	repo := db.NewPostRepo(b.DB)
	p, err := repo.ByID(id)
	if err != nil {
		return mcp.NewToolResultError("post not found"), nil
	}
	mergeString(&p.Title, req, "title")
	mergeString(&p.Slug, req, "slug")
	mergeString(&p.Summary, req, "summary")
	mergeString(&p.BodyMD, req, "body_md")
	mergeString(&p.HeroImageURL, req, "hero_image_url")
	mergeString(&p.HeroImageAlt, req, "hero_image_alt")
	mergeString(&p.SEOTitle, req, "seo_title")
	mergeString(&p.SEODescription, req, "seo_description")
	mergeString(&p.OGImageURL, req, "og_image_url")
	mergeString(&p.CanonicalURL, req, "canonical_url")
	mergeBool(&p.NoIndex, req, "noindex")
	if hasArg(req, "status") {
		status := argString(req, "status")
		if !validStatus(status) {
			return mcp.NewToolResultError("status must be draft or published"), nil
		}
		p.Status = status
	}
	if hasArg(req, "tags") {
		p.Tags = tagsFromArgs(req)
	}
	if p.Slug == "" {
		p.Slug = db.Slugify(p.Title)
	}
	if p.Slug == "" || p.Title == "" {
		return mcp.NewToolResultError("title and slug are required"), nil
	}
	ensurePublished(p)
	if err := repo.Update(p); err != nil {
		if db.IsUniqueViolation(err) {
			return mcp.NewToolResultError("slug already in use"), nil
		}
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	updated, err := repo.ByID(id)
	return jsonResult(updated, err)
}

func (b Backend) deletePost(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := requiredID(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := db.NewPostRepo(b.DB).Delete(id); err != nil {
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	return jsonResult(map[string]int64{"deleted": id}, nil)
}

func (b Backend) setPostStatus(status string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := requiredID(req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		repo := db.NewPostRepo(b.DB)
		if _, err := repo.ByID(id); err != nil {
			return mcp.NewToolResultError("post not found"), nil
		}
		if err := repo.SetStatus(id, status); err != nil {
			return jsonResult(nil, err)
		}
		if err := b.reload(); err != nil {
			return jsonResult(nil, err)
		}
		updated, err := repo.ByID(id)
		return jsonResult(updated, err)
	}
}

func (b Backend) publishPost(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return b.setPostStatus("published")(ctx, req)
}

func (b Backend) unpublishPost(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return b.setPostStatus("draft")(ctx, req)
}

func (b Backend) listTags(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tags, err := db.NewPostRepo(b.DB).Tags()
	return jsonResult(tags, err)
}

func registerPostTools(s *server.MCPServer, b Backend) {
	s.AddTool(mcp.NewTool("list_posts",
		mcp.WithDescription("List posts. Drafts are hidden unless include_drafts is true."),
		mcp.WithBoolean("include_drafts"), mcp.WithString("tag"), mcp.WithNumber("limit"),
	), b.listPosts)
	s.AddTool(mcp.NewTool("get_post",
		mcp.WithDescription("Return a post by slug, draft or published."),
		mcp.WithString("slug", mcp.Required()),
	), b.getPost)
	s.AddTool(mcp.NewTool("create_post",
		mcp.WithDescription("Create a post. tags accepts a comma-separated string or an array of strings."),
		mcp.WithString("title", mcp.Required()), mcp.WithString("slug"), mcp.WithString("summary"),
		mcp.WithString("body_md"), mcp.WithString("status"), mcp.WithString("tags"),
		mcp.WithString("hero_image_url"), mcp.WithString("hero_image_alt"),
		mcp.WithString("seo_title"), mcp.WithString("seo_description"),
		mcp.WithString("og_image_url"), mcp.WithString("canonical_url"), mcp.WithBoolean("noindex"),
	), b.createPost)
	s.AddTool(mcp.NewTool("update_post",
		mcp.WithDescription("Update a post by id; only the fields you pass are changed. tags replaces the whole set."),
		mcp.WithNumber("id", mcp.Required()), mcp.WithString("title"), mcp.WithString("slug"),
		mcp.WithString("summary"), mcp.WithString("body_md"), mcp.WithString("status"), mcp.WithString("tags"),
		mcp.WithString("hero_image_url"), mcp.WithString("hero_image_alt"),
		mcp.WithString("seo_title"), mcp.WithString("seo_description"),
		mcp.WithString("og_image_url"), mcp.WithString("canonical_url"), mcp.WithBoolean("noindex"),
	), b.updatePost)
	s.AddTool(mcp.NewTool("delete_post",
		mcp.WithDescription("Delete a post by id."),
		mcp.WithNumber("id", mcp.Required()),
	), b.deletePost)
	s.AddTool(mcp.NewTool("publish_post",
		mcp.WithDescription("Publish a post by id."),
		mcp.WithNumber("id", mcp.Required()),
	), b.publishPost)
	s.AddTool(mcp.NewTool("unpublish_post",
		mcp.WithDescription("Return a post to draft by id."),
		mcp.WithNumber("id", mcp.Required()),
	), b.unpublishPost)
	s.AddTool(mcp.NewTool("list_tags",
		mcp.WithDescription("List every tag."),
	), b.listTags)
}
