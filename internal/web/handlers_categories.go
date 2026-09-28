package web

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/romus204/tooler/internal/models"
)

type categoriesPageData struct {
	Tree            []*models.CategoryNode
	Editing         *models.Category
	CategoryOptions []categoryOption
	FormAction      string
}

func (srv *Server) handleCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tree, err := srv.store.CategoryTree(ctx)
	if err != nil {
		srv.renderError(w, r, err)
		return
	}

	var editing *models.Category
	formAction := "/categories"
	var excludeID *int64

	if idParam := chi.URLParam(r, "id"); idParam != "" {
		id, err := strconv.ParseInt(idParam, 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		cat, err := srv.store.GetCategory(ctx, id)
		if err != nil {
			srv.renderError(w, r, err)
			return
		}
		editing = &cat
		formAction = "/categories/" + idParam
		excludeID = &id
	}

	var selected *int64
	if editing != nil {
		selected = editing.ParentID
	}

	render(w, srv.tmplCategories, PageData{
		Title: "Категории",
		Data: categoriesPageData{
			Tree:            tree,
			Editing:         editing,
			CategoryOptions: flattenCategories(tree, selected, excludeID, 0),
			FormAction:      formAction,
		},
	})
}

func (srv *Server) handleCategoryCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "название обязательно", http.StatusBadRequest)
		return
	}
	var parentID *int64
	if raw := r.FormValue("parent_id"); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			parentID = &id
		}
	}
	if _, err := srv.store.CreateCategory(r.Context(), name, parentID); err != nil {
		srv.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

func (srv *Server) handleCategoryUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "название обязательно", http.StatusBadRequest)
		return
	}
	var parentID *int64
	if raw := r.FormValue("parent_id"); raw != "" {
		if pid, err := strconv.ParseInt(raw, 10, 64); err == nil && pid != id {
			parentID = &pid
		}
	}
	if err := srv.store.UpdateCategory(r.Context(), id, name, parentID); err != nil {
		srv.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

func (srv *Server) handleCategoryDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := srv.store.DeleteCategory(r.Context(), id); err != nil {
		srv.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}
