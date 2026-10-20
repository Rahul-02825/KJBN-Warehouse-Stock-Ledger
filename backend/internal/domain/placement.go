package domain

import (
	"container/heap"
	"math"
)

const (
	ReasonChosen       = "chosen by you"
	ReasonProductBin   = "holds the most of this product"
	ReasonNearestEntry = "nearest to the receiving point"
)

type Point struct {
	X, Y float64
}

func Distance(a, b Point) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}

type Bin struct {
	ID              int64
	Code            string
	Position        Point
	Capacity        *int64
	Used            int64
	ProductQuantity int64
	AutoPlace       bool
}

func (b Bin) Free() int64 {
	if b.Capacity == nil {
		return math.MaxInt64
	}
	return max(*b.Capacity-b.Used, 0)
}

type Allocation struct {
	Bin      Bin
	Quantity int64
	Distance float64
}

func ChooseStart(bins []Bin, receiving Point, step int64) (Bin, string, bool) {
	candidates := placeableBins(bins, 0, step)
	if len(candidates) == 0 {
		return Bin{}, "", false
	}
	if bin, ok := binHoldingMostProduct(candidates); ok {
		return bin, ReasonProductBin, true
	}
	return nearestBin(candidates, receiving), ReasonNearestEntry, true
}

func Plan(start Bin, bins []Bin, quantity, step int64) ([]Allocation, int64) {
	var plan []Allocation
	remaining := quantity
	nearest := newBinHeap(start.Position, placeableBins(bins, start.ID, step))

	bin := start
	for {
		amount := min(remaining, bin.Free())
		amount -= amount % step
		if amount > 0 {
			plan = append(plan, Allocation{Bin: bin, Quantity: amount, Distance: Distance(start.Position, bin.Position)})
			remaining -= amount
		}
		if remaining == 0 || nearest.Len() == 0 {
			break
		}
		bin = heap.Pop(nearest).(Bin)
	}
	return plan, quantity - remaining
}

func placeableBins(bins []Bin, excludeID, step int64) []Bin {
	var out []Bin
	for _, b := range bins {
		if b.AutoPlace && b.ID != excludeID && b.Free() >= step {
			out = append(out, b)
		}
	}
	return out
}

func binHoldingMostProduct(bins []Bin) (Bin, bool) {
	var best Bin
	found := false
	for _, b := range bins {
		if b.ProductQuantity == 0 {
			continue
		}
		if !found || b.ProductQuantity > best.ProductQuantity || (b.ProductQuantity == best.ProductQuantity && b.Code < best.Code) {
			best, found = b, true
		}
	}
	return best, found
}

func nearestBin(bins []Bin, from Point) Bin {
	h := newBinHeap(from, bins)
	return heap.Pop(h).(Bin)
}

type binHeap struct {
	origin Point
	bins   []Bin
}

func newBinHeap(origin Point, bins []Bin) *binHeap {
	h := &binHeap{origin: origin, bins: append([]Bin(nil), bins...)}
	heap.Init(h)
	return h
}

func (h binHeap) Len() int { return len(h.bins) }

func (h binHeap) Less(i, j int) bool {
	di := squaredDistance(h.origin, h.bins[i].Position)
	dj := squaredDistance(h.origin, h.bins[j].Position)
	if di != dj {
		return di < dj
	}
	return h.bins[i].Code < h.bins[j].Code
}

func (h binHeap) Swap(i, j int) { h.bins[i], h.bins[j] = h.bins[j], h.bins[i] }

func (h *binHeap) Push(x any) { h.bins = append(h.bins, x.(Bin)) }

func (h *binHeap) Pop() any {
	last := h.bins[len(h.bins)-1]
	h.bins = h.bins[:len(h.bins)-1]
	return last
}

func squaredDistance(a, b Point) float64 {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx*dx + dy*dy
}
