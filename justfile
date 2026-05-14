default:
    @just --list

uid := `id -u`
gid := `id -g`
image := "ghcr.io/thesearchlife/videocompressor:main"
test_image := "vc:unit-test"
runtime_image := "vc:runtime"

# Build the image locally (for development)
build:
    docker build --target runtime -t {{image}} .

# Build the local runtime image used by Docker-only development tasks
build-runtime:
    docker build --target runtime -t {{runtime_image}} .

# Build the project-defined test image
build-test:
    docker build --target unit-test -t {{test_image}} .

# Run tests (inside Docker)
test:
    just build-test
    docker run --rm {{test_image}} go test ./...

# Run the complete Docker-only verification workflow
verify:
    scripts/verify-docker.zsh

# Run slow real-media S-Log3 delivery validation
verify-delivery:
    scripts/verify-delivery.zsh

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
    just build-test
    docker run --rm -v $(pwd):/src -w /src {{test_image}} sh -lc 'find . -name "*.go" -print0 | xargs -0 /usr/local/go/bin/gofmt -w'

# Run a basic lint pass inside Docker
lint:
    just build-test
    docker run --rm {{test_image}} go vet ./...

# Run e2e dev tests (comprehensive, uses local or pulled image)
e2e-dev:
    just build-runtime
    VC_IMAGE={{runtime_image}} bash tests/e2e/run.sh dev

# Run e2e post-build smoke tests (verify published image)
e2e-post-build:
    VC_IMAGE={{image}} bash tests/e2e/run.sh post-build

# Run all e2e tests
e2e:
    just build-runtime
    VC_IMAGE={{runtime_image}} bash tests/e2e/run.sh all
