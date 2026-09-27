# go-vanity

Vanity import paths for Go modules, as a Cloudflare Worker written in Go.

`import "go.calculon.tech/xmorph"` instead of `github.com/ananthb/xmorph`:
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
(`module go.calculon.tech/xmorph`), or `go get` refuses it.

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

## Access

The Worker has no authentication of its own: every path is public, and the
config it serves is the only thing it knows. To restrict it, put Cloudflare
Access in front of the hostname, the way any Worker on a custom domain is
gated. Keep in mind who has to get through: `go get`, `proxy.golang.org` and
pkg.go.dev fetch `?go-get=1` anonymously, so an Access application over the
whole host takes public modules offline for everyone else. For private
modules, gate only their paths (an Access application on
`go.example.com/private*`) and give the people who use them a service token or
a browser session; set `GOPRIVATE` so `go` skips the public proxy for them.

## Deploying

Each release carries the built Worker: `worker.mjs`, `runtime.mjs`,
`wasm_exec.js` and `app.wasm`, one asset each and together in
`go-vanity-worker.tar.gz`.

### Terraform

The Worker is four modules and a JSON binding, so the Cloudflare provider
deploys it with no build step. This is how go.calculon.tech runs:

```hcl
locals {
  go_vanity_version = "v0.1.0"
  go_vanity_modules = {
    "worker.mjs"   = "application/javascript+module"
    "runtime.mjs"  = "application/javascript+module"
    "wasm_exec.js" = "application/javascript+module"
    "app.wasm"     = "application/wasm"
  }
}

data "http" "go_vanity" {
  for_each = local.go_vanity_modules
  url      = "https://github.com/ananthb/go-vanity/releases/download/${local.go_vanity_version}/${each.key}"
}

resource "cloudflare_worker" "go_vanity" {
  account_id = var.account_id
  name       = "go-vanity"
}

resource "cloudflare_worker_version" "go_vanity" {
  account_id         = var.account_id
  worker_id          = cloudflare_worker.go_vanity.id
  compatibility_date = "2026-09-01"
  main_module        = "worker.mjs"
  modules = [for name, type in local.go_vanity_modules : {
    name           = name
    content_type   = type
    content_base64 = data.http.go_vanity[name].response_body_base64
  }]
  bindings = [{
    name = "CONFIG"
    type = "json"
    json = jsonencode({ modules = { foo = "https://github.com/me/foo" } })
  }]
}

resource "cloudflare_workers_deployment" "go_vanity" {
  account_id  = var.account_id
  script_name = cloudflare_worker.go_vanity.name
  strategy    = "percentage"
  versions    = [{ version_id = cloudflare_worker_version.go_vanity.id, percentage = 100 }]
}

resource "cloudflare_workers_custom_domain" "go_vanity" {
  account_id = var.account_id
  hostname   = "go.example.com"
  service    = cloudflare_worker.go_vanity.name
  zone_name  = "example.com"
}
```

### Wrangler

Put your config under `[vars.CONFIG]` in `wrangler.toml` and your hostname in
`routes`, unpack a release into `build/` (or run `./build.sh`), and
`wrangler deploy`.

### Anywhere else

```
CONFIG="$(cat config.json)" PORT=8080 go run go.calculon.tech/go-vanity@latest
```

## Development

```
nix develop
go test ./...
./build.sh && wrangler dev
```

TinyGo's `reflect` cannot run `html/template`, so the pages are written by
hand in `vanity/html.go`, every value through `html.EscapeString`.
