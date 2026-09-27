# go-vanity

Vanity import paths for Go modules, as a Cloudflare Worker written in Go.

`import "go.example.com/foo"` instead of `github.com/me/foo`:
the import names a domain you own, and the domain names the repository. Moving
the repository to another forge is a one-line config change, and nothing that
imports it has to change. The idea, and an nginx version of it, is in
[Don't couple your Go code to GitHub](https://iain.rocks/blog/dont-couple-your-go-code-to-github).

It is a `net/http` handler compiled with TinyGo and served through
[workers-go](https://github.com/syumai/workers-go): about 0.5 MB gzipped. The
same code runs as a plain HTTP server anywhere else.

## What it serves

| request | response |
|---|---|
| `/<module>[/pkg...]?go-get=1` | `go-import` and `go-source` meta tags for `<host>/<module>` |
| `/<module>[/pkg...]` | a page with the `go get` line, source and docs links, and the same tags; or a 302 with `browser` |
| `/` | an index of the modules, or a 302 to `index_redirect` |
| anything else | 404 |

Prefixes match whole path segments, longest first, so `tools` and `tools/x`
can be different repositories. Major-version subdirectories (`/v2`) need no
entry: the go-import prefix is the repository root and `go` reads the rest
from `go.mod`.

A module's own `go.mod` has to declare the vanity path
(`module go.example.com/foo`), or `go get` refuses it.

## Configuration

One JSON document, the `CONFIG` binding:

```json
{
  "title": "go.example.com",
  "modules": {
    "foo": "https://github.com/me/foo",
    "tools/bar": { "repo": "https://codeberg.org/me/bar", "branch": "trunk", "description": "Bar." },
    "private": { "repo": "https://git.example.com/me/private", "source": "none" }
  }
}
```

| top level | default | |
|---|---|---|
| `modules` | required | module path under the host → repo URL, or an object below |
| `browser` | `page` | `page`, or a 302 to the `repo` or to `pkgsite` |
| `title` | the host | index page heading |
| `index_redirect` | | a URL for `/` to redirect to instead |
| `host` | the request's | the module host, if it differs |

| module | default | |
|---|---|---|
| `repo` | required | clone URL, `.git` optional |
| `vcs` | `git` | `git`, `hg`, `svn`, `bzr`, `fossil` |
| `branch` | `main` | used in the source-link templates |
| `description` | | shown on the index and module pages |
| `source` | inferred for github.com, gitlab.com, codeberg.org | `github`, `gitlab`, `gitea`, `forgejo`, or `none` for no `go-source` tag |

## Deploying

Each release carries the built Worker: `worker.mjs`, `runtime.mjs`,
`wasm_exec.js` and `app.wasm`, one asset each and together in
`go-vanity-worker.tar.gz`.

### Cloudflare

Put your config under `[vars.CONFIG]` in `wrangler.toml` and your hostname in
`routes`, unpack a release into `build/` (or run `./build.sh`), and
`wrangler deploy`.

### Anywhere else

```
CONFIG="$(cat config.json)" PORT=8080 go run .
```

## Development

```
nix develop
go test ./...
./build.sh && wrangler dev
```

TinyGo's `reflect` cannot run `html/template`, so the pages are written by
hand in `vanity/html.go`, every value through `html.EscapeString`.
