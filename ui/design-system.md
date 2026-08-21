# codepuke design system

Direction C, "Wire Format". This file is the implementation handoff: it is the
contract between the design canvas and the templ/CSS that ships. Someone
implementing from this file should never need to reopen the canvas. The canvas
(claude.ai/code/artifact/0ca00156-3684-49e9-b308-64cf1520989b) remains the
visual reference only; where this file and a board disagree, this file wins,
because the boards contain mockup shortcuts that are called out below.

Scope: tokens, CSS behavior, and markup contracts. No Go and no templ in this
file. Where a contract says "the template emits", that is an instruction to the
templ layer, expressed as the exact HTML it must produce.

One design rule above all: every page works with JavaScript disabled. The full
JS inventory is two small custom elements (section 6). Nothing else.

---

## 1. Tokens

Copy-pasteable. Light is the base scheme; dark arrives only via
`prefers-color-scheme`. There is no class toggle and no toggle UI. The canvas
boards default to dark purely for presentation; that inversion is a canvas
tweak, not a product behavior.

```css
:root {
  /* color, by role */
  --color-bg:         #f2f5ec;  /* page ground */
  --color-surface:    #e8ede0;  /* raised panels: figures, pane headers */
  --color-ink:        #161c12;  /* primary text and structural rules */
  --color-ink-muted:  #5e6b56;  /* metadata, captions, secondary text */
  --color-border:     #c2cdb5;  /* hairlines: row separators, chips, cards */
  --color-accent:     #4a8a00;  /* the one green: links, active states */
  --color-on-accent:  #f2f5ec;  /* text sitting on an accent fill */
  --color-code-bg:    #e9eeda;  /* code blocks, hexdump, terminal, inputs */

  /* syntax, by role (see section 2) */
  --syn-keyword:  #4a8a00;
  --syn-string:   #8a6d1a;
  --syn-function: #1f7a68;
  --syn-type:     #3f6b2a;
  --syn-number:   #a05a20;
  --syn-comment:  #8a9480;

  /* type */
  --font-mono:  ui-monospace, "SF Mono", "Cascadia Mono", Menlo, Consolas,
                "DejaVu Sans Mono", monospace;
  --font-prose: system-ui, -apple-system, "Segoe UI", sans-serif;

  --fs-display: 64px;   /* wordmark, 404, project hero */
  --fs-h1:      40px;   /* article titles */
  --fs-h1-index: 36px;  /* index page titles: archive, projects, tag */
  --fs-h2:      20px;   /* article section headings */
  --fs-title:   16px;   /* record row titles, card names */
  --fs-prose:   16px;   /* long-form body text */
  --fs-ui:      14.5px; /* mono UI base */
  --fs-code:    13.5px; /* code blocks, terminal, hexdump */
  --fs-meta:    12.5px; /* tags, small links, card descriptions */
  --fs-kicker:  12px;   /* uppercase kickers and section labels */
  --fs-micro:   11.5px; /* footers, counts, chips, tiniest labels */

  --lh-ui:    1.7;
  --lh-prose: 1.75;
  --lh-code:  1.6;
  --lh-tight: 1.15;     /* display and h1 sizes */

  --track-label:   0.1em;   /* all uppercase labels */
  --track-display: 0.04em;  /* the wordmark only */

  /* spacing scale; nothing between these values */
  --sp-1: 4px;
  --sp-2: 8px;
  --sp-3: 12px;
  --sp-4: 16px;
  --sp-5: 24px;
  --sp-6: 32px;
  --sp-7: 40px;
  --sp-8: 56px;

  --page-pad: var(--sp-7);  /* horizontal page padding */

  /* radii: sharp corners are a design decision, not an omission */
  --radius: 0;

  /* border weights */
  --bw-hairline: 1px;  /* with --color-border */
  --bw-rule:     2px;  /* with --color-ink: header, footer, section heads */
  --bw-bar:      3px;  /* with --color-accent: left edge of code, callouts */

  /* structural constants */
  --col-offset:  84px;   /* record row hex offset column */
  --col-date:    110px;  /* record row / archive row date column */
  --docs-side:   264px;  /* docs sidebar width */
  --home-side:   300px;  /* home and project page aside width */
  --prose-max:   760px;  /* article measure */
}

@media (prefers-color-scheme: dark) {
  :root {
    --color-bg:         #0b0d0a;
    --color-surface:    #11150e;
    --color-ink:        #d6e0cf;
    --color-ink-muted:  #68765f;
    --color-border:     #242e1e;
    --color-accent:     #a4f43b;
    --color-on-accent:  #0b0d0a;
    --color-code-bg:    #0f130c;

    --syn-keyword:  #a4f43b;
    --syn-string:   #ddc978;
    --syn-function: #7cd4c0;
    --syn-type:     #b7e6a2;
    --syn-number:   #e09a66;
    --syn-comment:  #566249;
  }
}

@media (max-width: 480px) {
  :root {
    --fs-ui:    13.5px;
    --fs-code:  12.5px;
    --fs-prose: 15px;
    --page-pad: 18px;
  }
}
```

`body { background: var(--color-bg); color: var(--color-ink); }` must be
explicit; never rely on a UA default ground.

### Hardcoded values on the boards that must NOT be copied

These are the drift points. Each one exists on a board as a literal value and
must land in the stylesheet as the token named here:

- In-between font sizes. Boards use 10.5px, 11px, 12px, 13px, and 15px in
  labels and captions. Snap every one to the nearest of
  `--fs-micro / --fs-kicker / --fs-meta / --fs-ui`. There is no 10.5px in the
  system.
- Off-scale paddings. Boards use 13px, 14px, 18px, 26px, 28px, 44px, 48px.
  Snap to the `--sp-*` scale. 18px phone padding is the one survivor, kept as
  the 480px value of `--page-pad`.
- The 404 board renders "NOT FOUND" at 72px; the wordmark and the gq hero use
  64px. All three are `--fs-display`. 64px wins.
- The fade on overflow boxes is written on the boards as a gradient from
  `rgba(0,0,0,0)` to a literal background. Production writes it once, as
  `linear-gradient(to right, transparent, var(--color-code-bg))`, inside the
  scroll-box rules (section 4c), so it retints with the scheme automatically.
- The boards fake a scrollbar with `.sbar` and `.sthumb` divs. Those are
  mockup-only. Production styles the real scrollbar:
  `scrollbar-color: var(--color-ink-muted) var(--color-surface);` plus the
  `::-webkit-scrollbar` equivalents, thin, always visible on overflow boxes.
- The admin state dots are text glyphs on the board. Production draws an 8px
  square (radius 0, like everything else) via `::before`: accent fill for
  unsaved, `--color-border` outline for saved.
- Syntax colors are re-declared per board, and only on boards that contain
  code. In production they exist exactly once, in `:root` above.
- Letter-spacing appears on boards as 0.04em through 0.1em ad hoc. Only two
  values exist: `--track-display` for the wordmark, `--track-label` for every
  uppercase label.

---

## 2. Chroma theme

Chroma is configured with classed output, so the Go side emits
`<span class="k">`, `<span class="s1">`, and so on inside
`<pre class="chroma">`. This section is the complete stylesheet for that
output. The first rule is the safety net: any class not mapped below inherits
plain ink and the code background, so an omission renders as unstyled text,
never as broken text.

```css
/* base: catches every token class not listed below */
.chroma {
  background: var(--color-code-bg);
  color: var(--color-ink);
  font: var(--fs-code) / var(--lh-code) var(--font-mono);
}

/* keywords */
.chroma .k, .chroma .kc, .chroma .kd, .chroma .kn, .chroma .kp,
.chroma .kr, .chroma .ow,
.chroma .cp, .chroma .cpf          { color: var(--syn-keyword); }

/* types, classes, tags */
.chroma .kt, .chroma .nc, .chroma .nt, .chroma .ne,
.chroma .vc                        { color: var(--syn-type); }

/* functions, builtins, decorators */
.chroma .nf, .chroma .fm, .chroma .nb, .chroma .bp,
.chroma .nd                        { color: var(--syn-function); }

/* strings and string-like literals */
.chroma .s, .chroma .sa, .chroma .sb, .chroma .sc, .chroma .dl,
.chroma .sd, .chroma .s2, .chroma .se, .chroma .sh, .chroma .si,
.chroma .sx, .chroma .sr, .chroma .s1, .chroma .ss,
.chroma .ld                        { color: var(--syn-string); }

/* numbers and other literals */
.chroma .m, .chroma .mb, .chroma .mf, .chroma .mh, .chroma .mi,
.chroma .il, .chroma .mo, .chroma .l { color: var(--syn-number); }

/* comments */
.chroma .c, .chroma .ch, .chroma .cm, .chroma .c1,
.chroma .cs                        { color: var(--syn-comment); font-style: normal; }

/* names, operators, punctuation: plain ink, stated so intent is visible */
.chroma .n, .chroma .na, .chroma .no, .chroma .ni, .chroma .nl,
.chroma .nn, .chroma .nx, .chroma .py, .chroma .nv, .chroma .vg,
.chroma .vi, .chroma .vm, .chroma .o, .chroma .p,
.chroma .x, .chroma .g             { color: var(--color-ink); }

/* generics (diff output, REPL transcripts, tracebacks) */
.chroma .gi                        { color: var(--syn-keyword); }
.chroma .gd                        { color: var(--color-ink-muted); text-decoration: line-through; }
.chroma .gp                        { color: var(--color-ink-muted); }  /* prompt */
.chroma .go                        { color: var(--color-ink-muted); }  /* output */
.chroma .gh, .chroma .gu           { color: var(--color-ink); font-weight: 700; }
.chroma .ge                        { font-style: italic; }
.chroma .gs                        { font-weight: 700; }
.chroma .gl                        { text-decoration: underline; }
.chroma .gt                        { color: var(--color-ink-muted); }

/* errors: the palette has no red; mark, do not shout */
.chroma .err, .chroma .gr          { text-decoration: underline dotted var(--color-accent); }

/* whitespace and line machinery */
.chroma .w                         { color: inherit; }
.chroma .ln, .chroma .lnt          { color: var(--color-ink-muted); margin-right: var(--sp-3); }
.chroma .hl                        { background: var(--color-surface); display: block; }
```

Notes:

- The class set is chroma's standard short-name set (the Pygments names).
  If a future chroma release adds classes, they fall into the `.chroma` base
  rule until mapped, which is the correct failure mode.
- Comments are never italic. Italic mono is inconsistent across the system
  stack, so the theme relies on color alone.
- `.chroma .hl` (highlighted line) uses the surface color, not the accent;
  the accent is reserved for interactive and selected states.
- Bump `render_version` and reflow when this mapping changes; the classes are
  baked into stored HTML but the colors are not, so pure color changes need
  only a CSS deploy.

---

## 3. Layout primitives

All measurements reference the tokens in section 1. Breakpoints referenced
here are defined once, in section 5.

### 3.1 Page shell

Every public page: header bar, content, footer.

- Header: flex row, space-between, `--bw-rule` bottom border in
  `--color-ink`, `--sp-4 var(--page-pad)` padding. Left: the site name in
  700 mono. Right: nav links (articles, projects, docs, rss) in `--fs-kicker`
  uppercase, gap `--sp-5`, color `--color-ink`.
- Footer: same rule on top, `codepuke.com` left, `EOF` right, both
  `--fs-micro` muted uppercase.
- Content: `var(--page-pad)` horizontal padding.

### 3.2 Prose column

Used by articles and any long-form page.

- One column, `max-width: var(--prose-max)`, centered inside the shell.
- Body text: `--font-prose`, `--fs-prose / --lh-prose`. Everything else on
  the page is mono; running prose is deliberately not.
- Kicker above the title: `--fs-kicker` uppercase muted, the
  `article // date // author` line.
- h1: mono 800, `--fs-h1 / --lh-tight`, uppercase.
- h2: the offset-anchor contract, section 4f.
- Phone: full width, `--page-pad` handles the gutter, nothing else changes.

### 3.3 Sidebar shell

Two variants, one grid each:

- Docs: `grid-template-columns: var(--docs-side) minmax(0, 1fr)`, sidebar on
  the left with a `--bw-rule` right border. Sidebar content: project name row,
  then groups (label + items), items numbered per section 4d.
- Aside pages (home, gq project, projects detail): `minmax(0, 1fr)
  var(--home-side)`, gap `--sp-8`, aside on the right, no divider; section
  heads inside the aside carry their own `--bw-rule` bottom borders.

Degradation at 900px: single column. The docs sidebar is replaced by the
details bar (contract 4d). Aside content stacks below the main column in
source order, and a row of bordered jump links (`topics`, `projects`, in-page
anchors) appears under the page intro so nothing is unreachable. The jump
links are plain `<a href="#id">`; no JS.

### 3.4 Card grid

Projects index.

- `display: grid; grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--sp-4)`.
- Card: `--bw-hairline` border, `--sp-4` padding, column flex with `--sp-2`
  gap. Name (mono 700, `--fs-title`), description (muted, `--fs-meta`,
  flex-grow so footers align), footer row split by a hairline: version chip
  left, `open` link right.
- Hover: border color goes accent. That is the entire hover treatment.
- 900px: two columns. 480px: one column.
- A family with one project renders one card in the same grid; the grid does
  not stretch it full width.

### 3.5 Record row

The article listing row: home, tag pages.

- `display: grid; grid-template-columns: var(--col-offset) var(--col-date)
  minmax(0, 1fr); gap: var(--sp-5); align-items: baseline;` with
  `--sp-4 0` padding and a `--bw-hairline` bottom border. The final row of a
  list closes with `--bw-rule` in ink instead.
- Offset cell: muted mono `--fs-meta`. Derivation: position within the
  rendered list times 0x40, formatted `0x%04x`, starting at `0x0000`. Pure
  presentation, recomputed per page render; on a paginated list each page
  restarts at `0x0000`.
- Date cell: same size, ink.
- Body: title link (`--fs-title` 700, ink, accent on hover, no underline),
  then a meta line: `dan wolf //` muted, then tag chips (hairline border,
  `--fs-meta`, muted; accent text and border on hover).
- 480px: the grid collapses to a stacked block. Line one: offset and date as
  a muted inline pair. Line two: the title. Line three: the tag chips. Same
  borders.

### 3.6 Archive row

Denser than the record row; no offsets, no bylines.

- `grid-template-columns: var(--col-date) minmax(0, 1fr) auto;
  gap: var(--sp-4)`, `--sp-2 0` padding, hairline separators.
- Date muted, title 600 in ink, tags as plain muted mono text (no chip
  borders at this density), right-aligned.
- Year heads: flex space-between, year in 800 mono, `N records` count in
  `--fs-micro` uppercase muted, `--bw-rule` bottom border. Every year head
  carries an `id` (`y2026`) targeted by the `seek` row of bordered year links
  at the top. That seek row is the only navigation that grows with the
  archive.
- 480px: the tags column drops under the title; date column stays.

### 3.7 Family header

Families and projects are database rows, seeded by migration. The header is a
pure projection of three data fields; adding a family is an insert and must
never require a code change, a CSS change, or a template change.

- Data: `name` (short slug shown as the identity), `descriptor` (one clause,
  lowercase, shown after `//`), and a derived project count.
- Index variant (projects page): flex space-between baseline, `--bw-rule`
  bottom border. Left: name in 800 mono `--fs-h2`, then
  `// descriptor` in muted `--fs-meta`. Right: `N projects` in `--fs-micro`
  uppercase muted.
- Sidebar variant (home aside, phone home): one line at `--fs-micro`
  uppercase: name in 700 ink, `// descriptor` muted. No count.
- No numbering anywhere. Display order comes from the query (currently: by
  name); the design encodes nothing about order, so reordering is free.

---

## 4. Markup contracts

Exact DOM the templates emit. Class names are load-bearing.

### 4a. `<code-tabs>`

Server output for `:::examples <topic>`; every variant ships pre-highlighted:

```html
<code-tabs data-topic="encode-struct">
  <section data-lang="go">
    <h4 class="ct-label">Go</h4>
    <scroll-box><pre class="chroma"><!-- classed spans --></pre></scroll-box>
  </section>
  <section data-lang="typescript">
    <h4 class="ct-label">TypeScript</h4>
    <scroll-box><pre class="chroma">…</pre></scroll-box>
  </section>
  <section data-lang="python">…</section>
  <section data-lang="csharp">…</section>
</code-tabs>
```

`data-lang` values are chroma lexer names; the `ct-label` text is the human
name. Section order is the site-wide language order (go, typescript, python,
csharp).

**No-JS default state (also the pre-upgrade state):** exactly what is above.
Sections stack inside one hairline-bordered box; each `ct-label` renders as a
`--fs-micro` uppercase accent bar row with hairline separators. This state is
the styled base CSS, not a fallback hack; it must look finished.

**On upgrade** the element:

1. Prepends `<nav class="ct-tabs" role="tablist">` containing one
   `<button role="tab">` per section, labeled from that section's `ct-label`.
2. Hides the `ct-label` rows (`code-tabs[data-active] .ct-label
   { display: none; }`) and toggles the `hidden` attribute on every
   non-active section.
3. Reads the initial language from `localStorage["codepuke:lang"]`, falling
   back to the first section; reflects it as `data-active="<lang>"` on the
   host.
4. On tab click: writes localStorage and dispatches a
   `codepuke:lang` CustomEvent (detail: the lang) on `document`. Every
   code-tabs instance listens and follows, so all blocks on a page stay in
   sync.

Tab styling: `--fs-micro` uppercase mono chips, hairline border, muted;
active tab gets accent fill with `--color-on-accent` text and 700 weight.

**Below 480px** the tab row is
`display: grid; grid-template-columns: repeat(2, minmax(0, 1fr))`, buttons
full-width and centered, `--sp-3 --sp-4` padding. Four languages form a two by
two grid. This is a plain media query on `.ct-tabs`; the element does no
measuring.

### 4b. Hexdump strip

The template renders the SAME bytes twice, sixteen per row and eight per row,
and CSS shows exactly one. This doubles the strip's HTML weight, which is
noise, and keeps the reflow entirely server-plus-CSS:

```html
<figure class="hexdump">
  <pre class="hexdump-16">00000000  1f ff 81 03 01 01 05 50  6f 69 6e 74 01 ff 82 00  |.......Point....|
00000010  07 ff 82 01 06 01 08 00                           |........|</pre>
  <pre class="hexdump-8">00000000  1f ff 81 03 01 01 05 50  |.......P|
00000008  6f 69 6e 74 01 ff 82 00  |oint....|
00000010  07 ff 82 01 06 01 08 00  |........|</pre>
  <figcaption>gob.Encode(Point{X: 3, Y: 4}), all of it.</figcaption>
</figure>
```

```css
.hexdump pre   { /* code-bg fill, hairline border, muted text, pre whitespace */ }
.hexdump-8     { display: none; }
@media (max-width: 480px) {
  .hexdump-16  { display: none; }
  .hexdump-8   { display: block; }
}
```

Byte highlights (`<span class="hl">`) use `--color-accent` and must be
applied to both variants at render time. `display: none` removes the hidden
variant from the accessibility tree; no aria attributes needed.

### 4c. `<scroll-box>` (the overflow contract)

**The decision, explicitly:** the rule "the scroll affordance appears only
when a line actually overflows" is implemented as a small custom element that
measures. It is not CSS-only and not always-on chrome. The element compares
`scrollWidth` to `clientWidth` on its scrolling child, on connect and via one
`ResizeObserver`, and toggles `data-overflow` on itself. CSS draws the chip
and fade only under `scroll-box[data-overflow]`. About twenty lines of JS,
shared by every instance.

**No-JS fallback, defined:** the base CSS (no attribute) already gives the
child `overflow-x: auto` and an always-visible styled scrollbar. Without JS,
long lines scroll natively and the scrollbar is the affordance; the chip and
fade simply never appear. Nothing clips silently, nothing widens the page.
The constraint holds: the page works, JS only adds the louder signpost.

The templates emit `<scroll-box>` unconditionally around every code block and
every figure; the element (or the fallback) decides what shows:

```html
<scroll-box><pre class="chroma">…</pre></scroll-box>

<scroll-box>
  <figure class="fig">
    <svg width="660" height="180" viewBox="0 0 660 180">…</svg>
    <figcaption>fig 01 // …</figcaption>
  </figure>
</scroll-box>
```

```css
scroll-box { display: block; position: relative; }
scroll-box > * { overflow-x: auto; max-width: 100%;
  scrollbar-color: var(--color-ink-muted) var(--color-surface);
  scrollbar-width: thin; }
scroll-box[data-overflow]::before {   /* the chip */
  content: "\2194 scrolls"; position: absolute; top: -9px; right: 10px;
  background: var(--color-accent); color: var(--color-on-accent);
  font: 700 var(--fs-micro) var(--font-mono);
  letter-spacing: var(--track-label); text-transform: uppercase;
  padding: 2px 7px; }
scroll-box[data-overflow]::after {    /* the fade */
  content: ""; position: absolute; top: 1px; right: 1px; bottom: 1px;
  width: 44px; pointer-events: none;
  background: linear-gradient(to right, transparent, var(--color-code-bg)); }
```

Figures keep their natural size and scroll; never shrink an SVG to fit a
phone, the labels become unreadable.

### 4d. Docs navigation

One templ partial for the nav, rendered twice per docs page; CSS shows one:

```html
<!-- desktop: left column of the sidebar shell -->
<nav class="docs-nav" aria-label="documentation">
  <div class="docs-project">gobspect <span class="ver">v1.4</span></div>
  <div class="nav-group">// start here</div>
  <a class="nav-item" href="…"><span>00</span>overview</a>
  <a class="nav-item" href="…"><span>01</span>installation</a>
  <div class="nav-group">// guides</div>
  <a class="nav-item on" href="…" aria-current="page"><span>02</span>reading streams</a>
  <!-- … -->
</nav>

<!-- phone: first element of the content column -->
<details class="docs-toc">
  <summary>contents // gobspect <span class="ver">v1.4</span></summary>
  <!-- the same nav partial, verbatim -->
</details>
```

```css
.docs-toc { display: none; }
@media (max-width: 900px) {
  .docs-nav-column { display: none; }  /* the sidebar grid column */
  .docs-toc { display: block; }        /* native disclosure, zero JS */
}
```

- The summary row: hairline border, `--bw-bar` accent left edge, `--sp-4`
  padding, 700 mono. The default disclosure marker is kept.
- Item numbers (`00`, `01`, …) are the zero-padded decimal position of the
  page in the project's nav order, which is data (doc ordering from the sync
  manifest), not markup invention. Active item: accent, 700,
  `aria-current="page"`.
- The duplication is server-side and free; only one copy is ever displayed,
  so screen readers see a single nav.

### 4e. Footnotes

Goldmark's footnote extension emits this shape; the site styles it as-is and
adds nothing at render time:

```html
<p>…assigned type ID 65.<sup id="fnref:1"><a href="#fn:1"
   class="footnote-ref" role="doc-noteref">1</a></sup></p>

<div class="footnotes" role="doc-endnotes">
  <hr>
  <ol>
    <li id="fn:1">
      <p>Type IDs 0 through 64 are reserved… <a href="#fnref:1"
         class="footnote-backref" role="doc-backlink">&#x21a9;&#xfe0e;</a></p>
    </li>
  </ol>
</div>
```

- `.footnote-ref` gets square brackets via CSS
  (`::before { content: "[" } ::after { content: "]" }`), mono
  `--fs-micro`, accent. The brackets are presentation, never typed into
  content.
- `.footnotes hr` is restyled to the `--bw-rule` ink rule; the list is
  `--fs-micro` mono muted with `--sp-2` between items.
- The return arrow is goldmark's own `.footnote-backref`; every footnote gets
  one automatically. (A canvas board shipped without one once; that was a
  mockup bug, and the extension makes it impossible here.)

### 4f. Heading anchors, `0x01` style

Applied at publish time by the render pipeline (a goldmark AST transform),
h2 only:

```html
<h2 id="the-type-descriptor-comes-first">
  <a class="offset-anchor" href="#the-type-descriptor-comes-first">0x01</a>
  The type descriptor comes first
</h2>
```

- **id derivation:** slug of the heading text (lowercase, hyphenated,
  deduplicated with `-2` suffixes on collision). Text-derived so deep links
  survive reordering.
- **Label derivation:** the 1-based ordinal of the h2 among the document's
  h2 elements, formatted `0x%02x`. `0x01`, `0x02`, … Reordering headings
  renumbers labels at the next render; the ids do not move, so old links keep
  working even when the number shown has changed.
- Styling: h2 is a baseline flex row, gap `--sp-4`, mono 700 `--fs-h2`
  uppercase; the anchor is a bordered chip, `--fs-code` accent on a hairline
  border, accent border on hover, no underline ever.
- h3 and deeper: id only (same slug rule), no visible chip. If an article
  ever needs visible h3 anchors, that is a design addition, not a renderer
  default.
- These are unrelated to the list-row offsets (3.5), which are positional and
  page-local.

---

## 5. Breakpoints

Two. Exactly two. The 390px seen on the canvas is the test device width the
phone boards were drawn at, not a breakpoint.

**`max-width: 900px`** (layout collapse)

- Sidebar shells become single column; aside content stacks below main.
- Docs sidebar column hides; the `details` contents bar shows (4d).
- Jump-link rows appear on pages whose aside moved below the fold (3.3).
- Card grid: three columns to two.
- Admin editor panes stack, markdown above preview.

**`max-width: 480px`** (density and reflow)

- Token overrides: `--fs-ui`, `--fs-code`, `--fs-prose`, `--page-pad`
  (section 1).
- Hexdump swaps to the 8-byte variant (4b).
- `code-tabs` tab row becomes the two by two grid (4a).
- Record rows stack (3.5); archive rows drop tags under the title (3.6).
- Card grid: one column.

Nothing else may introduce a media query without adding it to this section.

---

## 6. JavaScript inventory

The complete list. Both are custom elements that enhance markup already
working without them, per CLAUDE.md.

1. **`<code-tabs>`** (4a): builds the tab row, toggles `hidden` on sections,
   persists to `localStorage["codepuke:lang"]`, syncs instances through the
   `codepuke:lang` document event. Roughly forty lines. Without it: all
   variants visible with labels.
2. **`<scroll-box>`** (4c): measures `scrollWidth` against `clientWidth`,
   toggles `data-overflow`, re-measures via one ResizeObserver. Roughly
   twenty lines. Without it: native horizontal scroll with a styled,
   always-visible scrollbar.

The docs contents bar, the theme switch, the hexdump reflow, jump links, tag
filters, and pagination are all zero-JS: native elements, media queries, and
real URLs.

---

## 7. Webfonts

**None. Zero bytes.** Every surface uses the two system stacks in section 1.
The identity comes from weight, case, color, and structure, not from glyph
shapes, so the mono stack differing across platforms (SF Mono on macOS,
Cascadia or Consolas on Windows, DejaVu on Linux) changes texture slightly
and identity not at all.

The only candidate worth naming: a single mono webfont would pin the wordmark
and hexdump rendering across platforms. Cost: roughly 25 to 45 KB woff2 per
weight, and the design uses two weights (400, 700/800), so 50 to 90 KB plus a
FOUT strategy. Recommendation: skip it. Revisit only if the wordmark becomes
a logo used off-site. (Newsreader existed only in the retired direction B and
is not part of this system.)

---

## 8. Data-driven chrome

Repeated here because it constrains the CSS: families and projects are
database rows, seeded by migration. The family header (3.7), the projects
index, the home aside, and the docs project switcher must all render from
query results with no per-family styling, no per-family templates, and no
ordering encoded in the presentation. Adding a family is an insert. If a CSS
rule ever needs to know a family's name, the rule is wrong.
