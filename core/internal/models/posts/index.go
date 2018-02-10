package posts

import (
	"sort"
	"strings"
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
	Posts map[int]Post `json:"posts"`
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
		Posts: map[int]Post{},
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
		ID:       p.ID,
		Title:    p.Title,
		Fragment: p.Fragment,
		AuthorID: p.AuthorID,
		Privacy:  p.Privacy,
		Tags:     p.Tags,
		Created:  p.Created,
		Updated:  p.Updated,
	}
	err := DB.Commit("blog/index", idx)
	return err
}

// Delete a blog's entry from the index.
func (idx *Index) Delete(p *Post) error {
	delete(idx.Posts, p.ID)
	return DB.Commit("blog/index", idx)
}

// Tag is a response from Tags including metadata about it.
type Tag struct {
	Name  string
	Count int
}

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

// CleanupFragments to clean up old URL fragments.
func CleanupFragments() error {
	idx, err := GetIndex()
	if err != nil {
		return err
	}
	return idx.CleanupFragments()
}

// CleanupFragments to clean up old URL fragments.
func (idx *Index) CleanupFragments() error {
	// Keep track of the active URL fragments so we can clean up orphans.
	fragments := map[string]struct{}{}
	for _, p := range idx.Posts {
		fragments[p.Fragment] = struct{}{}
	}

	// Clean up unused fragments.
	byFragment, err := DB.List("blog/fragments")
	for _, doc := range byFragment {
		parts := strings.Split(doc, "/")
		fragment := parts[len(parts)-1]
		if _, ok := fragments[fragment]; !ok {
			log.Debug("RebuildIndex() clean up old fragment '%s'", fragment)
			DB.Delete(doc)
		}
	}

	return err
}
