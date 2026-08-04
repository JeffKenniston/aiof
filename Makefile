# AIOF Enterprise Monorepo Makefile v0.2.0

.PHONY: all build test clean generate dev lint

all: build

build:
	cd apps/daemon && go build -o ../../bin/aiof ./cmd/aiof/main.go
	pnpm --filter "@aiof/*" build

test:
	cd apps/daemon && go test -v ./...

generate:
	cd packages/contracts && buf generate

dev:
	./run.sh

clean:
	rm -rf bin/ dist/ apps/client/dist apps/web/dist
