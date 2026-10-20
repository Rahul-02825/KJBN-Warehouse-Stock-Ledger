package ledger

import (
	"context"

	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/domain"
	"github.com/Rahul-02825/KJBN-Warehouse-Stock-Ledger/backend/internal/postgres/sqlc"
)

func (l *Ledger) findReplay(ctx context.Context, key, movementType string, productID int64) (sqlc.Movement, bool, error) {
	original, err := sqlc.New(l.pool).GetMovementByIdempotencyKey(ctx, &key)
	if err != nil {
		return sqlc.Movement{}, false, err
	}
	if original.MovementType != movementType || original.ProductID != productID {
		return sqlc.Movement{}, false, domain.ErrKeyReused
	}
	return original, true, nil
}

func (l *Ledger) findReversalReplay(ctx context.Context, key string, movementID int64) (sqlc.Movement, bool, error) {
	original, err := sqlc.New(l.pool).GetMovementByIdempotencyKey(ctx, &key)
	if err != nil {
		return sqlc.Movement{}, false, err
	}
	if original.MovementType != "REVERSAL" || original.ReversalOfID == nil || *original.ReversalOfID != movementID {
		return sqlc.Movement{}, false, domain.ErrKeyReused
	}
	return original, true, nil
}
