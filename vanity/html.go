package vanity

import (
	_ "embed"
	"html"
	"strings"
)

// Pages are written by hand rather than with html/template, which TinyGo
// cannot run (its reflect has no Type.NumOut). Every interpolated value goes
// through e.

//go:embed style.css
var style string

var e = html.EscapeString

const copyButton = `<button type="button" onclick="navigator.clipboard.writeText(this.previousElementSibling.textContent).then(()=>{this.textContent='copied';setTimeout(()=>this.textContent='copy',1200)})">copy</button>`

const filterInput = `<input type="search" placeholder="Filter" aria-label="Filter modules" oninput="for(const li of document.querySelectorAll('ul.mods li'))li.hidden=!li.textContent.toLowerCase().includes(this.value.toLowerCase())">`

func writePage(b *strings.Builder, title, head string, body func()) {
	b.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString(head)
	b.WriteString("<title>" + e(title) + "</title>\n<style>\n" + style + "</style>\n</head>\n<body>\n<main>\n")
	body()
	b.WriteString("<footer>Served by <a href=\"https://github.com/ananthb/go-vanity\">go-vanity</a>.</footer>\n</main>\n</body>\n</html>\n")
}

func indexHTML(p page) string {
	var b strings.Builder
	writePage(&b, p.Title, "", func() {
		b.WriteString("<h1>" + e(p.Title) + "</h1>\n")
		b.WriteString("<p class=\"sub\">Go modules at <code>" + e(p.Host) + "</code>.</p>\n")
		if len(p.Modules) > 5 {
			b.WriteString(filterInput + "\n")
		}
		b.WriteString("<ul class=\"mods\">\n")
		for _, m := range p.Modules {
			b.WriteString("<li><a href=\"/" + e(m.Prefix) + "\">" + e(p.Host+"/"+m.Prefix) + "</a>")
			if m.Description != "" {
				b.WriteString("<p>" + e(m.Description) + "</p>")
			}
			b.WriteString("</li>\n")
		}
		b.WriteString("</ul>\n")
	})
	return b.String()
}

func moduleHTML(p page) string {
	m := p.Module
	root := p.Host + "/" + m.Prefix
	head := "<meta name=\"go-import\" content=\"" + e(root+" "+m.VCS+" "+m.Repo) + "\">\n"
	if m.Source != nil {
		head += "<meta name=\"go-source\" content=\"" + e(root+" "+m.Repo+" "+m.Source[0]+" "+m.Source[1]) + "\">\n"
	}
	var b strings.Builder
	writePage(&b, p.Path, head, func() {
		b.WriteString("<h1>" + e(p.Path) + "</h1>\n<p class=\"sub\">")
		if m.Description != "" {
			b.WriteString(e(m.Description) + " ")
		}
		b.WriteString("<a href=\"/\">All modules</a></p>\n")
		b.WriteString("<div class=\"cmd\"><code>go get " + e(p.Path) + "</code>" + copyButton + "</div>\n")
		b.WriteString("<dl>\n<dt>Source</dt><dd><a href=\"" + e(m.Repo) + "\">" + e(m.Repo) + "</a></dd>\n")
		b.WriteString("<dt>Docs</dt><dd><a href=\"https://pkg.go.dev/" + e(p.Path) + "\">pkg.go.dev/" + e(p.Path) + "</a></dd>\n</dl>\n")
	})
	return b.String()
}

func notFoundHTML(p page) string {
	var b strings.Builder
	writePage(&b, "Not found", "", func() {
		b.WriteString("<h1><a href=\"/\">" + e(p.Title) + "</a></h1>\n")
		b.WriteString("<p class=\"sub\">No module at <code>" + e(p.Host+p.Path) + "</code>.</p>\n")
	})
	return b.String()
}
