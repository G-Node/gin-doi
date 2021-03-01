# BUILDER IMAGE
FROM golang:alpine AS binbuilder

RUN apk add --no-cache git openssh ca-certificates curl musl-dev openssh make gcc

# Download git-annex to builder image and extract
RUN mkdir /git-annex
RUN curl -Lo /git-annex/git-annex-standalone-amd64.tar.gz https://downloads.kitenet.net/git-annex/linux/current/git-annex-standalone-amd64.tar.gz
RUN cd /git-annex && tar -xzf git-annex-standalone-amd64.tar.gz && rm git-annex-standalone-amd64.tar.gz

RUN go version
COPY . /gindoid
WORKDIR /gindoid

RUN make

### ============================= ###

# RUNNER IMAGE
FROM alpine:latest

# Update certificates inside runner container
RUN apk add --no-cache git openssh ca-certificates

# Copy git-annex from builder image
COPY --from=binbuilder /git-annex /git-annex
ENV PATH="${PATH}:/git-annex/git-annex.linux"

COPY ./assets /assets
COPY --from=binbuilder /gindoid/build/gindoid /
VOLUME ["/doidata"]
VOLUME ["/config"]
VOLUME ["/doiprep"]

EXPOSE 10443
# ENTRYPOINT /gindoid start
ADD docker_startup.sh .
RUN chmod +x ./docker_startup.sh
ENTRYPOINT ["./docker_startup.sh"]
