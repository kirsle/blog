# Dockerfile for the blog.
#
# Building:
#
#    docker build -t blog .
#
# Running:
#
#    # listen on localhost:8000 and use /home/user/www as the user root
#    docker run -p 8000:80 -v /home/user/www:/data/www blog
#
# Running and Backgrounding:
#
#    # run it with a name to start with
#    docker run -d --name blog -v /home/user/www:/data/www blog
#
#    # later...
#    docker start blog
FROM fedora:latest

RUN dnf -y update
RUN dnf -y install golang make

WORKDIR /go/src/github.com/kirsle/blog
ADD . /go/src/github.com/kirsle/blog

ENV GOPATH /go
RUN go get ./...
RUN make build

EXPOSE 80
CMD ["/go/src/github.com/kirsle/blog/bin/blog", "-a", ":80", "/data/www"]
