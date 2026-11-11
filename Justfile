set shell := ["bash", "-u", "-c"]

# export environment variables
export GOBIN := `echo $PWD/.bin`
export GOFLAGS := '--tags=fts5'
export GOTOOLCHAIN := 'go1.27.4'
export scripts := ".github/workflows/scripts"
export memcached := env("MEMCACHED", "/tmp/mc.sock")

# print available commands
[private]
default:
    @just --list

# start with given config file
[group('build')]
run config: compile
    $GOBIN/onlylogins --config {{config}}

# compile the executable
[group('build')]
compile version="localdev": tidy
    go install -ldflags "-X cattlecloud.net/onlylogins/version.Version={{version}}"

# run specific unit test
[group('build')]
[no-cd]
test unit:
    go test -v -count=1 -race -run {{unit}} 2>/dev/null

# run tests across source tree
[group('build')]
tests:
    go test -count=1 -race ./... 2>/dev/null

# vet the source tree
[group('build')]
vet:
    go vet ./...

# lint the source tree
[group('build')]
lint: vet
    $GOBIN/golangci-lint --config $scripts/golangci.yaml run
    shellcheck $scripts/*.sh

# tidy up Go modules
[group('build')]
tidy:
    go mod tidy

# run generate commands
[group('build')]
generate:
    go generate ./...

# show host system information
[group('build')]
@sysinfo:
    echo "{{os()/arch()}} {{num_cpus()}}c"

# locally install build tools
[group('build')]
init:
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
    go install github.com/pelletier/go-toml/v2/cmd/tomljson@v2.2.4

# start memcached instance for local development
[group('memcached')]
mc-run:
    memcached --verbose --enable-shutdown --unix-socket /tmp/mc.sock

# show some basic memcached statistics
[group('memcached')]
mc-stats:
    echo "stats" | nc -U {{memcached}}

# open sqlite3 database with cli
[group('database')]
db-open config:
    #!/bin/sh -eu
    path=$($GOBIN/tomljson {{config}} | jq -r .database.path)
    sqlite3 "${path}"

# reset sqlite3 database
[group('database')]
db-reset config:
    #!/bin/sh -eu
    path=$($GOBIN/tomljson {{config}} | jq -r .database.path)
    rm -f "${path}" "${path}-wal" "${path}-shm"

# deploy artifact
[group('deploy')]
deploy host:
    HOST={{host}} envy exec gh-cattle $scripts/deploy.sh

# create release artifact
[group('deploy')]
artifact tag:
    gh workflow run build.yaml -f tag={{tag}}
