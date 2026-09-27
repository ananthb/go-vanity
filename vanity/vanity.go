// Package vanity serves vanity import paths for Go modules.
//
// `go get example.com/foo/bar` asks https://example.com/foo/bar?go-get=1 for a
// go-import meta tag naming the repository that holds it. Handler answers that
// from a Config, and shows people a page for the module with its install
// command, source and docs.
package vanity

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// Config is the whole configuration.
type Config struct {
	// Module path under the host → repository.
	Modules map[string]Module `json:"modules"`
	// What to show a browser on a module path: "page" (default), or a 302 to the
	// "repo" or to "pkgsite".
	Browser string `json:"browser,omitempty"`
	// Heading of the index page; defaults to the host.
	Title string `json:"title,omitempty"`
	// Where / redirects to. Empty serves the index page.
	IndexRedirect string `json:"index_redirect,omitempty"`
	// The module host, when it differs from the request's.
	Host string `json:"host,omitempty"`
}

// Module is one entry in Config.Modules. It unmarshals from either the
// repository URL alone or an object.
type Module struct {
	Repo        string `json:"repo"`
	VCS         string `json:"vcs,omitempty"`    // default "git"
	Branch      string `json:"branch,omitempty"` // default "main", for source links
	Description string `json:"description,omitempty"`
	// Source picks the go-source templates: "github", "gitlab", "gitea",
	// "forgejo", or "none". Inferred for github.com, gitlab.com, codeberg.org.
	Source string `json:"source,omitempty"`
}

func (m *Module) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		*m = Module{Repo: s}
		return nil
	}
	type plain Module
	return json.Unmarshal(b, (*plain)(m))
}

var sourceTemplates = map[string]func(repo, ref string) [2]string{
	"github": func(r, ref string) [2]string {
		return [2]string{r + "/tree/" + ref + "{/dir}", r + "/blob/" + ref + "{/dir}/{file}#L{line}"}
	},
	"gitlab": func(r, ref string) [2]string {
		return [2]string{r + "/-/tree/" + ref + "{/dir}", r + "/-/blob/" + ref + "{/dir}/{file}#L{line}"}
	},
	// Gitea and its fork Forgejo, Codeberg included.
	"gitea": func(r, ref string) [2]string {
		return [2]string{r + "/src/branch/" + ref + "{/dir}", r + "/src/branch/" + ref + "{/dir}/{file}#L{line}"}
	},
}

var inferredSource = map[string]string{
	"github.com":   "github",
	"gitlab.com":   "gitlab",
	"codeberg.org": "gitea",
}

type module struct {
	Prefix, Repo, VCS, Description string
	Source                         *[2]string
}

type handler struct {
	cfg     Config
	modules []module // longest prefix first
}

// Handler validates cfg and serves it.
func Handler(cfg Config) (http.Handler, error) {
	switch cfg.Browser {
	case "", "page", "repo", "pkgsite":
	default:
		return nil, fmt.Errorf("browser: %q is not page, repo or pkgsite", cfg.Browser)
	}
	h := &handler{cfg: cfg}
	for prefix, m := range cfg.Modules {
		if m.Repo == "" {
			return nil, fmt.Errorf("module %q: no repo", prefix)
		}
		repo := strings.TrimSuffix(strings.TrimRight(m.Repo, "/"), ".git")
		u, err := url.Parse(repo)
		if err != nil || u.Host == "" {
			return nil, fmt.Errorf("module %q: repo %q is not a URL", prefix, m.Repo)
		}
		mod := module{Prefix: strings.Trim(prefix, "/"), Repo: repo, VCS: m.VCS, Description: m.Description}
		if mod.VCS == "" {
			mod.VCS = "git"
		}
		kind := m.Source
		if kind == "" {
			kind = inferredSource[u.Host]
		}
		if kind == "forgejo" {
			kind = "gitea"
		}
		if kind != "" && kind != "none" {
			tmpl, ok := sourceTemplates[kind]
			if !ok {
				return nil, fmt.Errorf("module %q: unknown source %q", prefix, m.Source)
			}
			branch := m.Branch
			if branch == "" {
				branch = "main"
			}
			s := tmpl(repo, branch)
			mod.Source = &s
		}
		h.modules = append(h.modules, mod)
	}
	sort.Slice(h.modules, func(i, j int) bool { return len(h.modules[i].Prefix) > len(h.modules[j].Prefix) })
	return h, nil
}

// match returns the module whose prefix is the path, or a whole-segment
// prefix of it.
func (h *handler) match(path string) *module {
	p := strings.Trim(path, "/")
	for i, m := range h.modules {
		if p == m.Prefix || strings.HasPrefix(p, m.Prefix+"/") {
			return &h.modules[i]
		}
	}
	return nil
}

type page struct {
	Host, Title, Path string
	Module            *module
	Modules           []module
}

func (h *handler) render(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	io.WriteString(w, body)
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	host := h.cfg.Host
	if host == "" {
		host = r.Host
	}
	title := h.cfg.Title
	if title == "" {
		title = host
	}
	w.Header().Set("Cache-Control", "public, max-age=300")

	if r.URL.Path == "/" {
		if h.cfg.IndexRedirect != "" {
			http.Redirect(w, r, h.cfg.IndexRedirect, http.StatusFound)
			return
		}
		mods := append([]module(nil), h.modules...)
		sort.Slice(mods, func(i, j int) bool { return mods[i].Prefix < mods[j].Prefix })
		h.render(w, http.StatusOK, indexHTML(page{Host: host, Title: title, Modules: mods}))
		return
	}

	m := h.match(r.URL.Path)
	if m == nil {
		h.render(w, http.StatusNotFound, notFoundHTML(page{Host: host, Title: title, Path: r.URL.Path}))
		return
	}
	path := host + strings.TrimRight(r.URL.Path, "/")

	// The module page carries the meta tags too, so it answers go-get=1 as
	// well as a browser.
	if r.URL.Query().Get("go-get") == "1" || h.cfg.Browser == "" || h.cfg.Browser == "page" {
		h.render(w, http.StatusOK, moduleHTML(page{Host: host, Title: title, Path: path, Module: m}))
		return
	}
	target := m.Repo
	if h.cfg.Browser == "pkgsite" {
		target = "https://pkg.go.dev/" + path
	}
	http.Redirect(w, r, target, http.StatusFound)
}
