FROM golang:alpine

RUN mkdir /git-annex
ENV PATH="${PATH}:/git-annex/git-annex.linux"
RUN apk add --no-cache git openssh curl
RUN curl -Lo /git-annex/git-annex-standalone-amd64.tar.gz https://downloads.kitenet.net/git-annex/linux/current/git-annex-standalone-amd64.tar.gz
RUN cd /git-annex && tar -xzf git-annex-standalone-amd64.tar.gz && rm git-annex-standalone-amd64.tar.gz
RUN apk del --no-cache curl

RUN go version
RUN go get gopkg.in/yaml.v2
RUN go get github.com/sirupsen/logrus
RUN go get github.com/docopt/docopt-go
RUN go get github.com/G-Node/gin-core/gin
RUN go get golang.org/x/crypto/ssh
RUN go get github.com/gogits/go-gogs-client
RUN go get github.com/G-Node/libgin/libgin
RUN go get github.com/G-Node/gin-cli

COPY ./cmd/gindoid /gindoid
COPY ./tmpl /tmpl
WORKDIR /gindoid
RUN go build

VOLUME ["/doidata"]
VOLUME ["/gindoid/config"]

ENTRYPOINT ./gindoid --debug
EXPOSE 10443
