package posts

// ByUpdated sorts blog entries by most recently updated.
type ByUpdated []Post

func (a ByUpdated) Len() int      { return len(a) }
func (a ByUpdated) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByUpdated) Less(i, j int) bool {
	return a[i].Updated.Before(a[i].Updated) || a[i].ID < a[j].ID
}
