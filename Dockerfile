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
FROM golang:1.10

WORKDIR /go/src/github.com/kirsle/blog
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 80
CMD ["blog", "-a", ":80", "/data/www"]
