package events

// Index maps URL fragments to event IDs.
type Index struct {
	Fragments map[string]int `json:"fragments"`
}

// GetIndex loads the index DB, or rebuilds it if not found.
func GetIndex() (*Index, error) {
	if !DB.Exists("events/index") {
		index, err := RebuildIndex()
		return index, err
	}

	idx := &Index{}
	err := DB.Get("events/index", &idx)
	return idx, err
}

// RebuildIndex builds the event index from scratch.
func RebuildIndex() (*Index, error) {
	idx := &Index{
		Fragments: map[string]int{},
	}

	events, _ := DB.List("events/by-id")
	for _, doc := range events {
		ev := &Event{}
		err := DB.Get(doc, &ev)
		if err != nil {
			return nil, err
		}

		idx.Update(ev)
	}

	return idx, nil
}

// UpdateIndex updates the index with an event.
func UpdateIndex(event *Event) error {
	idx, err := GetIndex()
	if err != nil {
		return err
	}

	return idx.Update(event)
}

// Update an event in the index.
func (idx *Index) Update(event *Event) error {
	idx.Fragments[event.Fragment] = event.ID
	return DB.Commit("events/index", idx)
}

// Delete an event from the index.
func (idx *Index) Delete(event *Event) error {
	delete(idx.Fragments, event.Fragment)
	return DB.Commit("events/index", idx)
}
