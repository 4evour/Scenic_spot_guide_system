.PHONY: check test build frontend-check frontend-build encoding secrets

check: encoding secrets test frontend-check

encoding:
	cd web-vue && npm run check:encoding

secrets:
	node scripts/check-secrets.mjs

test:
	go test ./...

frontend-check:
	cd web-vue && npm run check

frontend-build:
	cd web-vue && npm run build

build:
	go build ./...
