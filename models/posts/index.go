package posts

import (
	"sort"
)

// UpdateIndex updates a post's metadata in the blog index.
func UpdateIndex(p *Post) error {
	idx, err := GetIndex()
	if err != nil {
		return err
	}
	return idx.Update(p)
}

// Index caches high level metadata about the blog's contents for fast access.
type Index struct {
	Posts     map[int]Post   `json:"posts"`
	Fragments map[string]int `json:"fragments"`
}

// GetIndex loads the index, or rebuilds it first if it doesn't exist.
func GetIndex() (*Index, error) {
	if !DB.Exists("blog/index") {
		index, err := RebuildIndex()
		return index, err
	}
	idx := &Index{}
	err := DB.Get("blog/index", &idx)
	return idx, err
}

// RebuildIndex builds the index from scratch.
func RebuildIndex() (*Index, error) {
	idx := &Index{
		Posts:     map[int]Post{},
		Fragments: map[string]int{},
	}
	entries, _ := DB.List("blog/posts")
	for _, doc := range entries {
		p := &Post{}
		err := DB.Get(doc, &p)
		if err != nil {
			return nil, err
		}

		idx.Update(p)
	}

	return idx, nil
}

// Update a blog's entry in the index.
func (idx *Index) Update(p *Post) error {
	idx.Posts[p.ID] = Post{
		ID:             p.ID,
		Title:          p.Title,
		Fragment:       p.Fragment,
		AuthorID:       p.AuthorID,
		Privacy:        p.Privacy,
		Sticky:         p.Sticky,
		EnableComments: p.EnableComments,
		Tags:           p.Tags,
		Created:        p.Created,
		Updated:        p.Updated,
	}
	idx.Fragments[p.Fragment] = p.ID
	err := DB.Commit("blog/index", idx)
	return err
}

// Delete a blog's entry from the index.
func (idx *Index) Delete(p *Post) error {
	delete(idx.Posts, p.ID)
	delete(idx.Fragments, p.Fragment)
	return DB.Commit("blog/index", idx)
}

// Tag is a response from Tags including metadata about it.
type Tag struct {
	Name  string
	Count int
}

// ByPopularity sort type.
type ByPopularity []Tag

func (s ByPopularity) Len() int {
	return len(s)
}
func (s ByPopularity) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByPopularity) Less(i, j int) bool {
	if s[i].Count < s[j].Count {
		return true
	} else if s[i].Count > s[j].Count {
		return false
	}
	return s[i].Name < s[j].Name
}

// Tags returns the tags sorted by most frequent.
func (idx *Index) Tags() ([]Tag, error) {
	idx, err := GetIndex()
	if err != nil {
		return nil, err
	}

	unique := map[string]*Tag{}

	for _, post := range idx.Posts {
		for _, name := range post.Tags {
			tag, ok := unique[name]
			if !ok {
				tag = &Tag{name, 0}
				unique[name] = tag
			}
			tag.Count++
		}
	}

	// Sort the tags.
	tags := []Tag{}
	for _, tag := range unique {
		tags = append(tags, *tag)
	}
	sort.Sort(sort.Reverse(ByPopularity(tags)))

	return tags, nil
}
