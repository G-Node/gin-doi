[![Tests](https://github.com/g-node/gin-doi/workflows/run-tests/badge.svg?branch=master)](https://github.com/G-Node/gin-doi/actions)
[![Coverage Status](https://coveralls.io/repos/github/G-Node/gin-doi/badge.svg?branch=master)](https://coveralls.io/github/G-Node/gin-doi?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/g-node/gin-doi)](https://goreportcard.com/report/github.com/g-node/gin-doi)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/g-node/gin-doi)](https://pkg.go.dev/github.com/G-Node/gin-doi)

# GIN DOI

GIN-DOI is the G-Node Infrastructure DOI service.
The service can, at the request of a repository owner, copy a public repository, pack everything into an archive file, store it in a safe location, and provide a DOI (digital object identifier) with which the archive can be cited.

Registered datasets can be found on the [Registered Datasets](https://doi.gin.g-node.org) on GIN.

For instructions on how to register a repository, see the [relevant help pages](https://gin.g-node.org/G-Node/Info/wiki/DOIfile).

GIN-DOI fulfills the [DataCite](https://www.datacite.org/) standard which (according to Wikipedia) tries to:
* Establish easier access to research data on the Internet.
* Increase acceptance of research data as legitimate, citable contributions to the scholarly record.
* Support data archiving that will permit results to be verified and re-purposed for future study.

## Dependencies

gin-doi is dependent on the [G-Node/libgin](https://github.com/G-Node/libgin) and the [G-Node/gin-cli](https://github.com/G-Node/gin-cli).

When building gin-doi from source and using a different version of `libgin` or `gin-cli` than specified in the `go.mod` file, use `go get` to fetch the latest `libgin` or `gin-cli` release or point to a specific commit in master.

As an example:
- `go get github.com/G-Node/libgin` to include the latest release
- `go get github.com/G-Node/libgin@[commit hash]` for a specifc commit in the master branch of G-Node/libgin
