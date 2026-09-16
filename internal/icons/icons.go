package icons

import (
	"embed"
	"html/template"
	"path"
	"regexp"
	"strings"
	"sync"
)

//go:embed svg/*.svg
var svgFS embed.FS

var (
	dimensionRE = regexp.MustCompile(`\s(width|height)="[^"]*"`)
	classRE     = regexp.MustCompile(`\sclass="[^"]*"`)
	fillRE      = regexp.MustCompile(`\sfill="[^"]*"`)
	cache       sync.Map
)

func Exists(name string) bool {
	_, err := svgFS.ReadFile(path.Join("svg", name+".svg"))
	return err == nil
}

func Inline(name, class string) template.HTML {
	if v, ok := cache.Load(name); ok {
		return render(v.(string), class)
	}
	raw, err := svgFS.ReadFile(path.Join("svg", name+".svg"))
	if err != nil {
		return ""
	}
	body := string(raw)
	cache.Store(name, body)
	return render(body, class)
}

func render(body, class string) template.HTML {
	start := strings.Index(body, "<svg")
	if start < 0 {
		return ""
	}
	body = body[start:]

	body = dimensionRE.ReplaceAllString(body, "")
	body = classRE.ReplaceAllString(body, "")
	body = fillRE.ReplaceAllStringFunc(body, func(match string) string {
		if strings.Contains(match, `"none"`) {
			return match
		}
		return ` fill="currentColor"`
	})

	end := strings.IndexByte(body, '>')
	if end < 0 {
		return ""
	}
	attrs := ` class="` + class + `" aria-hidden="true" focusable="false"`
	if !strings.Contains(body[:end], "fill=") {
		attrs += ` fill="currentColor"`
	}
	return template.HTML(body[:end] + attrs + body[end:])
}
