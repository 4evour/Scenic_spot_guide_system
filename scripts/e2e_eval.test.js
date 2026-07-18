const assert = require("node:assert/strict");
const fs = require("node:fs");
const http = require("node:http");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const {
  callAIChat,
  checkAnswer,
  runEvaluation,
  summarizeLatencies,
} = require("./e2e_eval.js");

test("checkAnswer requires every legacy expected keyword", () => {
  const check = checkAnswer("灵山大佛高88米", {
    expected_keywords: ["灵山大佛", "88", "铜像"],
  });
  assert.equal(check.passed, false);
  assert.deepEqual(check.missingRequired, ["铜像"]);
});

test("checkAnswer supports keyword groups and forbidden keywords", () => {
  const check = checkAnswer("建议查看景区官方公告，今天暂不能确认。", {
    required_keywords: ["官方"],
    keyword_groups: [["公告", "通知"]],
    forbidden_keywords: ["一定开放"],
    min_answer_chars: 8,
  });
  assert.equal(check.passed, true);

  const forbidden = checkAnswer("景区一定开放", {
    required_keywords: ["开放"],
    forbidden_keywords: ["一定开放"],
  });
  assert.equal(forbidden.passed, false);
  assert.deepEqual(forbidden.forbiddenMatches, ["一定开放"]);
});

test("callAIChat records real request latency", async () => {
  const server = http.createServer((request, response) => {
    request.resume();
    setTimeout(() => {
      response.setHeader("Content-Type", "application/json");
      response.end(JSON.stringify({ data: { response: "灵山大佛高88米", should_abstain: false } }));
    }, 10);
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  const result = await callAIChat("灵山大佛多高", {
    host: "127.0.0.1",
    port: address.port,
    timeoutMs: 1000,
  });
  await new Promise((resolve) => server.close(resolve));

  assert.equal(result.ok, true);
  assert.equal(result.answer, "灵山大佛高88米");
  assert.equal(result.shouldAbstain, false);
  assert.equal(result.hasStructuredAbstention, true);
  assert.ok(result.latency_ms >= 0);
});

test("summarizeLatencies calculates percentile metrics", () => {
  const summary = summarizeLatencies([
    { latency_ms: 10 },
    { latency_ms: 20 },
    { latency_ms: 30 },
    { latency_ms: 40 },
  ]);
  assert.equal(summary.p50_ms, 20);
  assert.equal(summary.p95_ms, 40);
  assert.equal(summary.max_ms, 40);
});

test("checkAnswer validates facts, source coverage, and structured abstention", () => {
  const factual = checkAnswer(
    "灵山大佛通高88米，地址是马山街道灵山路1号。",
    {
      expected_facts: [
        { type: "number", label: "高度", value: "88", unit: "米" },
        { type: "address", label: "地址", value: "马山街道灵山路1号" },
      ],
      must_have_sources: true,
      expected_source_ids: ["real-dafo-001"],
    },
    {
      sources: [{ id: "real-dafo-001" }],
      shouldAbstain: false,
      hasStructuredAbstention: true,
    },
  );
  assert.equal(factual.passed, true);
  assert.equal(factual.factsMatched, 2);

  const abstention = checkAnswer(
    "当前资料不足，今天的停车余位无法确认，请以官方最新公告为准。",
    { should_abstain: true, abstention_keywords: ["无法确认", "官方最新公告"] },
    { sources: [{ id: "real-boundary-parking-001" }], shouldAbstain: true, hasStructuredAbstention: true },
  );
  assert.equal(abstention.passed, true);
  assert.equal(abstention.actualShouldAbstain, true);
});

test("runEvaluation keeps one session across multi-turn cases", async () => {
  const requests = [];
  const server = http.createServer((request, response) => {
    let body = "";
    request.setEncoding("utf8");
    request.on("data", (chunk) => {
      body += chunk;
    });
    request.on("end", () => {
      const payload = JSON.parse(body);
      requests.push(payload);
      const answer = requests.length === 1 ? "亲子路线可以先看百子戏弥勒。" : "沿用刚才的亲子路线，建议安排约半天。";
      response.setHeader("Content-Type", "application/json");
      response.end(JSON.stringify({ data: { answer, sources: [{ id: "route-001" }], confidence: 0.75, should_abstain: false } }));
    });
  });
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "e2e-eval-"));
  const evalFile = path.join(tempDir, "cases.json");
  fs.writeFileSync(evalFile, JSON.stringify([
    {
      id: "context-001",
      category: "多轮上下文",
      turns: [
        { question: "我带孩子来，想轻松游览", required_keywords: ["亲子"] },
        { question: "那路线需要多长时间？", required_keywords: ["半天"], context_required: ["亲子"] },
      ],
    },
  ]));
  const report = await runEvaluation({
    evalFile,
    host: "127.0.0.1",
    port: address.port,
    delayMs: 0,
    writeReport: false,
  });
  await new Promise((resolve) => server.close(resolve));
  fs.rmSync(tempDir, { recursive: true, force: true });

  assert.equal(report.multi_turn_context_accuracy, 1);
  assert.equal(report.results[0].passed, true);
  assert.equal(requests.length, 2);
  assert.equal(requests[0].session_id, requests[1].session_id);
});
