package store

import (
	"fmt"
	"strings"
)

const (
	DefaultPageSize = 25
	MaxPageSize     = 100
)

// ListParams holds common pagination, search and sort options for list queries.
type ListParams struct {
	Search string
	Sort   string // SQL column expression (validated by API layer)
	Order  string // "asc" or "desc"
	Limit  int
	Offset int
}

// LicenseListParams extends ListParams with license-specific filters.
type LicenseListParams struct {
	ListParams
	Status    string // "active", "expired", "revoked", or ""
	ProductID *string
	PolicyID  *string
}

// PolicyListParams extends ListParams with policy-specific filters.
type PolicyListParams struct {
	ListParams
	ProductID *string
}

// LicenseStats holds aggregate license counts by status.
type LicenseStats struct {
	Total   int64
	Active  int64
	Expired int64
	Revoked int64
}

type queryBuilder struct {
	conditions []string
	args       []any
}

func newQueryBuilder() *queryBuilder {
	return &queryBuilder{}
}

func (qb *queryBuilder) addExpr(condition string) {
	qb.conditions = append(qb.conditions, condition)
}

func (qb *queryBuilder) add(condition string, args ...any) {
	if len(args) == 0 {
		qb.addExpr(condition)
		return
	}
	start := len(qb.args) + 1
	qb.args = append(qb.args, args...)
	placeholders := make([]any, len(args))
	for i := range args {
		placeholders[i] = start + i
	}
	qb.conditions = append(qb.conditions, fmt.Sprintf(condition, placeholders...))
}

func (qb *queryBuilder) whereClause() string {
	if len(qb.conditions) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(qb.conditions, " AND ")
}

func normalizeOrder(order string) string {
	if strings.EqualFold(order, "asc") {
		return "ASC"
	}
	return "DESC"
}

func buildOrderBy(sortExpr, order, tiebreaker string) string {
	return fmt.Sprintf(" ORDER BY %s %s, %s %s", sortExpr, normalizeOrder(order), tiebreaker, normalizeOrder(order))
}

func searchPattern(search string) string {
	return "%" + search + "%"
}

func limitOffsetClause(argOffset int) string {
	return fmt.Sprintf(" LIMIT $%d OFFSET $%d", argOffset, argOffset+1)
}
