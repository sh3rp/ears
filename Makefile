all: build

build:
	glide install
	go install cmd/ears.go

.PHONY: build
