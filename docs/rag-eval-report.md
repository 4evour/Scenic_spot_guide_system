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

本地多模式对比：

```powershell
go run ./cmd/rag-eval -knowledge knowledge/real/lingshan_real_chunks.jsonl -eval knowledge/real/lingshan_real_eval_open.json -k 8 -bench -concurrency 16 -repeat 1 -retrieval-only -compare-modes bm25-local,light-rerank
```

指定轻量 rerank 并输出 JSON：

```powershell
go run ./cmd/rag-eval -knowledge knowledge/real/lingshan_real_chunks.jsonl -eval knowledge/real/lingshan_real_eval_open.json -k 8 -bench -concurrency 16 -repeat 1 -retrieval-only -mode light-rerank -report-env -format json -out docs/eval-results/lingshan-real-rag-eval-light-rerank.json
```

失败样例定向优化后输出 JSON：

```powershell
go run ./cmd/rag-eval -knowledge knowledge/real/lingshan_real_chunks.jsonl -eval knowledge/real/lingshan_real_eval_open.json -k 8 -bench -concurrency 16 -repeat 1 -retrieval-only -compare-modes bm25-local,light-rerank -format json -out docs/eval-results/lingshan-real-rag-eval-targeted-improvement.json
```

## 优化前真实资料基线

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

## 多模式评估

`cmd/rag-eval` 现在支持以下检索模式：

- `bm25-local`：纯本地 BM25/词面检索，不依赖外部 Key，是默认可复现基线。
- `embedding`：使用外部 Embedding Provider 做语义相似度检索，需要 `embedding.api_key`。
- `hybrid-weighted`：按 `-embedding-weight` 与 `-bm25-weight` 做加权融合，适合做参数实验，但对不同分数尺度更敏感。
- `rrf-fusion`：使用 Reciprocal Rank Fusion 融合 BM25 与 Embedding 排名，默认推荐作为混合召回实验方向，因为它比直接加权更不依赖分数归一化。
- `light-rerank`：在本地 BM25 候选上做可解释重排，特征包括标题命中、查询词覆盖、景区实体词和来源类型；暂不引入 Cross-Encoder，避免增加部署复杂度。

2026-05-26 优化前在真实资料集上做的本地 retrieval-only 单轮对比：

| 模式 | 通过率 | Recall@8 | MRR@8 | 关键词覆盖率 | p50/p95 |
| --- | ---: | ---: | ---: | ---: | ---: |
| `bm25-local` | 88.2% | 85.5% | 0.749 | 94.3% | 约 8ms / 12ms |
| `light-rerank` | 88.2% | 86.0% | 0.761 | 94.6% | 约 9ms / 13ms |

这组结果说明轻量 rerank 对排序质量有小幅改善，但没有提高通过率；它更适合作为“可解释排序优化”的证据，而不是包装成模型能力跃迁。`embedding`、`hybrid-weighted` 和 `rrf-fusion` 需要可用 Embedding 配置，报告中必须和纯本地 BM25 分开说明。

## 失败样例定向优化结果

本阶段没有放宽评测规则，也没有删除困难问题；改动集中在两点：

- 在检索前增加本地 query expansion，只参与候选召回和打分，不改写用户原始问题，也不把扩展词写入生成 prompt。
- 按失败样例补强少量真实资料切片中的游客常用词和边界词，例如半天游、中轴线、导览服务、宠物/无人机、容易过期信息、只看大佛/商业化游乐等。

2026-05-26 定向优化后的真实资料 retrieval-only 单轮对比：

| 模式 | 通过率 | Recall@8 | MRR@8 | 关键词覆盖率 | p50/p95 | 失败数 |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `bm25-local` | 98.5% | 94.8% | 0.793 | 99.3% | 约 9ms / 16-19ms | 3 |
| `light-rerank` | 99.5% | 95.3% | 0.802 | 99.8% | 约 10ms / 20-21ms | 1 |

改善最明显的原失败类型：

- 开放路线问法：半天游、亲子路线、中轴线从“泛路线片段靠前”改善为能召回目标主线/亲子切片。
- 文化建筑问法：大佛之外、三大语系、藏式风格、木雕壁画琉璃等问法更稳定召回五印坛城、曼飞龙塔、灵山梵宫相关切片。
- 实时边界问法：排队、人多、宠物、无人机、导览服务、容易过期信息能优先召回边界切片，避免把离线资料当作实时承诺。

当前 `light-rerank` 仍失败 1 条：`灵山回答能不能凭第三方评论说今天人多？`。Top1 已召回 `real-boundary-weather-001`，但检索片段中缺少期望关键词“资料库没有”，属于边界表达词覆盖不足，后续可通过更细的负样本文案或人工标注处理。这个结果只能说明当前真实资料评测集上的问法映射和排序被加强，不能外推为开放域 99% 准确率或线上 SLA。

分组表现：

- `closed_real`：74 条，bench 后 222 次评估，通过率 98.6%，Recall@8 98.6%
- `open_real`：63 条，bench 后 189 次评估，通过率 81.0%，Recall@8 69.7%
- `negative`：66 条，bench 后 198 次评估，通过率 83.3%，Recall@8 85.9%

当前失败样例会按原因写入 JSON 结果中的 `failure_reason`，用于区分事实相近但未召回、关键词未覆盖、实时信息边界、开放问法不稳定和负样本误召回等情况。后续应继续补切片、补标注和调整查询改写/重排特征，而不是删掉困难样例。

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
