#!/usr/bin/env node
/**
 * Simulated voice pipeline benchmark: transcript -> guide answer -> streaming TTS.
 * This script does not claim to measure microphone, ASR, speaker, or VTuber device latency.
 */

const fs = require("fs");
const http = require("http");
const path = require("path");

const DEFAULT_HOST = "127.0.0.1";
const DEFAULT_PORT = 8080;
const DEFAULT_TRANSCRIPT_PATH = "/api/v1/dh/chat/voice-transcript";
const DEFAULT_TTS_PATH = "/api/v1/ai/tts/stream";
const DEFAULT_TIMEOUT_MS = 60000;
const DEFAULT_DELAY_MS = 300;
const DEFAULT_QUESTIONS = [
  "灵山大佛有多高？",
  "九龙灌浴是什么表演？",
  "带孩子适合怎么游览？",
  "景区有哪些文化景点？",
  "梵宫在哪里？",
  "景区几点开门？",
  "有没有适合老人的路线？",
  "灵山胜境有什么特色？",
  "半天时间怎么安排？",
  "景区附近有什么餐饮？",
];

function monotonicMs() {
  return Number(process.hrtime.bigint()) / 1e6;
}

function percentile(values, percentileValue) {
  if (values.length === 0) return null;
  const sorted = values.toSorted((a, b) => a - b);
  const rank = Math.max(1, Math.ceil((percentileValue / 100) * sorted.length));
  return Math.round(sorted[rank - 1]);
}

function metricSummary(samples, field) {
  const values = samples.map((sample) => sample[field]).filter((value) => Number.isFinite(value));
  return {
    count: values.length,
    average_ms: values.length > 0 ? Math.round(values.reduce((sum, value) => sum + value, 0) / values.length) : null,
    p50_ms: percentile(values, 50),
    p95_ms: percentile(values, 95),
    max_ms: values.length > 0 ? Math.round(Math.max(...values)) : null,
  };
}

function normalizeSample(sample) {
  const durations = sample.durations_ms || {};
  return {
    ...sample,
    passed: sample.passed ?? sample.status === 'completed',
    error_type: sample.error_type || (sample.status === 'failed' ? 'voice_pipeline_error' : null),
    mic_to_asr_ms: sample.mic_to_asr_ms ?? durations.mic_to_asr,
    asr_to_answer_start_ms: sample.asr_to_answer_start_ms ?? durations.asr_to_answer_start,
    answer_generation_ms: sample.answer_generation_ms ?? durations.answer_generation,
    answer_to_tts_first_byte_ms: sample.answer_to_tts_first_byte_ms ?? durations.answer_to_tts_first_byte,
    tts_to_audio_play_start_ms: sample.tts_to_audio_play_start_ms ?? durations.tts_to_audio_play_start,
    audio_playback_ms: sample.audio_playback_ms ?? durations.audio_playback,
    voice_pipeline_total_ms: sample.voice_pipeline_total_ms ?? durations.voice_pipeline_total,
  };
}

function requestJSON(options, payload) {
  const startedAt = monotonicMs();
  const body = JSON.stringify(payload);
  const headers = {
    "Content-Type": "application/json",
    "Content-Length": Buffer.byteLength(body),
    "X-Requested-With": "XMLHttpRequest",
    Accept: "application/json",
  };
  if (options.authToken) headers.Cookie = `auth_token=${options.authToken}`;

  return new Promise((resolve) => {
    const request = http.request(
      {
        hostname: options.host,
        port: options.port,
        path: options.path,
        method: "POST",
        headers,
        timeout: options.timeoutMs,
      },
      (response) => {
        let responseBody = "";
        response.setEncoding("utf8");
        response.on("data", (chunk) => {
          responseBody += chunk;
        });
        response.on("end", () => {
          const latencyMs = monotonicMs() - startedAt;
          if (response.statusCode < 200 || response.statusCode >= 300) {
            resolve({ ok: false, error_type: "http_error", status_code: response.statusCode, latency_ms: latencyMs });
            return;
          }
          try {
            resolve({ ok: true, status_code: response.statusCode, latency_ms: latencyMs, data: JSON.parse(responseBody) });
          } catch {
            resolve({ ok: false, error_type: "parse_error", status_code: response.statusCode, latency_ms: latencyMs });
          }
        });
      },
    );
    request.on("timeout", () => request.destroy(new Error("request timeout")));
    request.on("error", (error) => {
      resolve({
        ok: false,
        error_type: error.message === "request timeout" ? "timeout" : "network_error",
        status_code: null,
        latency_ms: monotonicMs() - startedAt,
      });
    });
    request.write(body);
    request.end();
  });
}

function streamTTS(options, text) {
  const startedAt = monotonicMs();
  const body = JSON.stringify({ text, voice: options.voice, rate: options.rate });
  const headers = {
    "Content-Type": "application/json",
    "Content-Length": Buffer.byteLength(body),
    "X-Requested-With": "XMLHttpRequest",
    Accept: "audio/mpeg",
  };
  if (options.authToken) headers.Cookie = `auth_token=${options.authToken}`;

  return new Promise((resolve) => {
    const request = http.request(
      {
        hostname: options.host,
        port: options.port,
        path: options.path,
        method: "POST",
        headers,
        timeout: options.timeoutMs,
      },
      (response) => {
        let firstByteMs = null;
        let bytes = 0;
        response.on("data", (chunk) => {
          if (firstByteMs === null) firstByteMs = monotonicMs() - startedAt;
          bytes += chunk.length;
        });
        response.on("end", () => {
          const completeMs = monotonicMs() - startedAt;
          if (response.statusCode < 200 || response.statusCode >= 300 || bytes === 0) {
            resolve({
              ok: false,
              error_type: response.statusCode < 200 || response.statusCode >= 300 ? "http_error" : "empty_audio",
              status_code: response.statusCode,
              first_byte_ms: firstByteMs,
              complete_ms: completeMs,
              audio_bytes: bytes,
            });
            return;
          }
          resolve({
            ok: true,
            status_code: response.statusCode,
            first_byte_ms: firstByteMs,
            complete_ms: completeMs,
            audio_bytes: bytes,
          });
        });
      },
    );
    request.on("timeout", () => request.destroy(new Error("request timeout")));
    request.on("error", (error) => {
      resolve({
        ok: false,
        error_type: error.message === "request timeout" ? "timeout" : "network_error",
        status_code: null,
        first_byte_ms: null,
        complete_ms: monotonicMs() - startedAt,
        audio_bytes: 0,
      });
    });
    request.write(body);
    request.end();
  });
}

async function runSample(question, options, index) {
  const totalStartedAt = monotonicMs();
  const transcript = await requestJSON(
    { ...options, path: options.transcriptPath },
    {
      session_id: `voice-latency-${Date.now()}-${index}`,
      transcript: question,
      confidence: 1,
    },
  );
  if (!transcript.ok) {
    return {
      question,
      passed: false,
      error_type: transcript.error_type,
      status_code: transcript.status_code,
      transcript_to_answer_ms: transcript.latency_ms,
      answer_generation_ms: transcript.latency_ms,
      voice_pipeline_total_ms: monotonicMs() - totalStartedAt,
    };
  }

  const answer = transcript.data?.data?.answer_text || transcript.data?.answer_text || "";
  if (!answer) {
    return {
      question,
      passed: false,
      error_type: "empty_answer",
      status_code: transcript.status_code,
      transcript_to_answer_ms: transcript.latency_ms,
      answer_generation_ms: transcript.latency_ms,
      voice_pipeline_total_ms: monotonicMs() - totalStartedAt,
    };
  }

  const tts = await streamTTS({ ...options, path: options.ttsPath }, answer);
  return {
    question,
    passed: tts.ok,
    error_type: tts.ok ? null : tts.error_type,
    status_code: tts.status_code,
    transcript_to_answer_ms: Math.round(transcript.latency_ms),
    answer_generation_ms: Math.round(transcript.latency_ms),
    answer_to_tts_first_byte_ms:
      tts.first_byte_ms === null ? null : Math.round(transcript.latency_ms + tts.first_byte_ms),
    tts_first_byte_ms: tts.first_byte_ms === null ? null : Math.round(tts.first_byte_ms),
    tts_complete_ms: Math.round(tts.complete_ms),
    voice_pipeline_total_ms: Math.round(monotonicMs() - totalStartedAt),
    audio_bytes: tts.audio_bytes,
  };
}

function buildReport(samples, options) {
  options ||= {};
  const normalizedSamples = samples.map(normalizeSample);
  const failed = normalizedSamples.filter((sample) => !sample.passed);
  return {
    timestamp: new Date().toISOString(),
    mode: options.mode || "simulated_transcript_to_tts",
    external_dependency_status: options.externalDependencyStatus || "not_measured",
    transcript_endpoint: options.transcriptPath,
    tts_endpoint: options.ttsPath,
    total: normalizedSamples.length,
    passed: normalizedSamples.length - failed.length,
    failed: failed.length,
    failure_rate: normalizedSamples.length > 0 ? Number((failed.length / normalizedSamples.length).toFixed(4)) : null,
    error_stats: failed.reduce((stats, sample) => {
      stats[sample.error_type || "unknown"] = (stats[sample.error_type || "unknown"] || 0) + 1;
      return stats;
    }, {}),
    latency: {
      asr: metricSummary(normalizedSamples, "mic_to_asr_ms"),
      asr_to_answer_start: metricSummary(normalizedSamples, "asr_to_answer_start_ms"),
      answer_generation: metricSummary(normalizedSamples, "answer_generation_ms"),
      answer_to_tts_first_byte: metricSummary(normalizedSamples, "answer_to_tts_first_byte_ms"),
      first_audio_play: metricSummary(normalizedSamples, "tts_to_audio_play_start_ms"),
      audio_playback: metricSummary(normalizedSamples, "audio_playback_ms"),
      transcript_to_answer: metricSummary(normalizedSamples, "transcript_to_answer_ms"),
      tts_first_byte: metricSummary(normalizedSamples, "tts_first_byte_ms"),
      tts_complete: metricSummary(normalizedSamples, "tts_complete_ms"),
      voice_pipeline_total: metricSummary(normalizedSamples, "voice_pipeline_total_ms"),
    },
    samples: normalizedSamples,
  };
}

function loadVoiceTraceSamples(filePath) {
  const data = JSON.parse(fs.readFileSync(filePath, "utf8"));
  if (Array.isArray(data)) return data;
  if (Array.isArray(data.samples)) return data.samples;
  if (Array.isArray(data.traces)) return data.traces;
  throw new Error("voice trace file must contain an array, samples, or traces");
}

function writeReportFiles(report, reportDir, prefix = "voice-latency") {
  fs.mkdirSync(reportDir, { recursive: true });
  const timestampName = report.timestamp.replace(/[:.]/g, "-");
  fs.writeFileSync(path.join(reportDir, `${prefix}-${timestampName}.json`), JSON.stringify(report, null, 2));
  fs.writeFileSync(path.join(reportDir, `${prefix}-latest.json`), JSON.stringify(report, null, 2));
}

function buildBrowserTraceReport(filePath, options = {}) {
  const report = buildReport(loadVoiceTraceSamples(filePath), {
    ...options,
    mode: "browser_real_voice",
    externalDependencyStatus: "browser_measured",
  });
  if (options.writeReport !== false) {
    writeReportFiles(report, options.reportDir || path.join(__dirname, "../docs/eval-results"));
  }
  return report;
}

async function runVoiceBenchmark(options = {}) {
  const settings = {
    host: options.host || DEFAULT_HOST,
    port: options.port || DEFAULT_PORT,
    transcriptPath: options.transcriptPath || DEFAULT_TRANSCRIPT_PATH,
    ttsPath: options.ttsPath || DEFAULT_TTS_PATH,
    timeoutMs: options.timeoutMs || DEFAULT_TIMEOUT_MS,
    voice: options.voice || "female_xiaoxiao",
    rate: options.rate || "+0%",
    authToken: options.authToken || "",
  };
  const questions = options.questions || DEFAULT_QUESTIONS;
  const warmup = options.warmup ?? 3;
  const sampleCount = options.sampleCount ?? 20;
  const delayMs = options.delayMs ?? DEFAULT_DELAY_MS;

  for (let index = 0; index < warmup; index += 1) {
    await runSample(questions[index % questions.length], settings, `warmup-${index}`);
  }

  const samples = [];
  for (let index = 0; index < sampleCount; index += 1) {
    const sample = await runSample(questions[index % questions.length], settings, index);
    samples.push(sample);
    if (delayMs > 0 && index < sampleCount - 1) {
      await new Promise((resolve) => setTimeout(resolve, delayMs));
    }
  }

  const report = buildReport(samples, settings);
  if (options.writeReport !== false) {
    writeReportFiles(report, options.reportDir || path.join(__dirname, "../docs/eval-results"));
  }
  return report;
}

async function main() {
  const traceFlagIndex = process.argv.indexOf('--traces');
  const traceFile = traceFlagIndex >= 0 ? process.argv[traceFlagIndex + 1] : process.env.E2E_VOICE_TRACE_FILE;
  const report = traceFile
    ? buildBrowserTraceReport(traceFile, {
        writeReport: true,
        reportDir: process.env.E2E_VOICE_REPORT_DIR,
      })
    : await runVoiceBenchmark({
    host: process.env.E2E_VOICE_HOST || DEFAULT_HOST,
    port: Number(process.env.E2E_VOICE_PORT) || DEFAULT_PORT,
    timeoutMs: Number(process.env.E2E_VOICE_TIMEOUT_MS) || DEFAULT_TIMEOUT_MS,
    authToken: process.env.E2E_VOICE_AUTH_TOKEN || "",
    voice: process.env.E2E_VOICE_ID || "female_xiaoxiao",
    rate: process.env.E2E_VOICE_RATE || "+0%",
    warmup: Number(process.env.E2E_VOICE_WARMUP ?? 3),
    sampleCount: Number(process.env.E2E_VOICE_SAMPLES ?? 20),
    delayMs: Number(process.env.E2E_VOICE_DELAY_MS ?? DEFAULT_DELAY_MS),
      });
  console.log("=== 语音链路延迟报告 ===");
  console.log(`样本: ${report.total}, 成功: ${report.passed}, 失败: ${report.failed}`);
  console.log(`失败率: ${report.failure_rate === null ? "-" : `${(report.failure_rate * 100).toFixed(1)}%`}`);
  console.log(`ASR P50/P95: ${report.latency.asr.p50_ms ?? "-"}/${report.latency.asr.p95_ms ?? "-"} ms`);
  console.log(`回答生成 P50/P95: ${report.latency.answer_generation.p50_ms ?? "-"}/${report.latency.answer_generation.p95_ms ?? "-"} ms`);
  console.log(`回答 P50/P95: ${report.latency.transcript_to_answer.p50_ms ?? "-"}/${report.latency.transcript_to_answer.p95_ms ?? "-"} ms`);
  const ttsFirstByte = report.latency.answer_to_tts_first_byte.count > 0 ? report.latency.answer_to_tts_first_byte : report.latency.tts_first_byte;
  const audioComplete = report.latency.audio_playback.count > 0 ? report.latency.audio_playback : report.latency.tts_complete;
  console.log(`TTS 首字节 P50/P95: ${ttsFirstByte.p50_ms ?? "-"}/${ttsFirstByte.p95_ms ?? "-"} ms`);
  console.log(`首次播放 P50/P95: ${report.latency.first_audio_play.p50_ms ?? "-"}/${report.latency.first_audio_play.p95_ms ?? "-"} ms`);
  console.log(`播放完成 P50/P95: ${audioComplete.p50_ms ?? "-"}/${audioComplete.p95_ms ?? "-"} ms`);
  console.log(`总链路 P50/P95: ${report.latency.voice_pipeline_total.p50_ms ?? "-"}/${report.latency.voice_pipeline_total.p95_ms ?? "-"} ms`);
  if (report.failed > 0) process.exitCode = 1;
}

if (require.main === module) {
  main().catch((error) => {
    console.error("语音评测失败:", error.message);
    process.exitCode = 1;
  });
}

module.exports = {
  buildBrowserTraceReport,
  buildReport,
  loadVoiceTraceSamples,
  metricSummary,
  percentile,
  runVoiceBenchmark,
  runSample,
};
