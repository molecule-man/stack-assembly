.PHONY: vendor

GO111MODULE := on
export GO111MODULE

test:
	go test ./...

clean-testcache:
	go clean -testcache ./...

test-nocache: clean-testcache
test-nocache: test

exec:
	go run cmd/*.go sync -f Stack-assembly.toml -f tpls/cfg.toml

info:
	go run cmd/*.go info -f Stack-assembly.toml -f tpls/cfg.toml

lint:
	golangci-lint run

vendor:
	rm -rf vendor
	go mod vendor
