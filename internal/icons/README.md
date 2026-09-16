# Icons

Vendored SVG icons, embedded into the binary with `go:embed` and rendered as
inline SVG at request time. Nothing here is fetched at runtime, so the icon
path adds no third-party origin.

## Provenance

| Set | Package | Version | Licence |
| --- | --- | --- | --- |
| Lucide outline icons (twelve shared glyphs) | [`lucide-static`](https://www.npmjs.com/package/lucide-static) | `0.503.0` | ISC (MIT for Feather-derived glyphs) |
| `arrow-up-right-from-square` | [`lucide-static`](https://www.npmjs.com/package/lucide-static) | `1.46.0` | ISC |
| Brand icons | [`simple-icons`](https://www.npmjs.com/package/simple-icons) | `13.21.0` | CC0-1.0 |

Lucide files keep the upstream `@license` comment header they were downloaded
with; Simple Icons files carry no such header upstream. `Inline` starts the
rendered output at the `<svg>` tag, so any leading comment is dropped.

### Version selection

Parity with the live site governs. The live site drew its outline glyphs from
`lucide-static@0.503.0`, so the twelve shared glyphs are vendored from that
exact version. Five of them — `message-square`, `mail`, `flame`, `kanban`, and
`brain` — were redrawn upstream between `0.503.0` and `1.46.0`; taking them
from `1.46.0` would render them differently from the live site. The other
seven are byte-identical across the two versions.

`arrow-up-right-from-square.svg` does not exist at `0.503.0` (the name appears
only in the `1.x` line), so that single glyph comes from `1.46.0`.

Simple Icons are a deliberate, accepted deviation: the live site used Font
Awesome brand glyphs, so there is no parity constraint on the brand marks.
`simple-icons` removed `linkedin.svg` in `14.0.0`; `13.21.0` is the last release
that ships LinkedIn and it also provides `bluesky`, `mastodon`, and `github`.

## Icons

`lucide-static@0.503.0`:
`brain`, `briefcase`, `database`, `flame`, `globe`, `kanban`, `mail`,
`message-square`, `server`, `shield`, `shield-check`, `square-terminal`.

`lucide-static@1.46.0`:
`arrow-up-right-from-square`.

`simple-icons@13.21.0`:
`bluesky`, `github`, `linkedin`, `mastodon`.

## Rendering

`Inline(name, class)`:

- returns `""` for an unknown name, so a bad lookup degrades to no icon rather
  than broken markup;
- drops the fixed `width`/`height` attributes so the stylesheet controls size;
- forces `fill="currentColor"` on brand glyphs so they inherit the surrounding
  link colour, while preserving Lucide's `fill="none"` (outline icons would
  otherwise render as solid shapes);
- adds `class`, `aria-hidden="true"`, and `focusable="false"` to the root
  element.

`template.HTML` is safe here because the content is our own embedded asset,
never user input.

## Licences

Lucide — ISC:

```
ISC License

Copyright (c) 2026 Lucide Icons and Contributors

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
```

Lucide glyphs derived from [Feather](https://feathericons.com/) are MIT:

```
The MIT License (MIT)

Copyright (c) 2013-present Cole Bemis

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

Simple Icons — [CC0-1.0](https://creativecommons.org/publicdomain/zero/1.0/).
The brand marks themselves remain trademarks of their respective owners.
