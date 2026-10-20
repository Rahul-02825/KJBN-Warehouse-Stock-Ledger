package ledger

import (
	"strings"

	"github.com/shopspring/decimal"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
)

func requireID(fields map[string]string, name string, id int64) {
	if id < 1 {
		fields[name] = "is required and must be a positive integer"
	}
}

func optionalID(fields map[string]string, name string, id int64) {
	if id < 0 {
		fields[name] = "must be a positive integer"
	}
}

func checkAmount(fields map[string]string, name string, amount decimal.Decimal) {
	if msg := domain.ValidateQuantity(amount); msg != "" {
		fields[name] = msg
	}
}

func checkReference(fields map[string]string, reference string) string {
	reference = strings.TrimSpace(reference)
	if len(reference) > 128 {
		fields["reference"] = "max 128 characters"
	}
	return reference
}

func checkCreatedBy(fields map[string]string, createdBy string) string {
	createdBy = strings.TrimSpace(createdBy)
	if createdBy == "" || len(createdBy) > 128 {
		fields["created_by"] = "is required, max 128 characters"
	}
	return createdBy
}

func checkNote(fields map[string]string, note string) string {
	note = strings.TrimSpace(note)
	if len(note) > 1000 {
		fields["note"] = "max 1000 characters"
	}
	return note
}

func result(fields map[string]string) error {
	if len(fields) > 0 {
		return domain.ValidationError(fields)
	}
	return nil
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
