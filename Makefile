LANG=en_US.UTF-8
SHELL=/bin/bash
.SHELLFLAGS=--norc --noprofile -e -u -o pipefail -c

build:
    # Use tabs for indentation
		go build -o bin/gollama cmd/main.go

clean:
		rm -rf bin/gollama

run: build
		bin/gollama