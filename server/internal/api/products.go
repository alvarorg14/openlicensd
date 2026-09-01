package api

import (
	"encoding/json"
	"net/http"

	"github.com/alvarorg14/openlicensd/server/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type productResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Code        string    `json:"code"`
	Description *string   `json:"description"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

type createProductRequest struct {
	Name        string  `json:"name"`
	Code        string  `json:"code"`
	Description *string `json:"description"`
}

type updateProductRequest struct {
	Name        string  `json:"name"`
	Code        string  `json:"code"`
	Description *string `json:"description"`
}

func productToResponse(p *store.Product) productResponse {
	return productResponse{
		ID:          p.ID,
		Name:        p.Name,
		Code:        p.Code,
		Description: p.Description,
		CreatedAt:   p.CreatedAt.Format(timeRFC3339),
		UpdatedAt:   p.UpdatedAt.Format(timeRFC3339),
	}
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func (s *Server) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	product, err := s.store.CreateProduct(r.Context(), req.Name, req.Code, req.Description)
	if err != nil {
		if writeStoreError(w, err, "") {
			return
		}
		writeInternalError(w, r, err, "failed to create product")
		return
	}

	writeJSON(w, http.StatusCreated, productToResponse(product))
}

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	params, err := parseListParams(r, productSorts)
	if err != nil {
		if writeListParamError(w, err) {
			return
		}
		writeError(w, http.StatusBadRequest, "invalid list parameters")
		return
	}

	products, total, err := s.store.ListProducts(r.Context(), store.ListParams{
		Search: params.Search,
		Sort:   params.Sort,
		Order:  params.Order,
		Limit:  params.Limit,
		Offset: params.Offset,
	})
	if err != nil {
		writeInternalError(w, r, err, "failed to list products")
		return
	}

	resp := make([]productResponse, 0, len(products))
	for _, p := range products {
		resp = append(resp, productToResponse(&p))
	}

	writeJSON(w, http.StatusOK, newPageResponse(resp, params.Page, params.PageSize, total))
}

func (s *Server) handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	var req updateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Code == "" {
		writeError(w, http.StatusBadRequest, "code is required")
		return
	}

	product, err := s.store.UpdateProduct(r.Context(), id, req.Name, req.Code, req.Description)
	if err != nil {
		if writeStoreError(w, err, "") {
			return
		}
		writeInternalError(w, r, err, "failed to update product")
		return
	}
	if product == nil {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}

	writeJSON(w, http.StatusOK, productToResponse(product))
}

func (s *Server) handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid product id")
		return
	}

	deleted, err := s.store.DeleteProduct(r.Context(), id)
	if err != nil {
		if writeStoreError(w, err, "") {
			return
		}
		writeInternalError(w, r, err, "failed to delete product")
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
