.PHONY: build test check fmt vet

build:
	task build

test:
	task test

fmt:
	task format:check

vet:
	task lint

check:
	task verify
