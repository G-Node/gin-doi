FROM golang:alpine

RUN mkdir /git-annex
ENV PATH="${PATH}:/git-annex/git-annex.linux"
RUN apk add --no-cache git openssh curl
RUN curl -Lo /git-annex/git-annex-standalone-amd64.tar.gz https://downloads.kitenet.net/git-annex/linux/current/git-annex-standalone-amd64.tar.gz
RUN cd /git-annex && tar -xzf git-annex-standalone-amd64.tar.gz && rm git-annex-standalone-amd64.tar.gz
RUN apk del --no-cache curl

RUN apk add --no-cache musl-dev gcc # for building deps

RUN go version

COPY ./go.mod ./go.sum /gindoid/
COPY ./tmpl /tmpl
WORKDIR /gindoid
# download deps before bringing in the main package
RUN go mod download

COPY ./cmd /gindoid/cmd/
RUN go build ./cmd/gindoid

VOLUME ["/doidata"]
VOLUME ["/gindoid/config"]

ENTRYPOINT ./gindoid --debug
EXPOSE 10443
