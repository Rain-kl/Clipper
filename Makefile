.PHONY: swagger license license-check build-embedded

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
	cd frontend && npx eslint . --max-warnings 0
