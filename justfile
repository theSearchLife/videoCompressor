default:
    @just --list

uid := `id -u`
gid := `id -g`

# Build the binary (inside Docker)
build:
    docker build -t vc .

# Run tests (inside Docker)
test:
    docker run --rm -v $(pwd):/src -w /src golang:1.24-alpine go test ./...

# Run compress mode with test data
run *ARGS:
    docker run -it --rm --user {{uid}}:{{gid}} -v $(pwd)/testdata:/videos vc /videos {{ARGS}}

# Run cleanup mode with test data
cleanup *ARGS:
    docker run -it --rm --user {{uid}}:{{gid}} -v $(pwd)/testdata:/videos vc cleanup /videos {{ARGS}}

# Run assessment on test samples
assess *ARGS:
    docker run --rm --user {{uid}}:{{gid}} -v $(pwd)/testdata/samples:/samples:ro -v $(pwd)/comparison_reports:/reports vc assess /samples --output /reports {{ARGS}}

# Build and run assessment in one step
assess-local *ARGS:
    just build && just assess {{ARGS}}

# Shell into the container for debugging
shell:
    docker run -it --rm --user {{uid}}:{{gid}} -v $(pwd)/testdata:/videos --entrypoint sh vc

# Clean comparison report outputs
clean-reports:
    docker run --rm -v $(pwd)/comparison_reports:/reports alpine sh -c "rm -rf /reports/*/encoded/"

# Format Go files inside Docker
fmt:
    docker run --rm -v $(pwd):/src -w /src golang:1.24 sh -lc 'find . -name "*.go" -print0 | xargs -0 /usr/local/go/bin/gofmt -w'

# Run a basic lint pass inside Docker
lint:
    docker run --rm -v $(pwd):/src -w /src golang:1.24-alpine go vet ./...
