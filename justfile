#!/usr/bin/env just -f

# This help
@help:
    just -l -u

# Init go development
[group('golang')]
go-init:
  #!/usr/bin/env bash
  if [ ! -f go.mod ]; then
    go mod init github.com/badele/godown
    go mod tidy
  fi

# build project
[group('golang')]
@go-build: go-init
  go build

# test project
[group('golang')]
@go-test:
  go test

# install precommit hooks
[group('precommit')]
@precommit-install:
  pre-commit install
  pre-commit install --hook-type commit-msg

# test precommit hooks
[group('precommit')]
precommit-test:
  pre-commit run --all-files

# Nix build project
[group('nix')]
@nix-build:
  nix build

# Run the simple markdown web server
@serve:
  go run main.go

# Build Docker image
[group('docker')]
@docker-build:
  docker build -t badele/godown:latest .

# Push Docker image to registry
[group('docker')]
@docker-push:
  docker push badele/godown:latest

# Run Docker container
[group('docker')]
@docker-run PORT="8080": docker-build
  docker run -v .:/docs -p {{PORT}}:{{PORT}} badele/godown:latest --port {{PORT}}
