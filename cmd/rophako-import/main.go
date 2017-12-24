// rophako-import: import the JSON DB from the Rophako CMS to the format
// used by the Go blog.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kirsle/blog/core/jsondb"
	"github.com/kirsle/blog/core/models/comments"
	"github.com/kirsle/blog/core/models/posts"
	"github.com/kirsle/golog"
)

var (
	inPath  string
	outPath string

	log   *golog.Logger
	inDB  *jsondb.DB
	outDB *jsondb.DB

	urlSafe  = regexp.MustCompile(`[^A-Za-z0-9/]`)
	wikiLink = regexp.MustCompile(`\[\[(.+?)\]\]`)
)

func init() {
	flag.StringVar(&inPath, "in", "", "Input path: your Rophako JsonDB root")
	flag.StringVar(&outPath, "out", "", "Output path: your Blog web root")

	log = golog.GetLogger("rophako-import")
	log.Configure(&golog.Config{
		Theme:  golog.DarkTheme,
		Colors: golog.ExtendedColor,
		Level:  golog.DebugLevel,
	})
}

func main() {
	flag.Parse()
	if inPath == "" || outPath == "" {
		log.Error("Usage: rophako-import -in /opt/rophako/db -out /path/to/blog/root")
		os.Exit(1)
	} else if strings.Contains(outPath, "/.private") {
		log.Error("Do not provide the /.private suffix to -out, only the parent web root")
		os.Exit(1)
	}

	inDB = jsondb.New(inPath)
	outDB = jsondb.New(strings.TrimSuffix(filepath.Join(outPath, ".private"), "/"))
	fmt.Printf(
		"Importing Rophako DB from: %s\n"+
			"Writing output JsonDB to: %s\n"+
			"OK to continue? [yN] ",
		inDB.Root,
		outDB.Root,
	)

	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	if !strings.HasPrefix(strings.ToLower(answer), "y") {
		fmt.Println("Exiting")
		os.Exit(1)
	}

	// Migrate everything over.
	migrateBlog()
	migrateComments()
	migrateWiki()
}

func migrateBlog() {
	log.Warn("Migrating blog entries...")
	log.Info("Note: all entries will be owned by the admin user (UID 1)")
	posts.DB = outDB

	entries, err := inDB.List("blog/entries")
	if err != nil {
		log.Error("No blog entries found: %s", err.Error())
		return
	}

	for _, doc := range entries {
		parts := strings.Split(doc, "/")
		id, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			log.Error("Blog ID not a number? %s", doc)
			continue
		}

		legacy := legacyBlog{}
		err = inDB.Get(doc, &legacy)
		if err != nil {
			log.Error("Error reading legacy blog %s: %s", doc, err)
			continue
		}

		// Convert unix times to proper times.
		time := time.Unix(int64(legacy.Time), 0)

		new := &posts.Post{
			ID:             id,
			Title:          legacy.Subject,
			Fragment:       legacy.FriendlyID,
			ContentType:    "html",
			AuthorID:       1,
			Body:           legacy.Body,
			Privacy:        legacy.Privacy,
			Sticky:         legacy.Sticky,
			EnableComments: legacy.Comments,
			Tags:           legacy.Categories,
			Created:        time,
			Updated:        time,
		}
		if legacy.Format == "markdown" {
			new.ContentType = "markdown"
		}

		log.Debug("Convert post %d: %s", new.ID, new.Title)
		err = new.Save()
		if err != nil {
			log.Error("Save error: %s", err.Error())
		}
	}
}

func migrateComments() {
	log.Warn("Migrating comments...")
	comments.DB = outDB

	// Load the mailing list
	list := comments.LoadMailingList()

	threads, err := inDB.List("comments/threads")
	if err != nil {
		log.Error("No comments found: %s", err.Error())
		return
	}

	for _, doc := range threads {
		parts := strings.Split(doc, "/")
		id := parts[len(parts)-1]

		// Convert blog-# to post-#
		if strings.HasPrefix(id, "blog-") {
			id = strings.Replace(id, "blog-", "post-", 1)
		}

		legacyThread := legacyThread{}
		err = inDB.Get(doc, &legacyThread)
		if err != nil {
			log.Error("Error reading legacy thread %s: %s", doc, err)
			continue
		}

		log.Debug("Converting comment thread: %s", id)
		t, err := comments.Load(id)
		if err != nil {
			t = comments.New(id)
		}

		for commentID, legacy := range legacyThread {
			// Convert unix times to proper times.
			time := time.Unix(int64(legacy.Time), 0)

			new := &comments.Comment{
				ID:          commentID,
				UserID:      legacy.UserID,
				Name:        legacy.Name,
				Avatar:      legacy.Image,
				Body:        legacy.Message,
				EditToken:   legacy.Token,
				DeleteToken: uuid.New().String(),
				Created:     time,
				Updated:     time,
			}
			new.LoadAvatar() // in case it has none

			log.Debug("Comment by %s on thread %s", new.Name, id)
			t.Post(new)
		}

		// Re-sort the comments by date.
		sort.Sort(comments.ByCreated(t.Comments))

		// Check for subscribers
		subs := legacySubscribers{}
		err = inDB.Get(fmt.Sprintf("comments/subscribers/%s", id), &subs)
		if err == nil {
			for email := range subs {
				log.Debug("Subscribe %s to thread %s", email, id)
				list.Subscribe(id, email)
			}
		}
	}
}

func migrateWiki() {
	log.Warn("Migrating wiki...")

	threads, err := inDB.List("wiki/pages")
	if err != nil {
		log.Error("No pages found: %s", err.Error())
		return
	}

	// Prepare the output directory.
	wikiPath := filepath.Join(outPath, "wiki")
	if _, err = os.Stat(wikiPath); os.IsNotExist(err) {
		err = os.Mkdir(wikiPath, 0755)
		if err != nil {
			log.Error("Can't create wiki root %s: %s", wikiPath, err)
			return
		}
	}

	for _, doc := range threads {
		parts := strings.Split(doc, "/")
		title := parts[len(parts)-1]

		// File name is the title but with hyphens, to match the URL version.
		filename := makeURLSafe(title)

		page := &legacyWiki{}
		err = inDB.Get(doc, &page)
		if err != nil {
			log.Error("Error reading wiki DB %s: %s", doc, err)
			continue
		}

		// Take the highest revision as the current version to use.
		var highest *legacyRevision
		for _, rev := range page.Revisions {
			if highest == nil || rev.Time > highest.Time {
				highest = &rev
			}
		}
		if highest == nil {
			log.Error("Wiki page %s has no revisions?", doc)
			continue
		}

		// Insert the title as the first <h1> in the Markdown.
		markdown := fmt.Sprintf("# %s\n\n%s", title, highest.Body)

		// Find and replace inter-wiki links with normal URIs.
		links := wikiLink.FindAllStringSubmatch(markdown, -1)
		for _, match := range links {
			href, label := match[1], match[1]
			if strings.Contains(label, "|") {
				parts := strings.SplitN(label, "|", 2)
				label, href = parts[0], parts[1]
			}
			href = "/wiki/" + makeURLSafe(href)

			markdown = strings.Replace(
				markdown,
				match[0],
				fmt.Sprintf("[%s](%s)", label, href),
				1,
			)
		}

		// Write the body into Markdown files.
		path := filepath.Join(wikiPath, fmt.Sprintf("%s.md", filename))
		log.Debug("Writing page '%s' to: %s", title, path)
		err = ioutil.WriteFile(path, []byte(markdown), 0644)
		if err != nil {
			log.Error("Error writing: %s", err)
		}
	}
}

func makeURLSafe(input string) string {
	return strings.Trim(
		strings.Replace(
			urlSafe.ReplaceAllString(input, "-"),
			"--",
			"-",
			0,
		),
		"-",
	)
}

func commit(document string, v interface{}) {
	err := outDB.Commit(document, v)
	if err != nil {
		log.Error("Commit error: %s: %s", document, err.Error())
	}
}

type legacyBlog struct {
	Author     int      `json:"author"`
	Body       string   `json:"body"`
	Format     string   `json:"format"`
	Categories []string `json:"categories"`
	Comments   bool     `json:"comments"`
	FriendlyID string   `json:"fid"`
	Privacy    string   `json:"privacy"`
	Sticky     bool     `json:"sticky"`
	Subject    string   `json:"subject"`
	Time       float64  `json:"time"`
}

type legacyComment struct {
	Message string  `json:"message"`
	Token   string  `json:"token"`
	Name    string  `json:"name"`
	Image   string  `json:"image"`
	Time    float64 `json:"time"`
	UserID  int     `json:"uid"`
}

type legacyThread map[string]legacyComment

type legacySubscribers map[string]float64

type legacyWiki struct {
	Revisions []legacyRevision `json:"revisions"`
}

type legacyRevision struct {
	Body   string  `json:"body"`
	Author int     `json:"author"`
	Time   float64 `json:"time"`
	Note   string  `json:"note"`
	ID     string  `json:"id"`
}
