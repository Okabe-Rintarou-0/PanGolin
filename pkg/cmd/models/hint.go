package models

type HintEntry interface {
	DisplayValue() string
	RealValue() string
	Compare(HintEntry) int
}

type HintEntries []HintEntry

func (e HintEntries) Len() int      { return len(e) }
func (e HintEntries) Swap(i, j int) { e[i], e[j] = e[j], e[i] }
func (e HintEntries) Less(i, j int) bool {
	return e[i].Compare(e[j]) < 0
}
