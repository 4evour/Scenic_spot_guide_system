.PHONY: check test build frontend-check frontend-build encoding secrets rag-eval rag-bench demo-seed

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
	go run ./cmd/rag-eval -k 8 -eval knowledge/lingshan_eval_qa.json -knowledge knowledge/lingshan_chunks.jsonl

rag-bench:
	go run ./cmd/rag-eval -knowledge knowledge/lingshan_scale_3000.jsonl -eval knowledge/lingshan_eval_300.json -k 8 -bench -concurrency 16 -repeat 1 -retrieval-only -fail-on-miss

demo-seed:
	go run ./cmd/demo-seed
