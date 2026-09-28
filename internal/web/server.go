// Package web реализует веб-морду для каталога инструментов.
package web

import (
	"crypto/subtle"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/romus204/tooler/internal/store"
)

type Server struct {
	store *store.Store
	auth  *basicAuth

	tmplToolsList      *template.Template
	tmplToolForm       *template.Template
	tmplToolDetail     *template.Template
	tmplCategories     *template.Template
	tmplVariantForm    *template.Template
	tmplVariantHistory *template.Template
}

type basicAuth struct {
	username string
	password string
}

func NewServer(s *store.Store, webUsername, webPassword string) *Server {
	srv := &Server{
		store:              s,
		tmplToolsList:      mustParse("layout.html", "tools_list.html"),
		tmplToolForm:       mustParse("layout.html", "tool_form.html"),
		tmplToolDetail:     mustParse("layout.html", "tool_detail.html"),
		tmplCategories:     mustParse("layout.html", "categories.html"),
		tmplVariantForm:    mustParse("layout.html", "variant_form.html"),
		tmplVariantHistory: mustParse("layout.html", "variant_history.html"),
	}
	if webUsername != "" || webPassword != "" {
		srv.auth = &basicAuth{username: webUsername, password: webPassword}
	}
	return srv
}

func (srv *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	if srv.auth != nil {
		r.Use(srv.basicAuthMiddleware)
	}

	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/tools", http.StatusFound)
	})

	r.Get("/tools", srv.handleToolsList)
	r.Get("/tools/new", srv.handleToolFormNew)
	r.Post("/tools", srv.handleToolCreate)
	r.Get("/tools/{id}", srv.handleToolDetail)
	r.Get("/tools/{id}/edit", srv.handleToolFormEdit)
	r.Post("/tools/{id}", srv.handleToolUpdate)
	r.Post("/tools/{id}/delete", srv.handleToolDelete)
	r.Post("/tools/{id}/variants", srv.handleVariantCreate)

	r.Get("/variants/{id}/edit", srv.handleVariantFormEdit)
	r.Post("/variants/{id}", srv.handleVariantUpdate)
	r.Post("/variants/{id}/delete", srv.handleVariantDelete)
	r.Post("/variants/{id}/adjust", srv.handleVariantAdjust)
	r.Get("/variants/{id}/history", srv.handleVariantHistory)

	r.Get("/categories", srv.handleCategories)
	r.Get("/categories/{id}/edit", srv.handleCategories)
	r.Post("/categories", srv.handleCategoryCreate)
	r.Post("/categories/{id}", srv.handleCategoryUpdate)
	r.Post("/categories/{id}/delete", srv.handleCategoryDelete)

	return r
}

func (srv *Server) basicAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(user), []byte(srv.auth.username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(pass), []byte(srv.auth.password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="tooler"`)
			http.Error(w, "требуется авторизация", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
