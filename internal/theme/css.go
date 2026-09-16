package theme

import "strings"

func CSS(t Theme) string {
	var b strings.Builder

	if len(t.Base) > 0 {
		b.WriteString(`[data-theme="` + t.Slug + `"]{`)
		b.WriteString(Vars(t.Base))
		b.WriteString("}\n")
	}

	for _, mode := range []struct {
		name   string
		tokens map[string]string
	}{
		{"light", t.Light},
		{"dark", t.Dark},
	} {
		if len(mode.tokens) == 0 {
			continue
		}
		b.WriteString(`[data-theme="` + t.Slug + `"][data-mode="` + mode.name + `"]{`)
		b.WriteString(Vars(mode.tokens))
		b.WriteString("}\n")
	}

	return b.String()
}
