package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func mergeString(target *string, req mcp.CallToolRequest, name string) {
	if hasArg(req, name) {
		*target = argString(req, name)
	}
}

func mergeInt(target *int, req mcp.CallToolRequest, name string) {
	if hasArg(req, name) {
		*target = argInt(req, name)
	}
}

func mergeBool(target *bool, req mcp.CallToolRequest, name string) {
	if hasArg(req, name) {
		*target = argBool(req, name)
	}
}

func requiredID(req mcp.CallToolRequest) (int64, error) {
	if !hasArg(req, "id") {
		return 0, fmt.Errorf("id is required")
	}
	id := int64(argInt(req, "id"))
	if id <= 0 {
		return 0, fmt.Errorf("id must be a positive integer")
	}
	return id, nil
}

func (b Backend) getProfile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p, err := db.NewProfileRepo(b.DB).Get()
	return jsonResult(p, err)
}

func (b Backend) updateProfile(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	p, err := db.NewProfileRepo(b.DB).Get()
	if err != nil {
		return jsonResult(nil, err)
	}
	mergeString(&p.Name, req, "name")
	mergeString(&p.Handle, req, "handle")
	mergeString(&p.Location, req, "location")
	mergeString(&p.DOB, req, "dob")
	mergeString(&p.Tagline, req, "tagline")
	mergeString(&p.HeroBio, req, "hero_bio")
	mergeString(&p.Bio, req, "bio")
	mergeString(&p.AboutPara1, req, "about_para_1")
	mergeString(&p.AboutPara2, req, "about_para_2")
	mergeString(&p.Avatar, req, "avatar")
	if p.Name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	if err := db.NewProfileRepo(b.DB).Update(p); err != nil {
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	return jsonResult(p, nil)
}

func (b Backend) listProjects(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	items, err := db.NewContentRepo(b.DB).Projects()
	return jsonResult(items, err)
}

func (b Backend) createProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if argString(req, "name") == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	p := &db.Project{
		Name:        argString(req, "name"),
		URL:         argString(req, "url"),
		Description: argString(req, "description"),
		Icon:        argString(req, "icon"),
		IsLink:      argBool(req, "is_link"),
		URLLabel:    argString(req, "url_label"),
		Sort:        argInt(req, "sort"),
		Visible:     !hasArg(req, "visible") || argBool(req, "visible"),
	}
	repo := db.NewContentRepo(b.DB)
	id, err := repo.CreateProject(p)
	if err != nil {
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	created, err := repo.Project(id)
	return jsonResult(created, err)
}

func (b Backend) updateProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := requiredID(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	repo := db.NewContentRepo(b.DB)
	p, err := repo.Project(id)
	if err != nil {
		return mcp.NewToolResultError("project not found"), nil
	}
	mergeString(&p.Name, req, "name")
	mergeString(&p.URL, req, "url")
	mergeString(&p.Description, req, "description")
	mergeString(&p.Icon, req, "icon")
	mergeBool(&p.IsLink, req, "is_link")
	mergeString(&p.URLLabel, req, "url_label")
	mergeInt(&p.Sort, req, "sort")
	mergeBool(&p.Visible, req, "visible")
	if err := repo.UpdateProject(p); err != nil {
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	return jsonResult(p, nil)
}

func (b Backend) deleteProject(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := requiredID(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := db.NewContentRepo(b.DB).DeleteProject(id); err != nil {
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	return jsonResult(map[string]int64{"deleted": id}, nil)
}

func (b Backend) listExperience(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	items, err := db.NewContentRepo(b.DB).Experience()
	return jsonResult(items, err)
}

func (b Backend) createExperience(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if argString(req, "role") == "" {
		return mcp.NewToolResultError("role is required"), nil
	}
	e := &db.Experience{
		Years:   argString(req, "years"),
		Role:    argString(req, "role"),
		Company: argString(req, "company"),
		Icon:    argString(req, "icon"),
		Sort:    argInt(req, "sort"),
		Visible: !hasArg(req, "visible") || argBool(req, "visible"),
	}
	repo := db.NewContentRepo(b.DB)
	id, err := repo.CreateExperience(e)
	if err != nil {
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	created, err := repo.ExperienceItem(id)
	return jsonResult(created, err)
}

func (b Backend) updateExperience(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := requiredID(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	repo := db.NewContentRepo(b.DB)
	e, err := repo.ExperienceItem(id)
	if err != nil {
		return mcp.NewToolResultError("experience not found"), nil
	}
	mergeString(&e.Years, req, "years")
	mergeString(&e.Role, req, "role")
	mergeString(&e.Company, req, "company")
	mergeString(&e.Icon, req, "icon")
	mergeInt(&e.Sort, req, "sort")
	mergeBool(&e.Visible, req, "visible")
	if err := repo.UpdateExperience(e); err != nil {
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	return jsonResult(e, nil)
}

func (b Backend) deleteExperience(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := requiredID(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := db.NewContentRepo(b.DB).DeleteExperience(id); err != nil {
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	return jsonResult(map[string]int64{"deleted": id}, nil)
}

func (b Backend) listSkills(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	items, err := db.NewContentRepo(b.DB).Skills()
	return jsonResult(items, err)
}

func (b Backend) createSkill(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if argString(req, "name") == "" {
		return mcp.NewToolResultError("name is required"), nil
	}
	s := &db.Skill{
		Name:    argString(req, "name"),
		Sort:    argInt(req, "sort"),
		Visible: !hasArg(req, "visible") || argBool(req, "visible"),
	}
	repo := db.NewContentRepo(b.DB)
	id, err := repo.CreateSkill(s)
	if err != nil {
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	created, err := repo.Skill(id)
	return jsonResult(created, err)
}

func (b Backend) updateSkill(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := requiredID(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	repo := db.NewContentRepo(b.DB)
	s, err := repo.Skill(id)
	if err != nil {
		return mcp.NewToolResultError("skill not found"), nil
	}
	mergeString(&s.Name, req, "name")
	mergeInt(&s.Sort, req, "sort")
	mergeBool(&s.Visible, req, "visible")
	if err := repo.UpdateSkill(s); err != nil {
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	return jsonResult(s, nil)
}

func (b Backend) deleteSkill(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id, err := requiredID(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := db.NewContentRepo(b.DB).DeleteSkill(id); err != nil {
		return jsonResult(nil, err)
	}
	if err := b.reload(); err != nil {
		return jsonResult(nil, err)
	}
	return jsonResult(map[string]int64{"deleted": id}, nil)
}

func registerContentTools(s *server.MCPServer, b Backend) {
	s.AddTool(mcp.NewTool("get_profile",
		mcp.WithDescription("Return the site owner's profile as JSON."),
	), b.getProfile)
	s.AddTool(mcp.NewTool("update_profile",
		mcp.WithDescription("Update profile fields. Only the fields you pass are changed."),
		mcp.WithString("name"), mcp.WithString("handle"), mcp.WithString("location"),
		mcp.WithString("dob"), mcp.WithString("tagline"), mcp.WithString("hero_bio"),
		mcp.WithString("bio"), mcp.WithString("about_para_1"), mcp.WithString("about_para_2"),
		mcp.WithString("avatar"),
	), b.updateProfile)

	s.AddTool(mcp.NewTool("list_projects",
		mcp.WithDescription("List every project, including hidden ones."),
	), b.listProjects)
	s.AddTool(mcp.NewTool("create_project",
		mcp.WithDescription("Create a project."),
		mcp.WithString("name", mcp.Required()), mcp.WithString("url"),
		mcp.WithString("description"), mcp.WithString("icon"), mcp.WithString("url_label"),
		mcp.WithBoolean("is_link"), mcp.WithBoolean("visible"), mcp.WithNumber("sort"),
	), b.createProject)
	s.AddTool(mcp.NewTool("update_project",
		mcp.WithDescription("Update a project by id; only the fields you pass are changed."),
		mcp.WithNumber("id", mcp.Required()), mcp.WithString("name"), mcp.WithString("url"),
		mcp.WithString("description"), mcp.WithString("icon"), mcp.WithString("url_label"),
		mcp.WithBoolean("is_link"), mcp.WithBoolean("visible"), mcp.WithNumber("sort"),
	), b.updateProject)
	s.AddTool(mcp.NewTool("delete_project",
		mcp.WithDescription("Delete a project by id."),
		mcp.WithNumber("id", mcp.Required()),
	), b.deleteProject)

	s.AddTool(mcp.NewTool("list_experience",
		mcp.WithDescription("List every experience entry, including hidden ones."),
	), b.listExperience)
	s.AddTool(mcp.NewTool("create_experience",
		mcp.WithDescription("Create an experience entry."),
		mcp.WithString("years"), mcp.WithString("role", mcp.Required()), mcp.WithString("company"),
		mcp.WithString("icon"), mcp.WithBoolean("visible"), mcp.WithNumber("sort"),
	), b.createExperience)
	s.AddTool(mcp.NewTool("update_experience",
		mcp.WithDescription("Update an experience entry by id; only the fields you pass are changed."),
		mcp.WithNumber("id", mcp.Required()), mcp.WithString("years"), mcp.WithString("role"),
		mcp.WithString("company"), mcp.WithString("icon"), mcp.WithBoolean("visible"), mcp.WithNumber("sort"),
	), b.updateExperience)
	s.AddTool(mcp.NewTool("delete_experience",
		mcp.WithDescription("Delete an experience entry by id."),
		mcp.WithNumber("id", mcp.Required()),
	), b.deleteExperience)

	s.AddTool(mcp.NewTool("list_skills",
		mcp.WithDescription("List every skill, including hidden ones."),
	), b.listSkills)
	s.AddTool(mcp.NewTool("create_skill",
		mcp.WithDescription("Create a skill."),
		mcp.WithString("name", mcp.Required()), mcp.WithBoolean("visible"), mcp.WithNumber("sort"),
	), b.createSkill)
	s.AddTool(mcp.NewTool("update_skill",
		mcp.WithDescription("Update a skill by id; only the fields you pass are changed."),
		mcp.WithNumber("id", mcp.Required()), mcp.WithString("name"), mcp.WithBoolean("visible"), mcp.WithNumber("sort"),
	), b.updateSkill)
	s.AddTool(mcp.NewTool("delete_skill",
		mcp.WithDescription("Delete a skill by id."),
		mcp.WithNumber("id", mcp.Required()),
	), b.deleteSkill)
}
