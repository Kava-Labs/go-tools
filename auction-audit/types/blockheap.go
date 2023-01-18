package types

type BlockHeap []BlockData

func (h BlockHeap) Len() int { return len(h) }
func (h BlockHeap) Less(i, j int) bool {
	return h[i].Block.Header.Height < h[j].Block.Header.Height
}
func (h BlockHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *BlockHeap) Push(x interface{}) {
	*h = append(*h, x.(BlockData))
}

func (h *BlockHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
