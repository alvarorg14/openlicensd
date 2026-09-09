package store

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var errNoFieldsToUpdate = errors.New("no fields to update")

type LicensePatch struct {
	Label             *string
	ExpiresAtSet      bool
	ExpiresAt         *time.Time
	MaxActivationsSet bool
	MaxActivations    *int
}

type ProductPatch struct {
	Name            *string
	Code            *string
	DescriptionSet  bool
	Description     *string
}

type PolicyPatch struct {
	Name               *string
	DescriptionSet     bool
	Description        *string
	DurationDaysSet    bool
	DurationDays       *int
	ExpirationBasisSet bool
	ExpirationBasis    ExpirationBasis
	GracePeriodDaysSet bool
	GracePeriodDays    int
	MaxActivationsSet  bool
	MaxActivations     *int
}

type setBuilder struct {
	sets []string
	args []any
	next int
}

func newSetBuilder(startArg int) *setBuilder {
	return &setBuilder{next: startArg}
}

func (b *setBuilder) add(column string, value any) {
	b.sets = append(b.sets, fmt.Sprintf("%s = $%d", column, b.next))
	b.args = append(b.args, value)
	b.next++
}

func (b *setBuilder) addExpr(expr string) {
	b.sets = append(b.sets, expr)
}

func (b *setBuilder) expr() (string, []any, error) {
	if len(b.sets) == 0 {
		return "", nil, errNoFieldsToUpdate
	}
	return strings.Join(b.sets, ", "), b.args, nil
}
