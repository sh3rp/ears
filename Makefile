all: build

build:
	glide install
	go install cmd/ears.go
	sudo chown root $$GOBIN/ears
	sudo chmod u+s $$GOBIN/ears

.PHONY: build
