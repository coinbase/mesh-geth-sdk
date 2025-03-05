.PHONY: deps test mocks lint format check-license add-license \
		shorten-lines salus check-format

GO_PACKAGES=./services/... ./client/... ./configuration/... ./utils/... ./examples/... ./contracts/... ./types/...
TEST_SCRIPT=go test ${GO_PACKAGES}
LINT_CONFIG=.golangci.yml
GOIMPORTS_INSTALL=go install golang.org/x/tools/cmd/goimports@latest
GOIMPORTS_CMD=goimports
LINT_SETTINGS=golint,misspell,gocyclo,gocritic,whitespace,goconst,gocognit,bodyclose,unconvert,lll,unparam
ADDLICENSE_INSTALL=go install github.com/google/addlicense@latest
ADDLICENSE_CMD=addlicense
ADDLICENSE_IGNORE=-ignore ".github/**/*" -ignore ".idea/**/*" -ignore .codeflow.yml -ignore salus.yaml -ignore "examples/ethereum/*/*/*"
ADDLICENCE_SCRIPT=${ADDLICENSE_CMD} -c "Coinbase, Inc." -l "apache" -v ${ADDLICENSE_IGNORE}
GOLINES_INSTALL=go install github.com/segmentio/golines@latest
GOLINES_CMD=golines

deps:
	go get ./...

build:
	go build ${GO_PACKAGES}

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

fix-imports:
	${GOIMPORTS_INSTALL}
	${GOIMPORTS_CMD} -w .

add-license:
	${ADDLICENSE_INSTALL}
	${ADDLICENCE_SCRIPT} .

check-license:
	${ADDLICENSE_INSTALL}
	${ADDLICENCE_SCRIPT} -check .

shorten-lines:
	${GOLINES_INSTALL}
	${GOLINES_CMD} -w --shorten-comments configuration client examples services types utils

salus:
	docker run --rm -t -v ${PWD}:/home/repo coinbase/salus

check-format:
	! gofmt -s -l . | read;
	${GOIMPORTS_INSTALL}
	! ${GOIMPORTS_CMD} -l . | read;
