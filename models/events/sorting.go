package events

// ByDate sorts events by their start time.
type ByDate []*Event

func (a ByDate) Len() int      { return len(a) }
func (a ByDate) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByDate) Less(i, j int) bool {
	return a[i].StartTime.Before(a[j].StartTime) || a[i].ID < a[j].ID
}
