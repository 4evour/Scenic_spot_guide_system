# RAG 数据集说明

本文档说明 `knowledge/` 目录内数据文件的用途、来源和边界，方便面试官或评审复现评估结果。

## 文件清单

- `lingshan_chunks.jsonl`：基础 smoke test 知识切片，当前 81 条，按景点、小节和问法语义切分。
- `lingshan_eval_qa.json`：基础 smoke test 问答，当前 5 条。
- `lingshan_scale_3000.jsonl`：合成规模验证知识切片，当前 3000 条。
- `lingshan_eval_300.json`：合成规模验证问答，当前 300 条。

## 数据来源和生成方式

基础 81/5 数据用于说明知识导入、分片、检索和评估流程。3000/300 数据是围绕灵山胜境主题构造的合成闭集实验数据，用于验证当前检索链路在固定规模下能否稳定运行。

这些数据不是来自真实景区业务数据库，也不是官方完整导览知识库。它们不应被描述为“真实生产知识库”或“完整景区 FAQ 覆盖”。

## 评估字段

`lingshan_eval_300.json` 每条评测包含：

- `question`：评测问题。
- `expected_keywords`：回答或检索片段中应覆盖的关键词。
- `expected_chunk_ids`：期望命中的知识切片 ID。
- `category`：问题类别。
- `difficulty`：难度标签。

## 指标边界

当前 3000/300 评估是闭集检索实验：评测问题和期望切片由同一组合成流程生成，因此 Recall@8 达到 100.0% 只能说明当前检索链路能稳定找回这组合成数据中的目标切片。

该结果不能外推为真实游客开放域问答中的召回率，也不能证明模型回答没有幻觉。真实落地还需要独立人工标注测试集、真实景区资料、相关性等级、来源引用和线上反馈闭环。

## 复现命令

基础 smoke test：

```powershell
go run ./cmd/rag-eval -k 8 -fail-on-miss
```

合成规模验证：

```powershell
go run ./cmd/rag-eval -knowledge knowledge/lingshan_scale_3000.jsonl -eval knowledge/lingshan_eval_300.json -k 8 -bench -concurrency 16 -repeat 1 -retrieval-only -fail-on-miss
```
