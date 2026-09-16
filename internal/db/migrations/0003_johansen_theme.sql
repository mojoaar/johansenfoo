INSERT INTO theme (slug, name, description, tokens_base, tokens_light, tokens_dark, sort, created_at, updated_at)
VALUES (
    'johansen',
    'Johansen',
    'The original johansen.foo theme.',
    '{"--font-mono":"\"JetBrains Mono\", \"Fira Code\", \"Cascadia Code\", ui-monospace, monospace","--font-sans":"-apple-system, BlinkMacSystemFont, \"Segoe UI\", Roboto, sans-serif","--radius":"10px"}',
    '{"--accent":"#5b4edc","--accent-glow":"rgba(91, 78, 220, 0.15)","--accent2":"#7c6af7","--bg":"#f4f6fb","--bg2":"#ffffff","--bg3":"#eef1f8","--border":"#d0d7e3","--green":"#059669","--shadow":"0 4px 24px rgba(0, 0, 0, 0.08)","--text":"#1a1f2e","--text-muted":"#5a6278"}',
    '{"--accent":"#7c6af7","--accent-glow":"rgba(124, 106, 247, 0.25)","--accent2":"#a78bfa","--bg":"#0f1117","--bg2":"#161b27","--bg3":"#1e2433","--border":"#2a3045","--green":"#34d399","--shadow":"0 4px 32px rgba(0, 0, 0, 0.5)","--text":"#e2e8f0","--text-muted":"#8892a4"}',
    0,
    strftime('%Y-%m-%dT%H:%M:%SZ','now'),
    strftime('%Y-%m-%dT%H:%M:%SZ','now')
)
ON CONFLICT(slug) DO UPDATE SET
    name         = excluded.name,
    description  = excluded.description,
    tokens_base  = excluded.tokens_base,
    tokens_light = excluded.tokens_light,
    tokens_dark  = excluded.tokens_dark,
    sort         = excluded.sort,
    updated_at   = excluded.updated_at;
