FROM golang:alpine


ENV PATH="${PATH}:/tmp/git-annex.linux"
RUN apk add --no-cache git curl
RUN curl -Lo /tmp/git-annex-standalone-amd64.tar.gz https://downloads.kitenet.net/git-annex/linux/current/git-annex-standalone-amd64.tar.gz
RUN cd /tmp && tar -xzf git-annex-standalone-amd64.tar.gz && rm git-annex-standalone-amd64.tar.gz
RUN apk del --no-cache curl

RUN go version
RUN go get gopkg.in/yaml.v2
RUN go get github.com/Sirupsen/logrus
RUN go get github.com/docopt/docopt-go
RUN go get github.com/G-Node/gin-core/gin
RUN go get golang.org/x/crypto/ssh
RUN go get github.com/gogits/go-gogs-client

COPY ./cmd/gindoid /gindoid
COPY ./tmpl /tmpl
WORKDIR /gindoid
RUN go build

VOLUME ["/doidata"]
VOLUME ["/repos"]

ENTRYPOINT ./gindoid --debug --target=/doidata --templates=/tmpl --key=$tokenkey --port=10443
EXPOSE 10443
