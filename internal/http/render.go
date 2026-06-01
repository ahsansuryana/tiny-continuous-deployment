package http

import (
	"bytes"
	"html/template"
	"io/fs"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type Renderer struct {
	templates *template.Template
}

func NewRenderer(fsys fs.FS, glob string) (*Renderer, error) {
	funcMap := template.FuncMap{
		"timeAgo":    timeAgo,
		"upperFirst": upperFirst,
		"join":       strings.Join,
		"splitLines": splitLines,
	}

	tmpl := template.New("").Funcs(funcMap)

	err := fs.WalkDir(fsys, path.Dir(glob), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".html") {
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		_, err = tmpl.Parse(string(data))
		return err
	})
	if err != nil {
		return nil, err
	}

	return &Renderer{templates: tmpl}, nil
}

func (r *Renderer) Render(w http.ResponseWriter, req *http.Request, page string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if req.Header.Get("HX-Request") == "true" {
		r.templates.ExecuteTemplate(w, page, data)
		return
	}

	standalone := map[string]bool{"login.html": true, "error.html": true}
	if standalone[page] {
		r.templates.ExecuteTemplate(w, page, data)
		return
	}

	var buf bytes.Buffer
	if err := r.templates.ExecuteTemplate(&buf, page, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err := r.templates.ExecuteTemplate(w, "base.html", map[string]interface{}{
		"Content": template.HTML(buf.String()),
		"Data":    data,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (r *Renderer) RenderError(w http.ResponseWriter, req *http.Request, status int, message string) {
	w.WriteHeader(status)
	r.Render(w, req, "error.html", map[string]string{
		"Message": message,
	})
}

func timeAgo(t string) string {
	if t == "" {
		return ""
	}

	parsed, err := time.Parse("2006-01-02 15:04:05", t)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, t)
		if err != nil {
			return t
		}
	}

	diff := time.Since(parsed)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		m := int(diff.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return strconv.Itoa(m) + " minutes ago"
	case diff < 24*time.Hour:
		h := int(diff.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return strconv.Itoa(h) + " hours ago"
	case diff < 30*24*time.Hour:
		d := int(diff.Hours() / 24)
		if d == 1 {
			return "yesterday"
		}
		return strconv.Itoa(d) + " days ago"
	default:
		return parsed.Format("Jan 2, 2006")
	}
}

func upperFirst(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
