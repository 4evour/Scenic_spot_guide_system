# 灵山真实 RAG 数据集说明

本目录用于把灵山 RAG 评估从“合成闭集实验”扩展为更接近面试追问的真实资料评估。数据以官方公开页面和政府公开介绍为事实源，第三方攻略只用于补充游客问法，不作为权威事实。

## 文件

- `sources.yaml`：资料来源、来源类型、抓取日期、权威性和可使用字段。
- `lingshan_real_corpus.md`：人工清洗后的资料摘要，按景点、路线、服务、边界问题组织。
- `lingshan_real_chunks.jsonl`：可导入 RAG 的真实资料切片。
- `lingshan_real_eval_open.json`：203 条独立评测问答，覆盖 `closed_real`、`open_real` 和 `negative`。

当前数据规模为 122 个真实资料切片、203 条独立评测问答。并发 16、repeat 3 的 retrieval-only bench 会形成 609 次评估；一次 BM25 本地复现结果为 Recall@8 85.5%、MRR@8 0.749、关键词覆盖率 94.3%、纯检索 p50/p95 约 7ms/10ms。启用 DashScope `text-embedding-v2` 后，同口径一次复现结果为 Recall@8 85.5%、MRR@8 0.749、关键词覆盖率 94.3%、检索 p50/p95 约 69ms/80ms。

## 边界

- 本数据集不是灵山景区完整生产知识库，不覆盖所有实时通知、票务政策和临时活动。
- 票价、开放时间、演出场次、停车余位等易变信息只允许回答“以官方最新公告/购票页为准”。
- 评测指标只能说明当前资料集和评测集上的检索表现，不能外推为真实游客开放域问答的 100% 召回。
- 扩大真实资料集后指标低于合成闭集是正常现象，更能暴露开放问法、相近切片排序和资料边界处理问题。

## 复现

```powershell
go run ./cmd/rag-eval -knowledge knowledge/real/lingshan_real_chunks.jsonl -eval knowledge/real/lingshan_real_eval_open.json -k 8 -retrieval-only -report-env
```

```powershell
go run ./cmd/rag-eval -knowledge knowledge/real/lingshan_real_chunks.jsonl -eval knowledge/real/lingshan_real_eval_open.json -k 8 -bench -concurrency 16 -repeat 3 -retrieval-only -report-env -format json -out docs/eval-results/lingshan-real-rag-eval-bench.json
```

```powershell
go run ./cmd/rag-eval -knowledge knowledge/real/lingshan_real_chunks.jsonl -eval knowledge/real/lingshan_real_eval_open.json -k 8 -bench -concurrency 16 -repeat 3 -retrieval-only -embedding -config configs -report-env -format json -out docs/eval-results/lingshan-real-rag-eval-embedding-c16.json
```
