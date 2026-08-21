# Syncing content from sibling repos

`cmd/sync` pulls two things out of the repos listed in `sources.json`: code
snippets by region marker, and docs pages from a `docs/` directory. This file
is the contract a sibling repo follows. It is safe to hand to a Claude Code
session running inside any sibling repo.

Nothing here runs in the sibling repo. Authors only add markers and docs
files; the codepuke repo runs `go run ./cmd/sync` and commits the output.

## Snippet markers

Wrap a region of a real, compiling source file:

```go
// snippet:start encode-struct
enc := gob.NewEncoder(&buf)
enc.Encode(Point{X: 3, Y: 4})
// snippet:end
```

```python
# snippet:start encode-struct
enc = Encoder(buf)
enc.encode(Point(x=3, y=4))
# snippet:end
```

Rules:

- Comment token is the language's line comment: `//` for Go, TypeScript, and
  C#; `#` for Python. The marker must be the only thing on its line.
- Topic ids are `[a-z0-9][a-z0-9-]*`. A topic appears at most once per file
  and at most once per language across all repos.
- `snippet:end` may optionally repeat the topic. Regions may nest; a bare
  end closes the most recently opened region.
- Marker lines never appear in the extracted snippet. The region is dedented
  by its common leading whitespace and trimmed of surrounding blank lines,
  so marking a region inside a function body extracts clean top-level code.
- Anything malformed (typo verbs, unclosed regions, bad topic ids) fails the
  sync loudly. There is no silent skipping.

The point of markers is that snippets come from code that compiles and runs
in that repo's test suite, never from markdown. Prefer marking regions in
example or test files that CI executes.

### Topic coordination

A topic is one concept shown in up to four languages. The site renders every
variant of a topic in one tabbed block, so the variants should do the same
thing with the same data. Before adding a new topic, check
`content/manifest.json` here for the topic list, and use the same id in each
repo:

| repo | lang | file extension scanned |
| --- | --- | --- |
| gobspect, gobspect-mcp | go | .go |
| gobts | typescript | .ts |
| pygob | python | .py |
| gobdotnet | csharp | .cs |

The Go variant of a wire-format topic is standard library `encoding/gob`
usage, not gobspect API calls; gobspect is a decoder. Those stdlib snippets
live in the gobspect repo, in compiled example or test files, because it
already exercises the stdlib encoder to produce streams to introspect.
gobspect's own API topics are additional, gobspect-only topics and render as
single-tab blocks. Shell usage (gq) is never a snippet topic; it goes in
ordinary fenced code inside docs pages.

## Docs pages

Markdown files in the repo's `docs/` directory (`cmd/gq/docs/` for gq).

- Nav order is filename order. Number the files to control it:
  `00-overview.md`, `01-installation.md`. The numeric prefix is stripped
  from the slug, so `00-overview.md` becomes `/docs/<project>/overview`.
- The page title comes from front matter when present, else the first `# `
  heading, else the slug:

  ```markdown
  ---
  title: Reading Streams
  ---
  ```

  Front matter is flat `key: value` pairs only, and it is stripped from the
  published body.
- Docs may use GFM, footnotes, fenced code, and `:::examples <topic>` blocks
  (which expand to the tabbed snippet viewer for that topic).

## Running a sync

From the codepuke repo:

```sh
go1.27rc2 run ./cmd/sync    # reads sources.json, rebuilds content/
go1.27rc2 test ./...        # content feeds the render pipeline tests
```

Then commit the regenerated `content/` tree here. Sync reads each repo at
the ref pinned in `sources.json` (currently `main` for all), via git, so
uncommitted changes in a sibling repo are invisible to it.
