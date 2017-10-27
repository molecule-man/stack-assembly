GOFILES = $$(go list ./... | grep -v /vendor/)

test: preinstall
	go test ${GOFILES}

preinstall:
	go test -i -v ${GOFILES}

exec:
	go run cmd/main.go -cfg tpls/cfg.toml
