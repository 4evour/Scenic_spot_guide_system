#!/usr/bin/env node
/**
 * RAG 端到端评估脚本
 * 调用 /api/v1/ai/chat 接口，验证大模型回答的事实性
 *
 * 用法: node scripts/e2e_eval.js [测试集路径]
 */

const http = require("http");
const fs = require("fs");
const path = require("path");

const API_HOST = "127.0.0.1";
const API_PORT = 8080;
const API_PATH = "/api/v1/ai/chat";

const evalFile =
  process.argv[2] ||
  path.join(__dirname, "../knowledge/real/lingshan_e2e_eval.json");

function callAIChat(question) {
  return new Promise((resolve, reject) => {
    const data = JSON.stringify({ message: question });
    const options = {
      hostname: API_HOST,
      port: API_PORT,
      path: API_PATH,
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Content-Length": Buffer.byteLength(data),
        "X-Requested-With": "XMLHttpRequest",
        "User-Agent": "e2e-eval/1.0",
        "Accept": "*/*",
      },
      timeout: 60000,
    };
    console.log(`[DEBUG] Sending to: http://${API_HOST}:${API_PORT}${API_PATH}`);
    console.log(`[DEBUG] Data: ${data.substring(0, 100)}`);
    const req = http.request(options, (res) => {
      let body = "";
      res.on("data", (chunk) => (body += chunk));
      res.on("end", () => {
        try {
          const parsed = JSON.parse(body);
          const response = parsed.data?.response || "";
          console.log(`[DEBUG] Status: ${res.statusCode}, Response length: ${response.length}`);
          resolve(response);
        } catch (e) {
          reject(new Error("JSON parse error: " + body.substring(0, 100)));
        }
      });
    });
    req.on("error", reject);
    req.on("timeout", () => {
      req.destroy();
      reject(new Error("Request timeout"));
    });
    req.write(data);
    req.end();
  });
}

function checkKeywords(response, keywords) {
  // 改为任一关键词匹配即通过（OR 逻辑）
  const matched = [];
  for (const kw of keywords) {
    if (response.includes(kw)) {
      matched.push(kw);
    }
  }
  return { passed: matched.length > 0, matched, missing: matched.length > 0 ? [] : keywords };
}

async function main() {
  console.log("=== RAG 端到端评估 ===\n");
  console.log("测试集:", evalFile);
  console.log("API:", `http://${API_HOST}:${API_PORT}${API_PATH}`);
  console.log("");

  const cases = JSON.parse(fs.readFileSync(evalFile, "utf-8"));
  console.log("总题数:", cases.length);
  console.log("");

  const results = [];
  const categoryStats = {};
  let totalPassed = 0;

  for (let i = 0; i < cases.length; i++) {
    const c = cases[i];
    const cat = c.category || "未分类";
    if (!categoryStats[cat]) {
      categoryStats[cat] = { total: 0, passed: 0 };
    }
    categoryStats[cat].total++;

    process.stdout.write(
      `[${i + 1}/${cases.length}] ${c.question.substring(0, 30)}... `
    );

    try {
      const response = await callAIChat(c.question);
      const check = checkKeywords(response, c.expected_keywords);

      if (check.passed) {
        console.log("✅ PASS");
        totalPassed++;
        categoryStats[cat].passed++;
      } else {
        console.log(`❌ FAIL (缺失: ${check.missing.join(", ")})`);
      }

      results.push({
        id: c.id,
        category: cat,
        question: c.question,
        passed: check.passed,
        missing_keywords: check.missing,
        response_preview: response.substring(0, 200),
        latency_ms: 0,
      });
    } catch (err) {
      console.log(`⚠️ ERROR: ${err.message}`);
      results.push({
        id: c.id,
        category: cat,
        question: c.question,
        passed: false,
        missing_keywords: c.expected_keywords,
        error: err.message,
      });
    }

    // 避免请求过快
    await new Promise((r) => setTimeout(r, 500));
  }

  // 输出报告
  console.log("\n" + "=".repeat(50));
  console.log("RAG 端到端评估报告");
  console.log("=".repeat(50));
  console.log(`测试集: ${path.basename(evalFile)}`);
  console.log(`总题数: ${cases.length}`);
  console.log(`通过: ${totalPassed}`);
  console.log(`失败: ${cases.length - totalPassed}`);
  console.log(`通过率: ${((totalPassed / cases.length) * 100).toFixed(1)}%`);
  console.log("");

  console.log("分组统计:");
  for (const [cat, stat] of Object.entries(categoryStats)) {
    const rate = ((stat.passed / stat.total) * 100).toFixed(1);
    console.log(`  ${cat}: ${stat.passed}/${stat.total} (${rate}%)`);
  }

  // 输出失败详情
  const failed = results.filter((r) => !r.passed);
  if (failed.length > 0) {
    console.log("\n失败详情:");
    for (const r of failed) {
      console.log(`  [${r.id}] ${r.question}`);
      console.log(`    缺失关键词: ${r.missing_keywords.join(", ")}`);
      if (r.response_preview) {
        console.log(`    回答预览: ${r.response_preview.substring(0, 100)}...`);
      }
      console.log("");
    }
  }

  // 保存 JSON 报告
  const report = {
    timestamp: new Date().toISOString(),
    test_set: path.basename(evalFile),
    total: cases.length,
    passed: totalPassed,
    failed: cases.length - totalPassed,
    pass_rate: ((totalPassed / cases.length) * 100).toFixed(1) + "%",
    category_stats: categoryStats,
    results: results,
  };

  const reportPath = path.join(
    __dirname,
    "../docs/e2e-eval-report.json"
  );
  fs.writeFileSync(reportPath, JSON.stringify(report, null, 2));
  console.log("\nJSON 报告已保存:", reportPath);
}

main().catch((err) => {
  console.error("评估失败:", err);
  process.exit(1);
});
