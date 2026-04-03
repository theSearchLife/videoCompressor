default:
    @just --list

uid := `id -u`
gid := `id -g`
image := "ghcr.io/thesearchlife/videocompressor:main"

# Build the image locally (for development)
build:
    docker build -t {{image}} .

# Run tests (inside Docker)
test:
    docker run --rm -v $(pwd):/src -w /src golang:1.24-alpine go test ./...

# Run compress mode with test data
run *ARGS:
    docker run -it --rm --user {{uid}}:{{gid}} -v $(pwd)/testdata:/videos {{image}} /videos {{ARGS}}

# Run cleanup mode with test data
cleanup *ARGS:
    docker run -it --rm --user {{uid}}:{{gid}} -v $(pwd)/testdata:/videos {{image}} cleanup /videos {{ARGS}}

# Run assessment on test samples
assess *ARGS:
    docker run --rm --user {{uid}}:{{gid}} -v $(pwd)/testdata/samples:/samples:ro -v $(pwd)/comparison_reports:/reports {{image}} assess /samples --output /reports {{ARGS}}

# Build and run assessment in one step
assess-local *ARGS:
    just build && just assess {{ARGS}}

# Shell into the container for debugging
shell:
    docker run -it --rm --user {{uid}}:{{gid}} -v $(pwd)/testdata:/videos --entrypoint sh {{image}}

# Clean comparison report outputs
clean-reports:
    docker run --rm -v $(pwd)/comparison_reports:/reports alpine sh -c "rm -rf /reports/*/encoded/"

# Format Go files inside Docker
fmt:
    docker run --rm -v $(pwd):/src -w /src golang:1.24 sh -lc 'find . -name "*.go" -print0 | xargs -0 /usr/local/go/bin/gofmt -w'

# Run a basic lint pass inside Docker
lint:
    docker run --rm -v $(pwd):/src -w /src golang:1.24-alpine go vet ./...

# Run e2e dev tests (comprehensive, uses local or pulled image)
e2e-dev:
    bash tests/e2e/run.sh dev

# Run e2e post-build smoke tests (verify published image)
e2e-post-build:
    bash tests/e2e/run.sh post-build

# Run all e2e tests
e2e:
    bash tests/e2e/run.sh all
