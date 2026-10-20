package domain

import (
	"testing"
)

func capacity(n int64) *int64 { return &n }

func layout() []Bin {
	return []Bin{
		{ID: 1, Code: "A-01", Position: Point{X: 5, Y: 2}, Capacity: capacity(60), Used: 45, AutoPlace: true},
		{ID: 2, Code: "A-02", Position: Point{X: 5, Y: 8}, Capacity: capacity(60), Used: 20, AutoPlace: true},
		{ID: 3, Code: "B-01", Position: Point{X: 15, Y: 2}, Capacity: capacity(100), Used: 10, AutoPlace: true},
		{ID: 4, Code: "B-02", Position: Point{X: 15, Y: 8}, Capacity: capacity(80), Used: 70, AutoPlace: true},
		{ID: 5, Code: "DOCK", Position: Point{X: 0, Y: 0}, AutoPlace: false},
	}
}

func placed(plan []Allocation) map[string]int64 {
	out := map[string]int64{}
	for _, a := range plan {
		out[a.Bin.Code] = a.Quantity
	}
	return out
}

func TestPlan_FitsInStartBin(t *testing.T) {
	bins := layout()
	plan, fit := Plan(bins[0], bins, 15, 1)
	if fit != 15 || len(plan) != 1 || plan[0].Bin.Code != "A-01" {
		t.Fatalf("plan = %v, fit = %d", placed(plan), fit)
	}
}

func TestPlan_SpillsToNearestBinsInOrder(t *testing.T) {
	bins := layout()
	plan, fit := Plan(bins[0], bins, 60, 1)
	got := placed(plan)
	if fit != 60 || got["A-01"] != 15 || got["A-02"] != 40 || got["B-01"] != 5 || len(got) != 3 {
		t.Fatalf("plan = %v, fit = %d; want A-01 15, A-02 40, B-01 5", got, fit)
	}
	if plan[1].Bin.Code != "A-02" || plan[1].Distance != 6 {
		t.Errorf("second bin = %s at %.2f m, want A-02 at 6 m", plan[1].Bin.Code, plan[1].Distance)
	}
}

func TestPlan_SkipsBinsThatAreFullOrManualOnly(t *testing.T) {
	bins := layout()
	bins[1].Used = 60
	plan, _ := Plan(bins[0], bins, 30, 1)
	got := placed(plan)
	if _, ok := got["A-02"]; ok {
		t.Error("full bin A-02 should be skipped")
	}
	if _, ok := got["DOCK"]; ok {
		t.Error("DOCK is not auto-placed")
	}
	if got["B-01"] != 15 {
		t.Errorf("plan = %v, want the 15 left over in B-01", got)
	}
}

func TestPlan_ReportsHowMuchFitsWhenSpaceRunsOut(t *testing.T) {
	bins := layout()
	_, fit := Plan(bins[0], bins, 1000, 1)
	if fit != 15+40+90+10 {
		t.Errorf("fit = %d, want 155", fit)
	}
}

func TestPlan_ChosenManualBinAndUnlimitedCapacity(t *testing.T) {
	bins := layout()
	plan, fit := Plan(bins[4], bins, 500, 1)
	if fit != 500 || len(plan) != 1 || plan[0].Bin.Code != "DOCK" {
		t.Fatalf("an unlimited bin chosen by hand should take everything; plan = %v", placed(plan))
	}
}

func TestPlan_TiesGoToBinCode(t *testing.T) {
	start := Bin{ID: 1, Code: "M", Position: Point{}, Capacity: capacity(1), AutoPlace: true}
	bins := []Bin{
		start,
		{ID: 2, Code: "Z", Position: Point{X: 3}, AutoPlace: true},
		{ID: 3, Code: "B", Position: Point{X: -3}, AutoPlace: true},
	}
	plan, _ := Plan(start, bins, 5, 1)
	if plan[1].Bin.Code != "B" {
		t.Errorf("tie at 3 m should go to B, got %s", plan[1].Bin.Code)
	}
}

func TestChooseStart_PrefersBinWithMostOfTheProduct(t *testing.T) {
	bins := layout()
	bins[2].ProductQuantity = 10
	bins[1].ProductQuantity = 5
	bin, reason, ok := ChooseStart(bins, Point{}, 1)
	if !ok || bin.Code != "B-01" || reason != ReasonProductBin {
		t.Errorf("got %s (%s)", bin.Code, reason)
	}
}

func TestChooseStart_IgnoresProductBinWithoutSpace(t *testing.T) {
	bins := layout()
	bins[3].ProductQuantity = 70
	bins[3].Used = 80
	bins[1].ProductQuantity = 5
	bin, _, _ := ChooseStart(bins, Point{}, 1)
	if bin.Code != "A-02" {
		t.Errorf("full B-02 should be skipped; got %s", bin.Code)
	}
}

func TestChooseStart_NewProductStartsNearestTheEntrance(t *testing.T) {
	bin, reason, ok := ChooseStart(layout(), Point{}, 1)
	if !ok || bin.Code != "A-01" || reason != ReasonNearestEntry {
		t.Errorf("got %s (%s), want A-01 nearest to (0,0) excluding DOCK", bin.Code, reason)
	}
}

func TestChooseStart_NoSpaceAnywhere(t *testing.T) {
	bins := layout()
	for i := range bins {
		bins[i].AutoPlace = bins[i].Capacity != nil
		if bins[i].Capacity != nil {
			bins[i].Used = *bins[i].Capacity
		}
	}
	if _, _, ok := ChooseStart(bins, Point{}, 1); ok {
		t.Error("expected no starting bin when every bin is full")
	}
}

func TestPlan_PiecesAreNeverSplit(t *testing.T) {
	start := Bin{ID: 1, Code: "A", Capacity: capacity(100), Used: 5, AutoPlace: true}
	next := Bin{ID: 2, Code: "B", Position: Point{X: 1}, AutoPlace: true}
	plan, fit := Plan(start, []Bin{start, next}, 300, 10)
	got := placed(plan)
	if fit != 300 || got["A"] != 90 || got["B"] != 210 {
		t.Fatalf("plan = %v, want 90 (nine whole pieces) in A and the rest in B", got)
	}
}

func TestChooseStart_SkipsBinsWithLessThanOnePiece(t *testing.T) {
	almostFull := Bin{ID: 1, Code: "A", Capacity: capacity(100), Used: 95, ProductQuantity: 50, AutoPlace: true}
	roomy := Bin{ID: 2, Code: "B", Position: Point{X: 9}, Capacity: capacity(100), AutoPlace: true}
	bin, _, ok := ChooseStart([]Bin{almostFull, roomy}, Point{}, 10)
	if !ok || bin.Code != "B" {
		t.Errorf("got %s; a bin with half a piece of space cannot start a pcs receipt", bin.Code)
	}
}
