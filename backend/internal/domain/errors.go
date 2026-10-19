package domain

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

type ValidationError map[string]string

func (e ValidationError) Error() string { return "validation failed" }

type NotFoundError struct{ What string }

func (e NotFoundError) Error() string { return e.What + " not found" }

type InsufficientStockError struct {
	Location  string
	Available decimal.Decimal
	Requested decimal.Decimal
}

func (e InsufficientStockError) Error() string {
	return fmt.Sprintf("insufficient stock at %s: available %s, requested %s", e.Location, e.Available, e.Requested)
}

type NoSpaceError struct {
	Place     string
	Fit       decimal.Decimal
	Requested decimal.Decimal
}

func (e NoSpaceError) Error() string {
	return fmt.Sprintf("cannot accommodate %s units in %s: only %s fit", e.Requested, e.Place, e.Fit)
}

type NotReversibleError struct{ Reason string }

func (e NotReversibleError) Error() string { return e.Reason }

var ErrAlreadyReversed = errors.New("movement has already been reversed")

var ErrKeyReused = errors.New("this Idempotency-Key was already used for a different request")
