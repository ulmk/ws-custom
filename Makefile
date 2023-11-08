include .env


BINARY_NAME=$(shell basename "$(PWD)")

# GOBASE=$(shell pwd)
# GOPATH="$(GOBASE)/vendor:$(GOBASE)"
# GOBIN=$(GOBASE)/bin
# GOFILES=$(wildcard *.go)

build:
	go build -o ${BINARY_NAME} main.go protocol.go

run: build
	./${BINARY_NAME}