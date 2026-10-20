package ledger

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Ledger struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Ledger {
	return &Ledger{pool: pool}
}

type stockChange struct {
	LocationID int64
	Delta      decimal.Decimal
}

var errReplay = errors.New("idempotency key already used")
