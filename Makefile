GOOS ?= linux

.PHONY:
.SILENT:
.DEFAULT_GOAL := run

build:
	go mod download && CGO_ENABLED=0 GOOS=$(GOOS) go build -o ./.bin/app ./cmd/app/main.go

run: build
	docker-compose up --remove-orphans app

debug: build
	docker-compose up --remove-orphans debug

lint:
	golangci-lint run

env:
	cp env.dist .env