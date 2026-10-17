package application

import "testing"

// Transactional rules, run against PostgreSQL. Bodies are filled in during M2.

func todo(t *testing.T) {
	t.Helper()
	t.Fatal("not implemented (M2)")
}

func TestIn_UpdatesBalanceLedgerAndAverage(t *testing.T)          { todo(t) }
func TestIn_RejectsUnknownProductOrLocation(t *testing.T)         { todo(t) }
func TestOut_RejectsInsufficientStock(t *testing.T)               { todo(t) }
func TestOut_FreezesCostAtPosting(t *testing.T)                   { todo(t) }
func TestOut_ConcurrentRequestsCannotGoNegative(t *testing.T)     { todo(t) }
func TestTransfer_UpdatesBothBinsOrNeither(t *testing.T)          { todo(t) }
func TestTransfer_RejectsInsufficientStock(t *testing.T)          { todo(t) }
func TestTransfer_IsOneMovementWithTwoEntries(t *testing.T)       { todo(t) }
func TestAdjustment_RejectsNegativeResultingStock(t *testing.T)   { todo(t) }
func TestReversal_PreservesOriginalAndMarksReversed(t *testing.T) { todo(t) }
func TestReversal_RejectsDuplicate(t *testing.T)                  { todo(t) }
func TestReversal_RejectsReversingAReversal(t *testing.T)         { todo(t) }
func TestReversal_RejectsNegativeResultingStock(t *testing.T)     { todo(t) }
func TestLowStock_AtOrBelowReorderLevel(t *testing.T)             { todo(t) }
func TestReconcile_DetectsCorruptedCache(t *testing.T)            { todo(t) }
