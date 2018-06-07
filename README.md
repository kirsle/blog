# Blog

This is a web blog and personal homepage engine written in Go. It includes a
full-featured web blog (with tags, archive views, etc.) and can serve static
web assets, Go HTML templates and Markdown pages.

# Features

## Zero Configuration

Blog is designed to be extremely easy to run: just give it a path to your
website's document root. It doesn't have to exist: it can be created as needed!

```
blog $HOME/www
```

See `blog -h` for command line arguments, for example to make it listen on a
different port number.

The blog database is kept on disk as JSON files under the document root.

## Dual Template System

Whenever a web request is handled by the Blog program, it checks your
user-defined Document Root to serve the file before falling back on its
built-in store of "Core Files."

This way, you can copy and override files from the Core Root by creating files
with the same names in your Document Root. If you override `.layout.gohtml`,
you can customize the overall web design of your site. You can also
override individual templates to customize the look of built-in pages, such as
how blog entries are formatted.

The Blog has a built-in page editor that makes it easy to copy and override
the core template files.

## Render Go HTML and Markdown Files

You can write pages in the Go [html/template](https://golang.org/pkg/html/template/)
format (`.gohtml`) or in GitHub Flavored Markdown (`.md`). Markdown pages will
be rendered to HTML and inserted into your web design layout like normal pages.

## Built-in Page Editor

A built-in editor lists all of the pages in your Document Root and the Core
Root. You can click on a page to edit it, with the
[ACE Code Editor](https://ace.c9.io/) offering a rich code editing experience,
with syntax highlighting for HTML, Markdown, JavaScript and CSS.

Editing a Core Page and saving it will save an override version in your
Document Root.

In the default Blog theme, every page includes a link to edit the page
in the Page Editor for logged-in users. The 404 Error handler also
provides shortcuts to create a new page at that path.

# Setup

```bash
# If you're new to Go, set your GOPATH.
export GOPATH="${HOME}/go"

# Clone this repository to your go src folder.
git clone https://github.com/kirsle/blog ~/go/src/github.com/kirsle/blog
cd ~/go/src/github.com/kirsle/blog

# Run the server
make run

# Or to run it manually with go-reload to provide custom options:
./go-reload cmd/blog/main.go [options] [/path/to/document/root]
```

## Docker

This app includes a Dockerfile. Type `make docker.build` to build the
Docker image.

### Quick Start

```bash
make docker.build
make docker.run
```

### Docker Image

* Exposes port 80 for the web server
* User document root is mounted at `/data/www`

So to run the Docker image and have it listen on `localhost:8000` on the
host and bind the user document root to `/home/user/www`:

```bash
docker run -p 8000:80 -v /home/user/www:/data/www blog
```

You may also run `make docker.run` to run the Docker image on port 8000 using
the `./user-root` directory

## Recommendation: Redis

It is recommended to use the [Redis](https://redis.io) caching server in
conjunction with Blog. Redis helps boost the performance in two main areas:

The JSON database system will cache the JSON documents in Redis, speeding up
access time because the filesystem doesn't need to be read each time. If you
manually modify a JSON file on disk, it _will_ notice and re-read it next time
it's requested.

If you make use of source code blocks in GitHub Flavored Markdown, the Python
`pygmentize` command can render syntax-highlighted HTML from it. Calling this
program takes ~0.6s, and a page with many source blocks would take a _long_ time
to load. I alleviate this by MD5-hashing and Redis-caching the rendered HTML
code to minimize calls to `pygmentize`.

After you initialize the site, go to the admin settings to enable Redis.

## Syntax Highlighting with Pygments

To enable syntax highlighting within Markdown files (like with GitHub Flavored
Markdown), install [pygments](http://pygments.org) on your system. For example:

```
# Fedora/RHEL
sudo dnf install python3-pygments python3-pygments-markdown-lexer

# Debian
sudo apt install python3-pygments
```

Blog will automatically use the `pygmentize` command if it's available on its
`$PATH`.

# License

MIT.
