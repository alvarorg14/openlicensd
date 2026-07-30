package api

import (
	"errors"
	"math"
	"net/http"
	"strconv"
)

type pageResponse[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type parsedListParams struct {
	Search   string
	Sort     string
	Order    string
	Page     int
	PageSize int
	Limit    int
	Offset   int
}

var (
	errInvalidPage     = errors.New("invalid page")
	errInvalidPageSize = errors.New("invalid page_size")
	errInvalidSort     = errors.New("invalid sort")
	errInvalidOrder    = errors.New("invalid order")
)

func parseListParams(r *http.Request, allowedSorts map[string]string) (parsedListParams, error) {
	q := r.URL.Query()

	page := 1
	if raw := q.Get("page"); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil || p < 1 {
			return parsedListParams{}, errInvalidPage
		}
		page = p
	}

	pageSize := 25
	if raw := q.Get("page_size"); raw != "" {
		ps, err := strconv.Atoi(raw)
		if err != nil || ps < 1 || ps > 100 {
			return parsedListParams{}, errInvalidPageSize
		}
		pageSize = ps
	}

	sortKey := q.Get("sort")
	if sortKey == "" {
		sortKey = "created_at"
	}
	sortExpr, ok := allowedSorts[sortKey]
	if !ok {
		return parsedListParams{}, errInvalidSort
	}

	order := q.Get("order")
	if order == "" {
		order = "desc"
	}
	if order != "asc" && order != "desc" {
		return parsedListParams{}, errInvalidOrder
	}

	return parsedListParams{
		Search:   q.Get("search"),
		Sort:     sortExpr,
		Order:    order,
		Page:     page,
		PageSize: pageSize,
		Limit:    pageSize,
		Offset:   (page - 1) * pageSize,
	}, nil
}

func newPageResponse[T any](items []T, page, pageSize int, total int64) pageResponse[T] {
	if items == nil {
		items = []T{}
	}
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(pageSize)))
	}
	return pageResponse[T]{
		Items:      items,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

func writeListParamError(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, errInvalidPage):
		writeError(w, http.StatusBadRequest, "invalid page")
	case errors.Is(err, errInvalidPageSize):
		writeError(w, http.StatusBadRequest, "invalid page_size")
	case errors.Is(err, errInvalidSort):
		writeError(w, http.StatusBadRequest, "invalid sort")
	case errors.Is(err, errInvalidOrder):
		writeError(w, http.StatusBadRequest, "invalid order")
	default:
		return false
	}
	return true
}

var licenseSorts = map[string]string{
	"created_at":          "l.created_at",
	"label":               "l.label",
	"expires_at":          "l.expires_at",
	"product_name":        "p.name",
	"policy_name":         "pol.name",
	"last_validated_at":   "l.last_validated_at",
	"validation_count":    "l.validation_count",
}

var productSorts = map[string]string{
	"created_at": "created_at",
	"updated_at": "updated_at",
	"name":       "name",
	"code":       "code",
}

var policySorts = map[string]string{
	"created_at":        "pol.created_at",
	"name":              "pol.name",
	"product_name":      "p.name",
	"grace_period_days": "pol.grace_period_days",
}
