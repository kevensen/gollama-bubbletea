LANG=en_US.UTF-8
SHELL=/bin/bash
.SHELLFLAGS=--norc --noprofile -e -u -o pipefail -c

build:
    # Use tabs for indentation
		go build -o bin/gollama cmd/main.go

clean:
		rm -rf bin/gollama internal/tools/motd/plugin.wasm

run: build
		bin/gollama -ollama_host=http://127.0.0.1 -ollama_port=11434