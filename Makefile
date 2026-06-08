swagger:
	scripts/swagger.sh

build-embedded:
	cd frontend && pnpm build:embed
	rm -rf internal/router/dist
	cp -R frontend/out internal/router/dist
	go build -tags embed_frontend -o bin/wavelet main.go
