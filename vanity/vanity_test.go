package vanity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

const cfgJSON = `{
	"modules": {
		"chonker": "https://github.com/ananthb/chonker",
		"tools/x": {"repo": "https://calculon.tech/calculon-tech/x.git", "source": "forgejo", "branch": "trunk"},
		"tools": "https://gitlab.com/a/tools/",
		"private": {"repo": "https://git.example.com/p", "source": "none"}
	}
}`

func serve(t *testing.T, cfg Config, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	if cfg.Modules == nil {
		if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
			t.Fatal(err)
		}
	}
	h, err := Handler(cfg)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(method, "https://go.calculon.tech"+target, nil))
	return w
}

func meta(w *httptest.ResponseRecorder, name string) string {
	m := regexp.MustCompile(`<meta name="` + name + `" content="([^"]*)">`).FindStringSubmatch(w.Body.String())
	if m == nil {
		return ""
	}
	return m[1]
}

func TestGoImport(t *testing.T) {
	for _, path := range []string{"/chonker", "/chonker/", "/chonker/internal/x", "/chonker/v2"} {
		w := serve(t, Config{}, "GET", path+"?go-get=1")
		if w.Code != 200 {
			t.Fatalf("%s: %d", path, w.Code)
		}
		if got, want := meta(w, "go-import"), "go.calculon.tech/chonker git https://github.com/ananthb/chonker"; got != want {
			t.Errorf("%s: go-import %q, want %q", path, got, want)
		}
	}
}

func TestPrefixes(t *testing.T) {
	for path, want := range map[string]string{
		"/tools/x/y": "go.calculon.tech/tools/x git https://calculon.tech/calculon-tech/x",
		"/tools/xy":  "go.calculon.tech/tools git https://gitlab.com/a/tools",
		"/chonkers":  "",
	} {
		if got := meta(serve(t, Config{}, "GET", path+"?go-get=1"), "go-import"); got != want {
			t.Errorf("%s: %q, want %q", path, got, want)
		}
	}
}

func TestGoSource(t *testing.T) {
	for path, want := range map[string]string{
		"/chonker": "go.calculon.tech/chonker https://github.com/ananthb/chonker https://github.com/ananthb/chonker/tree/main{/dir} https://github.com/ananthb/chonker/blob/main{/dir}/{file}#L{line}",
		"/tools/x": "go.calculon.tech/tools/x https://calculon.tech/calculon-tech/x https://calculon.tech/calculon-tech/x/src/branch/trunk{/dir} https://calculon.tech/calculon-tech/x/src/branch/trunk{/dir}/{file}#L{line}",
		"/tools":   "go.calculon.tech/tools https://gitlab.com/a/tools https://gitlab.com/a/tools/-/tree/main{/dir} https://gitlab.com/a/tools/-/blob/main{/dir}/{file}#L{line}",
		"/private": "",
	} {
		if got := meta(serve(t, Config{}, "GET", path+"?go-get=1"), "go-source"); got != want {
			t.Errorf("%s: %q, want %q", path, got, want)
		}
	}
}

func TestBrowser(t *testing.T) {
	w := serve(t, Config{}, "GET", "/chonker/internal")
	if w.Code != 200 || meta(w, "go-import") == "" || !strings.Contains(w.Body.String(), "go get go.calculon.tech/chonker/internal") {
		t.Errorf("page: %d %s", w.Code, w.Body)
	}
	w = serve(t, Config{Browser: "repo"}, "GET", "/chonker/internal")
	if w.Code != http.StatusFound || w.Header().Get("Location") != "https://github.com/ananthb/chonker" {
		t.Errorf("repo: %d %q", w.Code, w.Header().Get("Location"))
	}
	w = serve(t, Config{Browser: "pkgsite"}, "GET", "/chonker/internal/")
	if got := w.Header().Get("Location"); got != "https://pkg.go.dev/go.calculon.tech/chonker/internal" {
		t.Errorf("pkgsite: %q", got)
	}
	// go-get=1 always gets the tags, whatever browsers get.
	if got := meta(serve(t, Config{Browser: "repo"}, "GET", "/chonker?go-get=1"), "go-import"); got == "" {
		t.Error("repo mode: no go-import for go-get=1")
	}
}

func TestNotFound(t *testing.T) {
	for _, path := range []string{"/nope?go-get=1", "/nope"} {
		if w := serve(t, Config{}, "GET", path); w.Code != 404 {
			t.Errorf("%s: %d", path, w.Code)
		}
	}
}

func TestIndex(t *testing.T) {
	if w := serve(t, Config{}, "GET", "/"); !regexp.MustCompile(`go\.calculon\.tech/chonker`).MatchString(w.Body.String()) {
		t.Errorf("index: %s", w.Body)
	}
	if w := serve(t, Config{IndexRedirect: "https://calculon.tech/"}, "GET", "/"); w.Header().Get("Location") != "https://calculon.tech/" {
		t.Errorf("index redirect: %q", w.Header().Get("Location"))
	}
}

func TestHostOverride(t *testing.T) {
	w := serve(t, Config{Host: "go.example.com", Modules: map[string]Module{"a": {Repo: "https://github.com/x/a"}}}, "GET", "/a?go-get=1")
	if got := meta(w, "go-import"); got != "go.example.com/a git https://github.com/x/a" {
		t.Error(got)
	}
}

func TestMethods(t *testing.T) {
	if w := serve(t, Config{}, "POST", "/chonker"); w.Code != 405 {
		t.Error(w.Code)
	}
}

func TestBadConfig(t *testing.T) {
	if _, err := Handler(Config{Browser: "nope"}); err == nil {
		t.Error("bad browser: no error")
	}
	for _, m := range []Module{{}, {Repo: "nope"}, {Repo: "https://x.org/a", Source: "svnweb"}} {
		if _, err := Handler(Config{Modules: map[string]Module{"a": m}}); err == nil {
			t.Errorf("%+v: no error", m)
		}
	}
}

func TestEscaping(t *testing.T) {
	w := serve(t, Config{Modules: map[string]Module{"a": {Repo: `https://github.com/x/a"><script>`, Description: "<b>"}}}, "GET", "/a")
	if strings.Contains(w.Body.String(), "<script>") || strings.Contains(w.Body.String(), "<b>") {
		t.Error(w.Body)
	}
}
