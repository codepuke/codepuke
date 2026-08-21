# codepuke

The site behind codepuke.com. A Go server that publishes dated, categorized
markdown articles plus documentation and multi-language code examples for the
`codepuke` gob projects.

## Non-negotiables

- **No frontend framework. No HTMX. No SPA.** Server-rendered HTML is the
  product. Interactivity is added only as small vanilla-JS custom elements that
  progressively enhance already-working markup. If a feature cannot degrade to
  something readable with JS disabled, redesign the feature.
- **No sqlc, no ORM.** `pgx/v5` directly, with its `RowToStructByName` /
  `CollectRows` helpers for scanning. Hand-written SQL lives next to the code
  that uses it.
- **Render at write time, not read time.** Markdown is converted to HTML when
  content is published or synced, and the HTML is stored. Request handlers do
  not run goldmark.

## Stack

| Concern | Choice |
| --- | --- |
| Language | Go 1.27 (`go.mod` go directive set to 1.27) |
| Templates | [templ](https://templ.guide) |
| Routing | stdlib `net/http` with Go 1.22 pattern syntax (`GET /articles/{slug}`) |
| Database | PostgreSQL via `github.com/jackc/pgx/v5` (+ `pgxpool`) |
| Migrations | `github.com/pressly/goose/v3`, embedded FS, `up` on boot |
| Markdown | `github.com/yuin/goldmark` (GFM, footnotes, attributes, heading anchors) |
| Highlighting | `github.com/alecthomas/chroma/v2` via goldmark-highlighting, classed output |
| Diagrams | mermaid-cli sidecar called at publish time, SVG inlined and cached by source hash |
| Sanitizing | `github.com/microcosm-cc/bluemonday` on rendered HTML |
| Auth | Authentik OIDC via `github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2`, PKCE |
| Sessions | Encrypted cookie (chacha20poly1305), no session table |
| Logging | `log/slog` |
| Config | Environment variables, parsed once into a struct at startup |
| Tests | `stretchr/testify` (`require` for preconditions, `assert` for checks) |

Until Go 1.27 is GA, the toolchain comes from
`go install golang.org/dl/go1.27rc2@latest && go1.27rc2 download`.

## Design

`ui/design-system.md` is the implementation source of truth for all markup and
CSS. Where it and the design canvas disagree, the file wins. Do not invent
markup for a surface it already specifies.

## Data, not code

Families, projects, and categories are database rows seeded by migration.
Adding a family or a project must never require a code change or a redeploy of
anything but data.

## Go conventions

Use current-generation Go. Specifically:

- `new(expression)` instead of hand-rolled `ptr[T]` / `strPtr` helpers.
- `min` / `max` builtins; `cmp.Or` for fallback chains.
- The `slices` and `maps` packages instead of manual loops.
- Range over int and range-over-func. Write real iterators (`iter.Seq`,
  `iter.Seq2`) when the element count is unbounded or the sequence gets reused.
- `errors.AsType[T](err)` over `errors.As` with a target pointer. `errors.Is`
  for sentinels. Wrap with `%w` when the cause should stay inspectable.
  `errors.Join` for accumulating.
- `sync.OnceFunc` / `sync.OnceValue` instead of `sync.Once` plus a var.
- `io/fs.FS` for read-only file access. Reach for `spf13/afero` only where
  writes genuinely need faking; skip the abstraction when it is trivial.
- Run `go fix ./...`, `go mod tidy`, and `gofmt` before declaring a task done.

### Tests

- Table-driven with `t.Run` subtests, grouped into valid / invalid / edge cases.
- `t.Parallel()` by default unless there is shared mutable state.
- `t.Cleanup` over defer-in-helper.
- Fuzz tests for pure functions taking primitives or `[]byte` (slug generation,
  snippet marker parsing, front matter parsing).
- `testing/synctest` for anything time- or concurrency-sensitive.
- Database tests run against a real Postgres via testcontainers, not mocks.

## Repository layout

```
codepuke/
  cmd/
    codepuke/        # the server
    sync/            # pulls docs + snippets from sibling repos into content/
  internal/
    content/         # goldmark pipeline, block extensions, mermaid client
    store/           # pgx queries and row structs
    auth/            # OIDC + cookie sessions
    web/             # handlers, middleware, routing
  ui/                # .templ files, hand-written CSS, custom elements
  content/           # synced markdown + snippets, committed, go:embed'ed
  migrations/        # goose SQL
```

Sibling repos are cloned parallel to this one and are read-only inputs to
`cmd/sync`:

- `../gobspect` (gob introspection library, `gq` CLI)
- `../gobspect-mcp` (MCP server)
- `../gobts` (TypeScript port of encoding/gob)
- `../pygob` (Python port)
- `../gobdotnet` (.NET port)

Never edit files outside this repository.

## Content model

One `documents` table serves everything, discriminated by `kind`:

- `article` - authored in the web editor, source of truth is the database.
- `doc` - authored in a sibling repo under `docs/`, synced in, read-only in the
  web UI, upserted by `codepuke sync-db` on deploy.

Columns include `slug`, `title`, `author`, `body_md`, `body_html`,
`render_version`, `published_at`, `updated_at`, and a nullable `version` for
versioned docs. Categories are a join table. There are no comments and no
comment schema.

Bumping the render pipeline means bumping `render_version` and running
`codepuke reflow`, which re-renders every row below the current version.

### Custom markdown blocks

Extensions register into a single map in `internal/content/blocks.go`. The
first two:

- `:::examples <topic-id>` expands to a `<code-tabs>` element containing every
  language variant of that topic, each pre-highlighted server-side. All
  variants ship in the HTML; CSS hides the inactive ones; ~40 lines of JS
  handles switching and persists the choice to `localStorage`, broadcasting a
  `codepuke:lang` event so every block on the page stays in sync.
- ```` ```mermaid ```` is rendered to SVG at publish time and inlined.

Snippets are never hand-written into markdown. `cmd/sync` extracts them from
the sibling repos by region marker:

```go
// snippet:start encode-struct
...
// snippet:end
```

and writes `content/examples/<topic>/<lang>.<ext>`, which is committed. CI does
not need the sibling repos checked out.

## Auth

Everything under `/admin` sits behind one middleware. Nothing else in the
codebase knows auth exists. Authorization is by Authentik group claim, not by
subject, so authors can be added without a deploy.

## Writing style

- **No em-dashes in human-facing prose.** That covers this file, READMEs,
  commit messages, code comments, doc pages, and UI copy. Em-dashes inside
  code, config, regex, and string literals are fine.
- Commit messages: imperative mood, one-line subject, body only when the
  reasoning is not obvious from the diff.

## Task runner

None for now. Plain `go` commands. Only introduce a Taskfile if the number of
steps genuinely stops fitting in a person's head, and never a Makefile.
