.PHONY: all build clean test install

# Build the zelang compiler
all: build

build:
	@echo "Building ZeLang compiler..."
	cd cmd/zelang && go build -o ../../zelang
	@echo "✓ Compiler built: ./zelang"

# Clean build artifacts
clean:
	rm -f zelang
	rm -f examples/*.c
	rm -f examples/simple
	rm -f examples/*.db

# Test the compiler with the simple example
test: build
	@echo "Testing with simple.zl..."
	./zelang build examples/simple.zl
	@echo "Running compiled binary..."
	cd examples && ./simple
	@echo "✓ Test passed"

# Install to /usr/local/bin
install: build
	cp zelang /usr/local/bin/
	@echo "✓ Installed to /usr/local/bin/zelang"

# Show help
help:
	@echo "ZeLang Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  make         - Build the compiler"
	@echo "  make test    - Build and test with examples"
	@echo "  make clean   - Remove build artifacts"
	@echo "  make install - Install to /usr/local/bin"
	@echo "  make help    - Show this help"
