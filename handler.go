package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"go.uber.org/sally/templates"
)

var (
	indexTemplate = template.Must(
		template.New("index.html").Parse(templates.Index))
	packageTemplate = template.Must(
		template.New("package.html").Parse(templates.Package))
)

// CreateHandler creates a Sally http.Handler
func CreateHandler(config *Config) http.Handler {
	h := sallyHandler{config: config}
	for name, pkg := range config.Packages {
		h.packages.Set(name, pkg)
	}
	return &h
}

type sallyHandler struct {
	config   *Config
	packages pathTree[Package]
}

func (h *sallyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	pkgName, pkg, ok := h.packages.Lookup(path)
	if !ok {
		pkgs := h.packages.ListByPath(path)
		if len(pkgs) == 0 {
			if r.Method != http.MethodGet {
				http.NotFound(w, r)
				return
			}
		}

		var data struct {
			URL      string
			Packages map[string]Package
			Godoc    struct{ Host string }
		}

		data.URL = h.config.URL
		data.Packages = pkgs
		data.Godoc.Host = h.config.Godoc.Host

		if err := indexTemplate.Execute(w, data); err != nil {
			http.Error(w, err.Error(), 500)
		}

		return
	}

	// Extract the relative path to subpackages, if any.
	//	"/foo/bar" => "/bar"
	//	"/foo" => ""
	relPath := strings.TrimPrefix(r.URL.Path, "/"+pkgName)

	baseURL := h.config.URL
	if pkg.URL != "" {
		baseURL = pkg.URL
	}
	canonicalURL := fmt.Sprintf("%s/%s", baseURL, pkgName)

	var data struct {
		Repo         string
		Branch       string
		CanonicalURL string
		GodocURL     string
	}
	data.Repo = pkg.Repo
	data.Branch = pkg.Branch
	data.CanonicalURL = canonicalURL
	data.GodocURL = fmt.Sprintf("https://%s/%s%s", h.config.Godoc.Host, canonicalURL, relPath)
	if err := packageTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
