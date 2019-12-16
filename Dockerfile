# BUILDER IMAGE
FROM golang:alpine AS binbuilder

RUN apk add --no-cache git openssh ca-certificates curl musl-dev openssh

# Download git-annex to builder image and extract
RUN mkdir /git-annex
RUN curl -Lo /git-annex/git-annex-standalone-amd64.tar.gz https://downloads.kitenet.net/git-annex/linux/current/git-annex-standalone-amd64.tar.gz
RUN cd /git-annex && tar -xzf git-annex-standalone-amd64.tar.gz && rm git-annex-standalone-amd64.tar.gz

RUN go version
COPY ./go.mod ./go.sum /gindoid/
COPY ./vendor /gindoid/vendor/
COPY ./cmd /gindoid/cmd/
WORKDIR /gindoid

RUN go build ./cmd/gindoid

### ============================= ###

# RUNNER IMAGE
FROM alpine:latest

# Update certificates inside runner container
RUN apk add --no-cache git openssh ca-certificates

# Copy git-annex from builder image
COPY --from=binbuilder /git-annex /git-annex
ENV PATH="${PATH}:/git-annex/git-annex.linux"

COPY ./assets /assets
COPY --from=binbuilder /gindoid/gindoid /
VOLUME ["/doidata"]
VOLUME ["/config"]

ENTRYPOINT /gindoid
EXPOSE 10443
