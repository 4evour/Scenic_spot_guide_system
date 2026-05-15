.PHONY: check test build frontend-check frontend-build encoding secrets rag-eval demo-seed

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

rag-eval:
	go run ./cmd/rag-eval -eval knowledge/lingshan_eval_qa.json -knowledge knowledge/lingshan_chunks.jsonl

demo-seed:
	go run ./cmd/demo-seed
