.PHONY: swagger license license-check build-embedded build-test

swagger:
	scripts/swagger.sh

license:
	scripts/update_go_license.sh

license-check:
	scripts/update_go_license.sh --check

build-embedded:
	cd frontend && pnpm build:embed
	rm -rf internal/router/dist
	cp -R frontend/out internal/router/dist
	go build -tags embed_frontend -o bin/wavelet main.go

code-check:
	cd frontend && pnpm tsc --noEmit --jsx preserve && npx eslint . --max-warnings 0
	golangci-lint run

build-test:
	@echo "==> Running frontend and backend build tests in parallel..."
	@PIDS=""; \
	STATUS=0; \
	( cd frontend && pnpm build 2>&1 | sed 's/^/[frontend] /' ) & PIDS="$$PIDS $$!"; \
	( go build -o /dev/null ./... 2>&1 | sed 's/^/[backend]  /' ) & PIDS="$$PIDS $$!"; \
	for PID in $$PIDS; do \
		wait $$PID || STATUS=1; \
	done; \
	if [ $$STATUS -eq 0 ]; then \
		echo "==> All build tests passed."; \
	else \
		echo "==> Build test FAILED." >&2; \
		exit 1; \
	fi
