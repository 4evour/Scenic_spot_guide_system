# 灵山胜境知识库接入说明

## 已生成文件

- `lingshan_corpus.md`: 两份 Word 学习资料提取后的完整 Markdown 语料。
- `lingshan_chunks.jsonl`: 面向 RAG 导入的知识切片，每行一个 JSON 对象。
- `lingshan_eval_qa.json`: 用于验证知识学习效果的基础问答集。

## 推荐接入流程

1. 读取 `lingshan_chunks.jsonl`。
2. 对每条记录的 `content` 字段生成 embedding。
3. 将 `id`、`content`、`source`、`title`、`metadata` 和向量写入向量数据库。
4. 用户提问时，先对问题生成 embedding。
5. 在向量数据库中检索 Top K 相关切片，建议 K=3 到 5。
6. 将检索到的 `content` 作为上下文传给大模型。
7. 要求模型只基于上下文回答，并在可能时返回来源。

## 建议 Prompt 模板

```text
你是灵山胜境景区知识问答助手。请严格依据给定资料回答用户问题。

要求：
- 如果资料中有答案，准确回答。
- 如果资料中没有答案，说明“根据当前资料无法确认”。
- 不要编造景点信息、价格、开放时间或演出时间。
- 回答尽量简洁，必要时列出来源。

资料：
{{retrieved_context}}

用户问题：
{{user_question}}
```

## 验证方法

使用 `lingshan_eval_qa.json` 中的问题逐条测试你的 API。如果回答能覆盖标准答案中的核心事实，说明知识库检索和上下文注入基本生效。
