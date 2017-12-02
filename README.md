# Blog

This is the source code that will soon power [kirsle.net](https://www.kirsle.net/).

# Setup

```bash
# If you're new to Go, set your GOPATH.
export GOPATH="${HOME}/go"

# Clone this repository to your go src folder.
git clone https://github.com/kirsle/blog ~/go/src/github.com/kirsle/blog
cd ~/go/src/github.com/kirsle/blog

# Run the server
make run
```

## Syntax Highlighting with Pygments

To enable syntax highlighting within Markdown files (like with GitHub Flavored
Markdown), install [pygments](http://pygments.org) on your system. For example:

```
# Fedora/RHEL
sudo dnf install python3-pygments python3-pygments-markdown-lexer

# Debian
sudo apt install python3-pygments
```

# License

MIT.
