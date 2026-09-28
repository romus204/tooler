package web

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/romus204/tooler/internal/models"
	"github.com/romus204/tooler/internal/store"
)

type toolsListData struct {
	Tools           []models.ToolWithStock
	CategoryOptions []categoryOption
}

func (srv *Server) handleToolsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query().Get("q")

	filter := store.ToolFilter{Search: q}
	var selectedCategory *int64
	if raw := r.URL.Query().Get("category"); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			filter.CategoryID = &id
			selectedCategory = &id
		}
	}

	tools, err := srv.store.ListTools(ctx, filter)
	if err != nil {
		srv.renderError(w, r, err)
		return
	}

	tree, err := srv.store.CategoryTree(ctx)
	if err != nil {
		srv.renderError(w, r, err)
		return
	}

	render(w, srv.tmplToolsList, PageData{
		Title: "Инструменты",
		Query: q,
		Data: toolsListData{
			Tools:           tools,
			CategoryOptions: flattenCategories(tree, selectedCategory, nil, 0),
		},
	})
}

type toolFormData struct {
	Tool            *models.Tool
	CategoryOptions []categoryOption
	FormAction      string
}

func (srv *Server) handleToolFormNew(w http.ResponseWriter, r *http.Request) {
	tree, err := srv.store.CategoryTree(r.Context())
	if err != nil {
		srv.renderError(w, r, err)
		return
	}
	render(w, srv.tmplToolForm, PageData{
		Title: "Новый инструмент",
		Data: toolFormData{
			CategoryOptions: flattenCategories(tree, nil, nil, 0),
			FormAction:      "/tools",
		},
	})
}

func (srv *Server) handleToolFormEdit(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	tool, err := srv.store.GetTool(r.Context(), id)
	if err != nil {
		srv.renderError(w, r, err)
		return
	}
	tree, err := srv.store.CategoryTree(r.Context())
	if err != nil {
		srv.renderError(w, r, err)
		return
	}
	render(w, srv.tmplToolForm, PageData{
		Title: "Редактировать: " + tool.Name,
		Data: toolFormData{
			Tool:            &tool,
			CategoryOptions: flattenCategories(tree, tool.CategoryID, nil, 0),
			FormAction:      "/tools/" + strconv.FormatInt(tool.ID, 10),
		},
	})
}

func (srv *Server) handleToolCreate(w http.ResponseWriter, r *http.Request) {
	t, err := toolFromForm(r, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, err := srv.store.CreateTool(r.Context(), t)
	if err != nil {
		srv.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/tools/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (srv *Server) handleToolUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	t, err := toolFromForm(r, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := srv.store.UpdateTool(r.Context(), t); err != nil {
		srv.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/tools/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (srv *Server) handleToolDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := srv.store.DeleteTool(r.Context(), id); err != nil {
		srv.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/tools", http.StatusSeeOther)
}

type toolDetailData struct {
	Tool models.ToolDetail
}

func (srv *Server) handleToolDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	detail, err := srv.store.GetToolDetail(r.Context(), id)
	if err != nil {
		srv.renderError(w, r, err)
		return
	}
	render(w, srv.tmplToolDetail, PageData{
		Title: detail.Name,
		Data:  toolDetailData{Tool: detail},
	})
}

func toolFromForm(r *http.Request, id int64) (models.Tool, error) {
	if err := r.ParseForm(); err != nil {
		return models.Tool{}, err
	}
	t := models.Tool{
		ID:          id,
		Name:        r.FormValue("name"),
		Brand:       r.FormValue("brand"),
		Description: r.FormValue("description"),
		Unit:        r.FormValue("unit"),
	}
	if t.Unit == "" {
		t.Unit = "шт"
	}
	if raw := r.FormValue("category_id"); raw != "" {
		catID, err := strconv.ParseInt(raw, 10, 64)
		if err == nil {
			t.CategoryID = &catID
		}
	}
	return t, nil
}

func parseID(r *http.Request, param string) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, param), 10, 64)
}

func (srv *Server) renderError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, store.ErrNotFound) {
		status = http.StatusNotFound
	}
	http.Error(w, err.Error(), status)
}
