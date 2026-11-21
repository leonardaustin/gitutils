MODULES = gitcloneall gitpullall goget

.PHONY: lint
lint:
	@set -e; for module in $(MODULES); do \
		(cd "$$module" && golangci-lint run ./...); \
	done
