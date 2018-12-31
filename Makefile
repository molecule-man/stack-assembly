.PHONY: vendor

GO111MODULE := on
export GO111MODULE

build:
	go build -o bin/stas cmd/main.go

test:
	go test ./...

testrace:
	go test -race ./...

run-acctest:
	go test -tags acceptance -v ./tests $(GODOG_ARGS)

testacc: clean-testcache
testacc: build
testacc: run-acctest
testacc: cleanup

testaccwip: GODOG_ARGS = --godog.tags=wip --godog.concurrency=1 --godog.format=pretty
testaccwip: testacc

testaccshort: GODOG_ARGS = --godog.tags=short
testaccshort: testacc

clean-testcache:
	go clean -testcache ./...

test-nocache: clean-testcache
test-nocache: test

cleanup:
	aws cloudformation describe-stacks \
		| jq '.Stacks[] | select(.Tags[].Key == "STAS_TEST") | .StackId' -r \
		| xargs -r -l aws cloudformation delete-stack --stack-name
	aws cloudformation describe-stacks \
		| jq '.Stacks[] | select(.StackName | startswith("stastest-")) | .StackId' -r \
		| xargs -r -l aws cloudformation delete-stack --stack-name

exec:
	go run cmd/*.go sync -c Stack-assembly.toml -c tpls/cfg.toml

info:
	go run cmd/*.go info -c Stack-assembly.toml -c tpls/cfg.toml

lint:
	golangci-lint run

vendor:
	rm -rf vendor
	go mod vendor
