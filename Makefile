.PHONY: install-hooks test-race
install-hooks:
	git config core.hooksPath .githooks
	chmod +x .githooks/post-commit

# Run all tests with the race detector enabled.
# All concurrent code must pass this target before being merged.
test-race:
	go test -race ./...