package theme

func Johansen() Theme {
	return Theme{
		Slug:        "johansen",
		Name:        "Johansen",
		Description: "The original johansen.foo theme.",
		Base: map[string]string{
			"--radius":    "10px",
			"--font-mono": `"JetBrains Mono", "Fira Code", "Cascadia Code", ui-monospace, monospace`,
			"--font-sans": `-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif`,
		},
		Dark: map[string]string{
			"--bg":          "#0f1117",
			"--bg2":         "#161b27",
			"--bg3":         "#1e2433",
			"--border":      "#2a3045",
			"--text":        "#e2e8f0",
			"--text-muted":  "#8892a4",
			"--accent":      "#7c6af7",
			"--accent2":     "#a78bfa",
			"--accent-glow": "rgba(124, 106, 247, 0.25)",
			"--green":       "#34d399",
			"--shadow":      "0 4px 32px rgba(0, 0, 0, 0.5)",
		},
		Light: map[string]string{
			"--bg":          "#f4f6fb",
			"--bg2":         "#ffffff",
			"--bg3":         "#eef1f8",
			"--border":      "#d0d7e3",
			"--text":        "#1a1f2e",
			"--text-muted":  "#5a6278",
			"--accent":      "#5b4edc",
			"--accent2":     "#7c6af7",
			"--accent-glow": "rgba(91, 78, 220, 0.15)",
			"--green":       "#059669",
			"--shadow":      "0 4px 24px rgba(0, 0, 0, 0.08)",
		},
	}
}
