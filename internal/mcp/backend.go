package mcp

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

type Deps struct {
	DB      *sql.DB
	Reload  func() error
	APIKey  func() (string, error)
	Version string
	Started time.Time
	DataDir string
}

type Backend struct {
	DB      *sql.DB
	Reload  func() error
	Started time.Time
	DataDir string
}

func (b Backend) reload() error {
	if b.Reload == nil {
		return nil
	}
	return b.Reload()
}

func jsonResult(v any, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("encode error"), nil
	}
	return mcp.NewToolResultText(string(body)), nil
}

func hasArg(req mcp.CallToolRequest, name string) bool {
	_, ok := req.GetArguments()[name]
	return ok
}

func argString(req mcp.CallToolRequest, name string) string {
	v, _ := req.GetArguments()[name].(string)
	return v
}

func argInt(req mcp.CallToolRequest, name string) int {
	switch v := req.GetArguments()[name].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

func argBool(req mcp.CallToolRequest, name string) bool {
	v, _ := req.GetArguments()[name].(bool)
	return v
}

func argStrings(req mcp.CallToolRequest, name string) []string {
	switch v := req.GetArguments()[name].(type) {
	case string:
		var out []string
		for _, part := range strings.Split(v, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
		return out
	case []any:
		var out []string
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func argStringMap(req mcp.CallToolRequest, name string) (map[string]string, bool, error) {
	raw, ok := req.GetArguments()[name]
	if !ok {
		return nil, false, nil
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return nil, true, err
	}
	var out map[string]string
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, true, err
	}
	return out, true, nil
}
