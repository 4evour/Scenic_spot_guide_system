# RAG 评估报告

本文档记录当前 RAG 评估的复现方式、指标含义和局限。它是作品集实验报告，不是线上生产压测报告。

## 数据集

- 基础 smoke test：`knowledge/lingshan_chunks.jsonl` + `knowledge/lingshan_eval_qa.json`，32 个切片、5 条问答。
- 合成规模验证：`knowledge/lingshan_scale_3000.jsonl` + `knowledge/lingshan_eval_300.json`，3000 个切片、300 条问答，仅用于验证固定合成数据上的评估链路。
- 真实资料评估：`knowledge/real/lingshan_real_chunks.jsonl` + `knowledge/real/lingshan_real_eval_open.json`，122 个真实资料切片、203 条独立评测问答，覆盖 `closed_real`、`open_real` 和 `negative`。

真实资料以灵山胜境官网、政府公开介绍为事实源，第三方公开攻略只用于补充游客问法。来源和边界见 `knowledge/real/sources.yaml` 与 `knowledge/real/README.md`。

## 复现命令

真实资料 smoke：

```powershell
go run ./cmd/rag-eval -knowledge knowledge/real/lingshan_real_chunks.jsonl -eval knowledge/real/lingshan_real_eval_open.json -k 8 -retrieval-only -report-env
```

真实资料 bench：

```powershell
go run ./cmd/rag-eval -knowledge knowledge/real/lingshan_real_chunks.jsonl -eval knowledge/real/lingshan_real_eval_open.json -k 8 -bench -concurrency 16 -repeat 3 -retrieval-only -report-env -format json -out docs/eval-results/lingshan-real-rag-eval-bench.json
```

真实资料 Embedding bench：

```powershell
go run ./cmd/rag-eval -knowledge knowledge/real/lingshan_real_chunks.jsonl -eval knowledge/real/lingshan_real_eval_open.json -k 8 -bench -concurrency 16 -repeat 3 -retrieval-only -embedding -config configs -report-env -format json -out docs/eval-results/lingshan-real-rag-eval-embedding-c16.json
```

## 当前真实资料评估结果

- 知识切片数：122
- 评测问答数：203；bench 重复 3 轮后为 609 次评估
- TopK：8
- 并发：16
- 模式：retrieval-only，本地 BM25/词面检索，不包含 DashScope Embedding、DeepSeek 生成、ASR 或 TTS 网络链路
- 通过率：88.2%
- Recall@8：85.5%
- MRR@8：0.749
- 关键词覆盖率：94.3%
- 检索 p50/p95：约 7ms / 10ms
- 运行环境：Windows amd64，Go 1.26.1
- JSON 报告：`docs/eval-results/lingshan-real-rag-eval-bench.json`

启用 DashScope `text-embedding-v2` 后，同一数据集、同一并发 16、repeat 3、retrieval-only 口径的复现结果：

- 评估次数：609
- Embedding provider：`text-embedding-v2`
- Generation provider：`disabled`
- 通过率：88.2%
- Recall@8：85.5%
- MRR@8：0.749
- 关键词覆盖率：94.3%
- 检索 p50/p95：约 69ms / 80ms
- JSON 报告：`docs/eval-results/lingshan-real-rag-eval-embedding-c16.json`

分组表现：

- `closed_real`：74 条，bench 后 222 次评估，通过率 98.6%，Recall@8 98.6%
- `open_real`：63 条，bench 后 189 次评估，通过率 81.0%，Recall@8 69.7%
- `negative`：66 条，bench 后 198 次评估，通过率 83.3%，Recall@8 85.9%

## 指标解释

- Recall@8：期望切片是否出现在前 8 个检索结果中。
- MRR@8：第一个期望切片排名的倒数均值。
- 关键词覆盖率：检索片段拼接结果中命中期望关键词的比例。
- p50/p95：本地纯检索耗时分位数，不包含外部 Embedding、大模型生成、语音识别或 TTS。

## 局限

- 真实资料评估比合成闭集更接近游客问法，但仍不是完整景区生产知识库。
- 当前评估只验证检索链路，不验证大模型回答事实性、引用准确性或多轮对话质量。
- 数据集扩大后出现更多失败样例是预期结果，主要暴露开放问法排序、服务边界表达和相近景点召回干扰问题；这些失败样例应进入后续切片、rerank 和人工标注优化，而不是删除。
- 票价、开放时间、演出场次、停车余位等实时信息只能提示查看官方最新公告，不能写成固定答案。
- 合成 3000/300 的 100% Recall@8 只能说明合成闭集链路可运行，不作为简历卖点。
