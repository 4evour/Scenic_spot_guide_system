.PHONY: check test build frontend-check frontend-build encoding

check: encoding test frontend-check

encoding:
	cd web-vue && npm run check:encoding

test:
	go test ./...

frontend-check:
	cd web-vue && npm run check

frontend-build:
	cd web-vue && npm run build

build:
	go build ./...
