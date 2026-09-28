package web

import (
	"net/http"
	"strconv"

	"github.com/romus204/tooler/internal/models"
)

func (srv *Server) handleVariantCreate(w http.ResponseWriter, r *http.Request) {
	toolID, err := parseID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	v := models.ToolVariant{
		ToolID:   toolID,
		Size:     r.FormValue("size"),
		SKU:      r.FormValue("sku"),
		Location: r.FormValue("location"),
		Notes:    r.FormValue("notes"),
		Quantity: atoiDefault(r.FormValue("quantity"), 0),
	}
	if v.Size == "" {
		http.Error(w, "размер обязателен", http.StatusBadRequest)
		return
	}

	if _, err := srv.store.CreateVariant(r.Context(), v); err != nil {
		srv.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/tools/"+strconv.FormatInt(toolID, 10), http.StatusSeeOther)
}

type variantFormData struct {
	Variant    models.ToolVariant
	ToolName   string
	FormAction string
}

func (srv *Server) handleVariantFormEdit(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	v, err := srv.store.GetVariant(r.Context(), id)
	if err != nil {
		srv.renderError(w, r, err)
		return
	}
	tool, err := srv.store.GetTool(r.Context(), v.ToolID)
	if err != nil {
		srv.renderError(w, r, err)
		return
	}
	render(w, srv.tmplVariantForm, PageData{
		Title: "Редактировать размер",
		Data: variantFormData{
			Variant:    v,
			ToolName:   tool.Name,
			FormAction: "/variants/" + strconv.FormatInt(v.ID, 10),
		},
	})
}

func (srv *Server) handleVariantUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	existing, err := srv.store.GetVariant(r.Context(), id)
	if err != nil {
		srv.renderError(w, r, err)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	existing.Size = r.FormValue("size")
	existing.SKU = r.FormValue("sku")
	existing.Location = r.FormValue("location")
	existing.Notes = r.FormValue("notes")

	if err := srv.store.UpdateVariant(r.Context(), existing); err != nil {
		srv.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/tools/"+strconv.FormatInt(existing.ToolID, 10), http.StatusSeeOther)
}

func (srv *Server) handleVariantDelete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	v, err := srv.store.GetVariant(r.Context(), id)
	if err != nil {
		srv.renderError(w, r, err)
		return
	}
	if err := srv.store.DeleteVariant(r.Context(), id); err != nil {
		srv.renderError(w, r, err)
		return
	}
	http.Redirect(w, r, "/tools/"+strconv.FormatInt(v.ToolID, 10), http.StatusSeeOther)
}

func (srv *Server) handleVariantAdjust(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	delta := atoiDefault(r.FormValue("delta"), 0)
	reason := r.FormValue("reason")

	v, err := srv.store.GetVariant(r.Context(), id)
	if err != nil {
		srv.renderError(w, r, err)
		return
	}
	if delta != 0 {
		if _, err := srv.store.AdjustQuantity(r.Context(), id, delta, reason, "web"); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	http.Redirect(w, r, "/tools/"+strconv.FormatInt(v.ToolID, 10), http.StatusSeeOther)
}

type variantHistoryData struct {
	Variant   models.VariantWithTool
	Movements []models.StockMovement
}

func (srv *Server) handleVariantHistory(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	v, err := srv.store.GetVariantWithTool(r.Context(), id)
	if err != nil {
		srv.renderError(w, r, err)
		return
	}
	movements, err := srv.store.ListMovementsForVariant(r.Context(), id, 100)
	if err != nil {
		srv.renderError(w, r, err)
		return
	}
	render(w, srv.tmplVariantHistory, PageData{
		Title: "История: " + v.ToolName + " " + v.Size,
		Data:  variantHistoryData{Variant: v, Movements: movements},
	})
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
