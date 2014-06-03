GOPATH := $(shell pwd)/.gopath

default: test
test:
	GOPATH=$(GOPATH) go test
