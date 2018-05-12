package events

// ByDate sorts events by their start time.
type ByDate []*Event

func (a ByDate) Len() int      { return len(a) }
func (a ByDate) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByDate) Less(i, j int) bool {
	return a[i].StartTime.Before(a[j].StartTime) || a[i].ID < a[j].ID
}

// ByName sorts RSVPs by name.
type ByName []RSVP

func (a ByName) Len() int      { return len(a) }
func (a ByName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool {
	var leftName, rightName string
	if a[i].Contact.Name() != "" {
		leftName = a[i].Contact.Name()
	} else {
		leftName = a[i].Name
	}

	if a[j].Contact.Name() != "" {
		rightName = a[j].Contact.Name()
	} else {
		rightName = a[j].Name
	}

	return leftName < rightName
}
