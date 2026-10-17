package domain

import "testing"

// Business rules as test cases. Bodies are filled in during M2.

func todo(t *testing.T) {
	t.Helper()
	t.Fatal("not implemented (M2)")
}

func TestReceipt_RejectsZeroOrNegativeQuantity(t *testing.T)  { todo(t) }
func TestReceipt_RejectsZeroOrNegativeCost(t *testing.T)      { todo(t) }
func TestDispatch_RejectsZeroOrNegativeQuantity(t *testing.T) { todo(t) }
func TestPieces_RejectFractionalQuantity(t *testing.T)        { todo(t) }
func TestKgAndLitre_AllowFractionalQuantity(t *testing.T)     { todo(t) }
func TestTransfer_RejectsSameLocation(t *testing.T)           { todo(t) }
func TestAdjustment_RequiresKnownReason(t *testing.T)         { todo(t) }
func TestAdjustment_OtherRequiresNote(t *testing.T)           { todo(t) }
func TestAdjustment_RejectsZeroQuantity(t *testing.T)         { todo(t) }

func TestWeightedAverage_TwoReceipts(t *testing.T)               { todo(t) } // 50@1000 + 50@1200 = 1100
func TestWeightedAverage_FirstReceiptFromZero(t *testing.T)      { todo(t) }
func TestWeightedAverage_RoundingAfterManyReceipts(t *testing.T) { todo(t) }
func TestWeightedAverage_UnchangedByTransfer(t *testing.T)       { todo(t) }