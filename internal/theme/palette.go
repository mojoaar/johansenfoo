package theme

import catppuccingo "github.com/catppuccin/go"

type Palette struct {
	Bg      string
	Bg2     string
	Bg3     string
	Border  string
	Text    string
	Muted   string
	Accent  string
	Accent2 string
	Green   string
}

func (p Palette) Tokens() map[string]string {
	return map[string]string{
		"--bg":          p.Bg,
		"--bg2":         p.Bg2,
		"--bg3":         p.Bg3,
		"--border":      p.Border,
		"--text":        p.Text,
		"--text-muted":  p.Muted,
		"--accent":      p.Accent,
		"--accent2":     p.Accent2,
		"--accent-glow": p.Accent,
		"--green":       p.Green,
		"--shadow":      "0 4px 24px rgba(0, 0, 0, 0.35)",
	}
}

func seedTheme(slug, name, description string, light, dark Palette) Theme {
	return Theme{
		Slug:        slug,
		Name:        name,
		Description: description,
		Base: map[string]string{
			"--radius":    "10px",
			"--font-mono": `"JetBrains Mono", "Fira Code", ui-monospace, monospace`,
			"--font-sans": `-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif`,
		},
		Light: light.Tokens(),
		Dark:  dark.Tokens(),
	}
}

func catppuccinPalette(f catppuccingo.Flavor) Palette {
	return Palette{
		Bg:      f.Base().Hex,
		Bg2:     f.Mantle().Hex,
		Bg3:     f.Surface0().Hex,
		Border:  f.Surface1().Hex,
		Text:    f.Text().Hex,
		Muted:   f.Overlay1().Hex,
		Accent:  f.Mauve().Hex,
		Accent2: f.Lavender().Hex,
		Green:   f.Green().Hex,
	}
}

var nord = struct{ light, dark Palette }{
	light: Palette{"#eceff4", "#e5e9f0", "#d8dee9", "#c2c9d6", "#2e3440", "#4c566a", "#5e81ac", "#81a1c1", "#a3be8c"},
	dark:  Palette{"#2e3440", "#3b4252", "#434c5e", "#4c566a", "#eceff4", "#8794a8", "#88c0d0", "#81a1c1", "#a3be8c"},
}

var rosePine = struct{ light, dark Palette }{
	light: Palette{"#faf4ed", "#fffaf3", "#f2e9e1", "#dfdad9", "#575279", "#797593", "#907aa9", "#b4637a", "#56949f"},
	dark:  Palette{"#191724", "#1f1d2e", "#26233a", "#403d52", "#e0def4", "#908caa", "#c4a7e7", "#ebbcba", "#9ccfd8"},
}

var tokyoNight = struct{ light, dark Palette }{
	light: Palette{"#d5d6db", "#e9e9ec", "#cbccd1", "#b4b5be", "#343b58", "#6172b0", "#34548a", "#5a4a78", "#485e30"},
	dark:  Palette{"#1a1b26", "#24283b", "#292e42", "#3b4261", "#c0caf5", "#565f89", "#7aa2f7", "#bb9af7", "#9ece6a"},
}

var gruvbox = struct{ light, dark Palette }{
	light: Palette{"#fbf1c7", "#f2e5bc", "#ebdbb2", "#d5c4a1", "#3c3836", "#7c6f64", "#b57614", "#8f3f71", "#79740e"},
	dark:  Palette{"#282828", "#3c3836", "#504945", "#665c54", "#ebdbb2", "#a89984", "#d79921", "#d3869b", "#b8bb26"},
}

var everforest = struct{ light, dark Palette }{
	light: Palette{"#fdf6e3", "#f4f0d9", "#efebd4", "#e0dcc7", "#5c6a72", "#829181", "#8da101", "#dfa000", "#8da101"},
	dark:  Palette{"#2d353b", "#343f44", "#3d484d", "#475258", "#d3c6aa", "#859289", "#a7c080", "#d699b6", "#a7c080"},
}

var solarized = struct{ light, dark Palette }{
	light: Palette{"#fdf6e3", "#eee8d5", "#e4ddc9", "#d3cbb7", "#657b83", "#93a1a1", "#268bd2", "#6c71c4", "#859900"},
	dark:  Palette{"#002b36", "#073642", "#0a4455", "#15505f", "#93a1a1", "#657b83", "#268bd2", "#b58900", "#859900"},
}

func Seeds() []Theme {
	latte := catppuccinPalette(catppuccingo.Latte)
	mocha := catppuccinPalette(catppuccingo.Mocha)
	return []Theme{
		seedTheme("catppuccin-latte", "Catppuccin Latte", "The light Catppuccin flavour.", latte, mocha),
		seedTheme("catppuccin-frappe", "Catppuccin Frappé", "A low-contrast dark Catppuccin flavour.", latte, catppuccinPalette(catppuccingo.Frappe)),
		seedTheme("catppuccin-macchiato", "Catppuccin Macchiato", "A medium-contrast dark Catppuccin flavour.", latte, catppuccinPalette(catppuccingo.Macchiato)),
		seedTheme("catppuccin-mocha", "Catppuccin Mocha", "The darkest Catppuccin flavour.", latte, mocha),
		seedTheme("nord", "Nord", "An arctic, north-bluish colour palette.", nord.light, nord.dark),
		seedTheme("rose-pine", "Rosé Pine", "All natural pine, faux fur and a bit of soho vibes.", rosePine.light, rosePine.dark),
		seedTheme("tokyo-night", "Tokyo Night", "A clean, dark theme that celebrates the lights of downtown Tokyo.", tokyoNight.light, tokyoNight.dark),
		seedTheme("gruvbox", "Gruvbox", "A retro groove colour scheme.", gruvbox.light, gruvbox.dark),
		seedTheme("everforest", "Everforest", "A comfy, green colour scheme.", everforest.light, everforest.dark),
		seedTheme("solarized", "Solarized", "Precision colours with a low-contrast palette.", solarized.light, solarized.dark),
	}
}
