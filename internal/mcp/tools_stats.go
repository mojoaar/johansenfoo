package mcp

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func statPeriodDays(period string) int {
	switch period {
	case "today", "1d":
		return 1
	case "30d":
		return 30
	default:
		return 7
	}
}

func statPeriodStart(days int) time.Time {
	now := time.Now().UTC()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return midnight.AddDate(0, 0, -(days - 1))
}

func (b Backend) getVisitorStats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	period := argString(req, "period")
	if period == "" {
		period = "7d"
	}
	days := statPeriodDays(period)
	since := statPeriodStart(days).Format(time.RFC3339)
	repo := db.NewPageViewRepo(b.DB)

	views, err := repo.CountSince(since)
	if err != nil {
		return jsonResult(nil, err)
	}
	paths, err := repo.TopPaths(since, 5)
	if err != nil {
		return jsonResult(nil, err)
	}
	refs, err := repo.TopReferrers(since, 5)
	if err != nil {
		return jsonResult(nil, err)
	}
	recent, err := repo.Recent(10)
	if err != nil {
		return jsonResult(nil, err)
	}
	dailies := 0
	for i := 0; i < days; i++ {
		day := statPeriodStart(days).AddDate(0, 0, i).Format("2006-01-02")
		n, err := repo.DailyUniqueCount(day)
		if err == nil {
			dailies += n
		}
	}
	return jsonResult(map[string]any{
		"period":        period,
		"views":         views,
		"daily_uniques": dailies,
		"top_paths":     paths,
		"top_referrers": refs,
		"recent":        recent,
	}, nil)
}

func (b Backend) clearVisitorStats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if err := db.NewPageViewRepo(b.DB).Clear(); err != nil {
		return jsonResult(nil, err)
	}
	return jsonResult(map[string]bool{"cleared": true}, nil)
}

func registerStatTools(s *server.MCPServer, b Backend) {
	s.AddTool(mcp.NewTool("get_visitor_stats",
		mcp.WithDescription("Return visitor counts (views, per-day uniques, top pages, top referrers, recent hits) for a period."),
		mcp.WithString("period", mcp.Description("today, 7d (default) or 30d")),
	), b.getVisitorStats)
	s.AddTool(mcp.NewTool("clear_visitor_stats",
		mcp.WithDescription("Delete every recorded page view."),
	), b.clearVisitorStats)
}
