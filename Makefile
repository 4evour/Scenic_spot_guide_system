.PHONY: check test build frontend-check frontend-contracts frontend-build encoding secrets compose-healthcheck rag-eval rag-bench demo-seed

check: encoding secrets compose-healthcheck test frontend-check frontend-contracts

encoding:
	cd web-vue && npm run check:encoding

secrets:
	node scripts/check-secrets.mjs

compose-healthcheck:
	node scripts/check-compose-healthcheck.mjs

test:
	go test ./...

frontend-check:
	cd web-vue && npm run check

frontend-contracts:
	cd web-vue && npm run check:data-boundaries
	cd web-vue && npm run check:admin-queries
	cd web-vue && npm run check:admin-settings-i18n
	cd web-vue && npm run check:admin-avatar-i18n
	cd web-vue && npm run check:admin-query-i18n
	cd web-vue && npm run check:admin-qrcode-i18n
	cd web-vue && npm run check:admin-reports-i18n
	cd web-vue && npm run check:admin-users-i18n
	cd web-vue && npm run check:dashboard-i18n
	cd web-vue && npm run check:digital-human-docs

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
