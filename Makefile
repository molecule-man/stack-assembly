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
	go test -tags acceptance -v ./tests

run-acctest-short:
	go test -tags acceptance -v ./tests --godog.tags=short

testacc: clean-testcache
testacc: build
testacc: run-acctest
testacc: cleanup

testaccwip: clean-testcache
testaccwip: build
testaccwip:
	go test -tags acceptance -v ./tests --godog.tags=wip --godog.concurrency=1 --godog.format=pretty

testaccshort: clean-testcache
testaccshort: build
testaccshort: run-acctest-short
testaccshort: cleanup

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
	go run cmd/*.go sync -f Stack-assembly.toml -f tpls/cfg.toml

info:
	go run cmd/*.go info -f Stack-assembly.toml -f tpls/cfg.toml

lint:
	golangci-lint run

vendor:
	rm -rf vendor
	go mod vendor
