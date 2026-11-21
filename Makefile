MODULES = gitcloneall gitpullall goget

.PHONY: lint vulncheck
lint:
	@set -e; for module in $(MODULES); do \
		(cd "$$module" && golangci-lint run ./...); \
	done

vulncheck:
	@set -e; for module in $(MODULES); do \
		(cd "$$module" && GOFLAGS= go run golang.org/x/vuln/cmd/govulncheck@latest ./...); \
	done

.PHONY: hooks
hooks:
	git config core.hooksPath .githooks
