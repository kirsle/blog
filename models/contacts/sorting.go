package contacts

// ByName sorts contacts by name.
type ByName []*Contact

func (a ByName) Len() int      { return len(a) }
func (a ByName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool {
	return a[i].Name() < a[j].Name()
}
