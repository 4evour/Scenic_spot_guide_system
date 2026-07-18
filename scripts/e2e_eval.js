#!/usr/bin/env node
/**
 * End-to-end answer evaluation for the scenic guide API.
 *
 * Usage:
 *   node scripts/e2e_eval.js [evaluation-file]
 */

const http = require("http");
const fs = require("fs");
const path = require("path");

const DEFAULT_HOST = "127.0.0.1";
const DEFAULT_PORT = 8080;
const DEFAULT_API_PATH = "/api/v1/ai/chat";
const DEFAULT_TIMEOUT_MS = 60000;
const DEFAULT_DELAY_MS = 500;

function monotonicMs() {
  return Number(process.hrtime.bigint()) / 1e6;
}

function normalizeCase(testCase) {
  const requiredKeywords = Array.isArray(testCase.required_keywords)
    ? testCase.required_keywords
    : Array.isArray(testCase.expected_keywords)
      ? testCase.expected_keywords
      : [];
  const keywordGroups = Array.isArray(testCase.keyword_groups)
    ? testCase.keyword_groups.filter((group) => Array.isArray(group) && group.length > 0)
    : [];
  const forbiddenKeywords = Array.isArray(testCase.forbidden_keywords)
    ? testCase.forbidden_keywords
    : [];
  const minAnswerChars = Number.isFinite(Number(testCase.min_answer_chars))
    ? Math.max(0, Number(testCase.min_answer_chars))
    : 0;
  const expectedFacts = Array.isArray(testCase.expected_facts)
    ? testCase.expected_facts.map(normalizeFact).filter((fact) => fact.values.length > 0)
    : [];
  const expectedSourceIds = Array.isArray(testCase.expected_source_ids)
    ? testCase.expected_source_ids.filter((id) => typeof id === "string" && id.length > 0)
    : [];
  const abstentionKeywords = Array.isArray(testCase.abstention_keywords)
    ? testCase.abstention_keywords
    : ["无法确认", "资料不足", "没有找到足够依据", "不能直接", "以官方最新公告"];

  return {
    ...testCase,
    requiredKeywords,
    keywordGroups,
    forbiddenKeywords,
    minAnswerChars,
    expectedFacts,
    mustHaveSources: testCase.must_have_sources === true,
    expectedSourceIds,
    shouldAbstain: testCase.should_abstain === true,
    abstentionKeywords,
    contextRequired: Array.isArray(testCase.context_required) ? testCase.context_required : [],
    expectedAnswerType: testCase.expected_answer_type || "unspecified",
  };
}

function normalizeFact(fact) {
  if (typeof fact === "string") {
    return { type: "text", values: [fact], unit: "" };
  }
  if (!fact || typeof fact !== "object") {
    return { type: "text", values: [], unit: "" };
  }
  const values = [fact.value, ...(Array.isArray(fact.aliases) ? fact.aliases : [])]
    .filter((value) => value !== undefined && value !== null)
    .map(String)
    .map((value) => value.trim())
    .filter(Boolean);
  return {
    type: fact.type || "text",
    label: fact.label || "",
    values,
    unit: fact.unit ? String(fact.unit) : "",
  };
}

function comparableText(value) {
  return String(value || "")
    .replace(/\s+/g, "")
    .replace(/[，。；：、“”‘’（）()【】[\]]/g, "");
}

function factMatches(response, fact) {
  const comparableResponse = comparableText(response);
  return fact.values.some((value) => {
    const comparableValue = comparableText(value);
    if (!comparableResponse.includes(comparableValue)) return false;
    return !fact.unit || comparableResponse.includes(comparableText(fact.unit));
  });
}

function checkAnswer(response, testCase, metadata = {}) {
  const normalized = normalizeCase(testCase);
  const matchedRequired = normalized.requiredKeywords.filter((keyword) => response.includes(keyword));
  const missingRequired = normalized.requiredKeywords.filter((keyword) => !response.includes(keyword));
  const matchedGroups = normalized.keywordGroups.map((group) => group.filter((keyword) => response.includes(keyword)));
  const missingGroups = normalized.keywordGroups
    .map((group, index) => ({ index, expected: group, matched: matchedGroups[index] }))
    .filter((group) => group.matched.length === 0);
  const forbiddenMatches = normalized.forbiddenKeywords.filter((keyword) => response.includes(keyword));
  const matchedFacts = normalized.expectedFacts.filter((fact) => factMatches(response, fact));
  const missingFacts = normalized.expectedFacts.filter((fact) => !factMatches(response, fact));
  const sources = Array.isArray(metadata.sources) ? metadata.sources : [];
  const sourceIds = sources.map((source) => source && source.id).filter(Boolean);
  const missingSourceIds = normalized.expectedSourceIds.filter((id) => !sourceIds.includes(id));
  const hasStructuredAbstention = metadata.hasStructuredAbstention === true;
  const actualShouldAbstain = hasStructuredAbstention
    ? metadata.shouldAbstain === true
    : normalized.abstentionKeywords.some((keyword) => response.includes(keyword));
  const missingContext = normalized.contextRequired.filter((keyword) => !response.includes(keyword));
  const answerLength = Array.from(response).length;
  const answerTooShort = answerLength < normalized.minAnswerChars;
  const factsPassed = missingFacts.length === 0;
  const sourcePassed =
    (!normalized.mustHaveSources && normalized.expectedSourceIds.length === 0) ||
    (sources.length > 0 && missingSourceIds.length === 0);
  const sourceApplicable = normalized.mustHaveSources || normalized.expectedSourceIds.length > 0;
  const abstentionPassed = normalized.shouldAbstain ? actualShouldAbstain : !actualShouldAbstain;
  const contextPassed = missingContext.length === 0;
  const passed =
    missingRequired.length === 0 &&
    missingGroups.length === 0 &&
    forbiddenMatches.length === 0 &&
    factsPassed &&
    sourcePassed &&
    abstentionPassed &&
    contextPassed &&
    !answerTooShort;

  const failureReasons = [];
  if (missingRequired.length > 0 || missingGroups.length > 0) failureReasons.push("keyword_miss");
  if (missingFacts.length > 0) failureReasons.push("fact_miss");
  if (forbiddenMatches.length > 0) failureReasons.push("forbidden_content");
  if (!sourcePassed) failureReasons.push("source_coverage_miss");
  if (normalized.shouldAbstain && !actualShouldAbstain) failureReasons.push("ungrounded_answer");
  if (!normalized.shouldAbstain && actualShouldAbstain) failureReasons.push("unexpected_abstention");
  if (!contextPassed) failureReasons.push("context_miss");
  if (answerTooShort) failureReasons.push("answer_too_short");

  return {
    passed,
    matchedRequired,
    missingRequired,
    matchedGroups,
    missingGroups,
    forbiddenMatches,
    matchedFacts,
    missingFacts,
    missingSourceIds,
    sourceCount: sources.length,
    sourcePassed,
    sourceApplicable,
    actualShouldAbstain,
    abstentionPassed,
    missingContext,
    contextPassed,
    failureReasons,
    factsTotal: normalized.expectedFacts.length,
    factsMatched: matchedFacts.length,
    answerLength,
    minAnswerChars: normalized.minAnswerChars,
    answerTooShort,
  };
}

function percentile(values, percentileValue) {
  if (values.length === 0) return null;
  const sorted = values.toSorted((a, b) => a - b);
  const rank = Math.max(1, Math.ceil((percentileValue / 100) * sorted.length));
  return Math.round(sorted[rank - 1]);
}

function summarizeLatencies(results) {
  const values = results
    .map((result) => result.latency_ms)
    .filter((value) => Number.isFinite(value));
  return {
    count: values.length,
    p50_ms: percentile(values, 50),
    p95_ms: percentile(values, 95),
    max_ms: values.length > 0 ? Math.round(Math.max(...values)) : null,
    average_ms: values.length > 0 ? Math.round(values.reduce((sum, value) => sum + value, 0) / values.length) : null,
  };
}

function extractResponse(payload) {
  return payload?.data?.answer || payload?.data?.response || payload?.answer || payload?.response || "";
}

function extractStructuredFields(payload) {
  const data = payload?.data || payload || {};
  return {
    sources: Array.isArray(data.sources) ? data.sources : [],
    confidence: Number.isFinite(Number(data.confidence)) ? Number(data.confidence) : null,
    shouldAbstain: typeof data.should_abstain === "boolean" ? data.should_abstain : null,
    emotion: typeof data.emotion === "string" ? data.emotion : null,
    hasStructuredAbstention: typeof data.should_abstain === "boolean",
  };
}

function buildEvaluationResult(testCase, requestResult) {
  const normalized = normalizeCase(testCase);
  if (!requestResult.ok) {
    return {
      id: testCase.id,
      category: testCase.category || "未分类",
      expected_answer_type: normalized.expectedAnswerType,
      question: testCase.question,
      passed: false,
      error_type: requestResult.error_type,
      error: requestResult.error,
      status_code: requestResult.status_code,
      latency_ms: Math.round(requestResult.latency_ms),
      facts_total: normalized.expectedFacts.length,
      facts_matched: 0,
      source_coverage: normalized.mustHaveSources || normalized.expectedSourceIds.length > 0 ? false : null,
      expected_should_abstain: normalized.shouldAbstain,
      actual_should_abstain: false,
      failure_reasons: [requestResult.error_type || "request_error"],
    };
  }

  const check = checkAnswer(requestResult.answer, normalized, requestResult);
  return {
    id: testCase.id,
    category: testCase.category || "未分类",
    expected_answer_type: normalized.expectedAnswerType,
    question: testCase.question,
    passed: check.passed,
    status_code: requestResult.status_code,
    latency_ms: Math.round(requestResult.latency_ms),
    answer_length: check.answerLength,
    matched_required_keywords: check.matchedRequired,
    missing_required_keywords: check.missingRequired,
    matched_keyword_groups: check.matchedGroups,
    missing_keyword_groups: check.missingGroups,
    forbidden_matches: check.forbiddenMatches,
    matched_facts: check.matchedFacts,
    missing_facts: check.missingFacts,
    facts_total: check.factsTotal,
    facts_matched: check.factsMatched,
    source_count: check.sourceCount,
    missing_source_ids: check.missingSourceIds,
    source_coverage: check.sourceApplicable ? check.sourcePassed : null,
    expected_should_abstain: normalized.shouldAbstain,
    actual_should_abstain: check.actualShouldAbstain,
    abstention_passed: check.abstentionPassed,
    missing_context: check.missingContext,
    context_accuracy: check.contextPassed,
    confidence: requestResult.confidence,
    emotion: requestResult.emotion,
    failure_reasons: check.failureReasons,
    answer_too_short: check.answerTooShort,
    response_preview: requestResult.answer.slice(0, 500),
  };
}

function callAIChat(question, options = {}) {
  const hostname = options.host || DEFAULT_HOST;
  const port = options.port || DEFAULT_PORT;
  const apiPath = options.apiPath || DEFAULT_API_PATH;
  const timeoutMs = options.timeoutMs || DEFAULT_TIMEOUT_MS;
  const requestBody = { message: question };
  if (options.sessionId) requestBody.session_id = options.sessionId;
  const data = JSON.stringify(requestBody);
  const startedAt = monotonicMs();

  return new Promise((resolve) => {
    const request = http.request(
      {
        hostname,
        port,
        path: apiPath,
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Content-Length": Buffer.byteLength(data),
          "X-Requested-With": "XMLHttpRequest",
          "User-Agent": "e2e-eval/2.0",
          Accept: "application/json",
        },
        timeout: timeoutMs,
      },
      (response) => {
        let body = "";
        response.setEncoding("utf8");
        response.on("data", (chunk) => {
          body += chunk;
        });
        response.on("end", () => {
          const latencyMs = monotonicMs() - startedAt;
          if (response.statusCode < 200 || response.statusCode >= 300) {
            resolve({
              ok: false,
              status_code: response.statusCode,
              latency_ms: latencyMs,
              error_type: "http_error",
              error: `HTTP ${response.statusCode}`,
            });
            return;
          }
          let payload;
          try {
            payload = JSON.parse(body);
          } catch {
            resolve({
              ok: false,
              status_code: response.statusCode,
              latency_ms: latencyMs,
              error_type: "parse_error",
              error: "response is not valid JSON",
            });
            return;
          }
          const answer = extractResponse(payload);
          if (!answer) {
            resolve({
              ok: false,
              status_code: response.statusCode,
              latency_ms: latencyMs,
              error_type: "empty_response",
              error: "response does not contain an answer",
            });
            return;
          }
          resolve({
            ok: true,
            status_code: response.statusCode,
            latency_ms: latencyMs,
            response,
            answer,
            ...extractStructuredFields(payload),
          });
        });
      },
    );

    request.on("timeout", () => {
      request.destroy(new Error("request timeout"));
    });
    request.on("error", (error) => {
      resolve({
        ok: false,
        status_code: null,
        latency_ms: monotonicMs() - startedAt,
        error_type: error.message === "request timeout" ? "timeout" : "network_error",
        error: error.message === "request timeout" ? "request timeout" : "network request failed",
      });
    });
    request.write(data);
    request.end();
  });
}

async function runCase(testCase, options = {}) {
  const normalized = normalizeCase(testCase);
  if (!Array.isArray(testCase.turns) || testCase.turns.length === 0) {
    return buildEvaluationResult(testCase, await callAIChat(testCase.question, options));
  }

  const sessionId = testCase.session_id || `${options.sessionNamespace || "eval"}-${testCase.id || Date.now()}`;
  const turnResults = [];
  for (let index = 0; index < testCase.turns.length; index += 1) {
    const turn = testCase.turns[index];
    const turnCase = normalizeCase({
      ...testCase,
      ...turn,
      id: `${testCase.id || "case"}-turn-${index + 1}`,
      category: testCase.category || turn.category,
      question: turn.question,
      context_required: turn.context_required || (index > 0 ? normalized.contextRequired : []),
    });
    const requestResult = await callAIChat(turn.question, { ...options, sessionId });
    turnResults.push(buildEvaluationResult(turnCase, requestResult));
  }
  const passed = turnResults.every((result) => result.passed);
  const latencyMs = turnResults.reduce((sum, result) => sum + (result.latency_ms || 0), 0);
  const finalTurn = turnResults[turnResults.length - 1];
  return {
    id: testCase.id,
    category: testCase.category || "多轮上下文",
    expected_answer_type: normalized.expectedAnswerType,
    question: testCase.question || testCase.turns[0].question,
    passed,
    latency_ms: latencyMs,
    context_accuracy: passed,
    turn_results: turnResults,
    facts_total: turnResults.reduce((sum, result) => sum + (result.facts_total || 0), 0),
    facts_matched: turnResults.reduce((sum, result) => sum + (result.facts_matched || 0), 0),
    source_coverage: turnResults.some((result) => result.source_coverage !== null)
      ? turnResults.filter((result) => result.source_coverage !== null).every((result) => result.source_coverage)
      : null,
    expected_should_abstain: turnResults.some((result) => result.expected_should_abstain),
    actual_should_abstain: finalTurn.actual_should_abstain,
    confidence: finalTurn.confidence,
    emotion: finalTurn.emotion,
    failure_reasons: [...new Set(turnResults.flatMap((result) => result.failure_reasons || []))],
    response_preview: finalTurn.response_preview,
  };
}

function addCategoryResult(categoryStats, testCase, result) {
  const category = testCase.category || "未分类";
  const type = testCase.expectedAnswerType || "unspecified";
  for (const key of [category, `type:${type}`]) {
    categoryStats[key] ||= { total: 0, passed: 0, failed: 0, timed_out: 0 };
    categoryStats[key].total += 1;
    if (result.passed) categoryStats[key].passed += 1;
    else categoryStats[key].failed += 1;
    if (result.error_type === "timeout") categoryStats[key].timed_out += 1;
  }
}

async function runEvaluation(options = {}) {
  const evalFile = options.evalFile || path.join(__dirname, "../knowledge/real/lingshan_accuracy_eval.json");
  const cases = JSON.parse(fs.readFileSync(evalFile, "utf8")).map(normalizeCase);
  const delayMs = options.delayMs ?? DEFAULT_DELAY_MS;
  const results = [];
  const categoryStats = {};
  const sessionNamespace = options.sessionNamespace || `eval-${Date.now()}`;

  for (let index = 0; index < cases.length; index += 1) {
    const testCase = cases[index];
    const displayQuestion = testCase.question || testCase.turns?.[0]?.question || "";
    process.stdout.write(`[${index + 1}/${cases.length}] ${displayQuestion.slice(0, 30)}... `);
    const result = await runCase(testCase, { ...options, sessionNamespace });
    console.log(result.passed ? "PASS" : result.error_type ? `ERROR (${result.error_type})` : "FAIL");

    addCategoryResult(categoryStats, testCase, result);
    results.push(result);
    if (delayMs > 0 && index < cases.length - 1) {
      await new Promise((resolve) => setTimeout(resolve, delayMs));
    }
  }

  const passed = results.filter((result) => result.passed).length;
  const timedOut = results.filter((result) => result.error_type === "timeout").length;
  const report = {
    timestamp: new Date().toISOString(),
    test_set: path.basename(evalFile),
    total: results.length,
    passed,
    failed: results.length - passed,
    timed_out: timedOut,
    pass_rate: results.length > 0 ? Number(((passed / results.length) * 100).toFixed(1)) : 0,
    fact_accuracy: summarizeMetric(results, "facts_matched", "facts_total"),
    key_number_accuracy: summarizeFactType(results, ["number", "currency", "time", "address"]),
    source_coverage_rate: ratio(results.filter((result) => result.source_coverage === true).length, results.filter((result) => result.source_coverage !== null && result.source_coverage !== undefined).length),
    ungrounded_answer_rate: ratio(results.filter((result) => result.expected_should_abstain === true && result.actual_should_abstain !== true).length, results.filter((result) => result.expected_should_abstain === true).length),
    correct_abstention_rate: ratio(results.filter((result) => result.expected_should_abstain === true && result.actual_should_abstain === true).length, results.filter((result) => result.expected_should_abstain === true).length),
    multi_turn_context_accuracy: ratio(results.filter((result) => Array.isArray(result.turn_results)).filter((result) => result.context_accuracy).length, results.filter((result) => Array.isArray(result.turn_results)).length),
    category_stats: categoryStats,
    latency: summarizeLatencies(results),
    results,
  };

  if (options.writeReport !== false) {
    const reportDir = options.reportDir || path.join(__dirname, "../docs/eval-results");
    fs.mkdirSync(reportDir, { recursive: true });
    const timestampName = report.timestamp.replace(/[:.]/g, "-");
    fs.writeFileSync(path.join(reportDir, `e2e-eval-${timestampName}.json`), JSON.stringify(report, null, 2));
    fs.writeFileSync(path.join(reportDir, "e2e-eval-latest.json"), JSON.stringify(report, null, 2));
    fs.writeFileSync(path.join(__dirname, "../docs/e2e-eval-report.json"), JSON.stringify(report, null, 2));
  }

  return report;
}

function ratio(numerator, denominator) {
  return denominator > 0 ? Number((numerator / denominator).toFixed(4)) : null;
}

function summarizeMetric(results, matchedKey, totalKey) {
  const applicable = results.filter((result) => Number(result[totalKey]) > 0);
  const total = applicable.reduce((sum, result) => sum + result[totalKey], 0);
  const matched = applicable.reduce((sum, result) => sum + result[matchedKey], 0);
  return { matched, total, rate: ratio(matched, total) };
}

function summarizeFactType(results, types) {
  const facts = results.flatMap((result) => result.turn_results || [result]).flatMap((result) => result.matched_facts || []);
  const missing = results.flatMap((result) => result.turn_results || [result]).flatMap((result) => result.missing_facts || []);
  const all = [...facts, ...missing].filter((fact) => types.includes(fact.type));
  const matched = all.filter((fact) => facts.includes(fact)).length;
  return { matched, total: all.length, rate: ratio(matched, all.length) };
}

async function main() {
  const evalFile = process.argv[2];
  const report = await runEvaluation({
    evalFile,
    host: process.env.E2E_EVAL_HOST || DEFAULT_HOST,
    port: Number(process.env.E2E_EVAL_PORT) || DEFAULT_PORT,
    timeoutMs: Number(process.env.E2E_EVAL_TIMEOUT_MS) || DEFAULT_TIMEOUT_MS,
    delayMs: Number(process.env.E2E_EVAL_DELAY_MS ?? DEFAULT_DELAY_MS),
  });
  console.log("\n=== 最终回答评测报告 ===");
  console.log(`总题数: ${report.total}`);
  console.log(`通过: ${report.passed}`);
  console.log(`失败: ${report.failed}`);
  console.log(`超时: ${report.timed_out}`);
  console.log(`准确率: ${report.pass_rate}%`);
  console.log(`事实准确率: ${report.fact_accuracy.rate === null ? "-" : `${(report.fact_accuracy.rate * 100).toFixed(1)}%`}`);
  console.log(`来源覆盖率: ${report.source_coverage_rate === null ? "-" : `${(report.source_coverage_rate * 100).toFixed(1)}%`}`);
  console.log(`正确拒答率: ${report.correct_abstention_rate === null ? "-" : `${(report.correct_abstention_rate * 100).toFixed(1)}%`}`);
  console.log(`无依据回答率: ${report.ungrounded_answer_rate === null ? "-" : `${(report.ungrounded_answer_rate * 100).toFixed(1)}%`}`);
  console.log(`多轮上下文准确率: ${report.multi_turn_context_accuracy === null ? "-" : `${(report.multi_turn_context_accuracy * 100).toFixed(1)}%`}`);
  console.log(`延迟 P50/P95: ${report.latency.p50_ms ?? "-"}/${report.latency.p95_ms ?? "-"} ms`);
  if (report.failed > 0) {
    console.log("失败题目:");
    for (const result of report.results.filter((item) => !item.passed)) {
      const reasons = (result.failure_reasons || [result.error_type || "未分类"]).join(", ");
      console.log(`- ${result.id || "未命名"}: ${result.question || "多轮题"} [${reasons}]`);
    }
  }
  if (report.failed > 0 || report.timed_out > 0) {
    process.exitCode = 1;
  }
}

if (require.main === module) {
  main().catch((error) => {
    console.error("评测失败:", error.message);
    process.exitCode = 1;
  });
}

module.exports = {
  callAIChat,
  checkAnswer,
  extractStructuredFields,
  normalizeCase,
  percentile,
  runEvaluation,
  summarizeLatencies,
};
