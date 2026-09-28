package web

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/romus204/tooler/internal/models"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

var funcMap = template.FuncMap{
	"indent": func(depth int) string {
		return strings.Repeat("— ", depth)
	},
}

func mustParse(files ...string) *template.Template {
	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = "templates/" + f
	}
	return template.Must(template.New("layout.html").Funcs(funcMap).ParseFS(templatesFS, paths...))
}

// PageData — общие данные, передаваемые во все страницы.
type PageData struct {
	Title string
	Query string
	Flash string
	Error string
	Data  any
}

func render(w http.ResponseWriter, tmpl *template.Template, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("render error: %v", err)
		http.Error(w, "внутренняя ошибка сервера", http.StatusInternalServerError)
	}
}

// categoryOption — плоское представление категории для выпадающих списков.
type categoryOption struct {
	ID       int64
	Name     string
	Depth    int
	Selected bool
}

// flattenCategories строит плоский список категорий с глубиной вложенности
// для отображения в <select>. excludeID (если не nil) исключает узел
// и всех его потомков (чтобы нельзя было назначить категорию родителем самой себе).
func flattenCategories(nodes []*models.CategoryNode, selectedID *int64, excludeID *int64, depth int) []categoryOption {
	var out []categoryOption
	for _, n := range nodes {
		if excludeID != nil && n.ID == *excludeID {
			continue
		}
		out = append(out, categoryOption{
			ID:       n.ID,
			Name:     n.Name,
			Depth:    depth,
			Selected: selectedID != nil && *selectedID == n.ID,
		})
		out = append(out, flattenCategories(n.Children, selectedID, excludeID, depth+1)...)
	}
	return out
}
