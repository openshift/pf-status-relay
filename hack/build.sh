#!/usr/bin/env bash

set -o errexit

eval $(go env | grep -e "GOHOSTOS" -e "GOHOSTARCH")

GOOS=${GOOS:-${GOHOSTOS}}
GOARCH=${GOARCH:-${GOHOSTARCH}}

GOFLAGS=${GOFLAGS:"-buildmode=pie" "-trimpath"}
LDFLAGS=${LDFLAGS:"-s -w"}

CGO_ENABLED=0 GO111MODULE=on GOOS=${GOOS} GOARCH=${GOARCH} go build ${GOFLAGS} -ldflags "${LDFLAGS}" -o bin/pf-status-relay cmd/pf-status-relay.go
