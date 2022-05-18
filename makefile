.PHONY: deps test mocks lint format

GO_PACKAGES=./services/... ./client/... ./configuration/... ./utils/... ./examples/...
TEST_SCRIPT=go test ${GO_PACKAGES}
LINT_CONFIG=.golangci.yml
GOIMPORTS_CMD=go run golang.org/x/tools/cmd/goimports
LINT_SETTINGS=golint,misspell,gocyclo,gocritic,whitespace,goconst,gocognit,bodyclose,unconvert,lll,unparam

deps:
	go get ./...

test:
	${TEST_SCRIPT}

mocks:
	rm -rf mocks;
	mockery --dir services --all --case underscore --outpkg services --output mocks/services;
	mockery --dir client --all --case underscore --outpkg client --output mocks/client;

lint:
	golangci-lint run --timeout 2m0s -v -E ${LINT_SETTINGS},gomnd

format:
	gofmt -s -w -l .
	${GOIMPORTS_CMD} -w .
