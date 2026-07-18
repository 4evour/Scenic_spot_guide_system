const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");

const { buildBrowserTraceReport, buildReport, metricSummary } = require("./voice_latency_eval.js");

test("metricSummary reports P50, P95, and max for voice samples", () => {
  const samples = [
    { transcript_to_answer_ms: 10, tts_first_byte_ms: 20, tts_complete_ms: 40, voice_pipeline_total_ms: 50 },
    { transcript_to_answer_ms: 20, tts_first_byte_ms: 30, tts_complete_ms: 50, voice_pipeline_total_ms: 60 },
    { transcript_to_answer_ms: 30, tts_first_byte_ms: 40, tts_complete_ms: 60, voice_pipeline_total_ms: 70 },
    { transcript_to_answer_ms: 40, tts_first_byte_ms: 50, tts_complete_ms: 70, voice_pipeline_total_ms: 80 },
  ];
  assert.deepEqual(metricSummary(samples, "voice_pipeline_total_ms"), {
    count: 4,
    average_ms: 65,
    p50_ms: 60,
    p95_ms: 80,
    max_ms: 80,
  });
});

test("buildReport separates failed samples from latency metrics", () => {
  const report = buildReport(
    [
      { question: "a", passed: true, transcript_to_answer_ms: 10, tts_first_byte_ms: 20, tts_complete_ms: 30, voice_pipeline_total_ms: 40 },
      { question: "b", passed: false, error_type: "timeout", voice_pipeline_total_ms: 100 },
    ],
    { transcriptPath: "/voice", ttsPath: "/tts" },
  );
  assert.equal(report.total, 2);
  assert.equal(report.passed, 1);
  assert.equal(report.failure_rate, 0.5);
  assert.equal(report.error_stats.timeout, 1);
  assert.equal(report.latency.voice_pipeline_total.p95_ms, 100);
  assert.equal(report.external_dependency_status, "not_measured");
});

test("buildBrowserTraceReport summarizes real browser stage durations", () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "voice-trace-"));
  const file = path.join(dir, "traces.json");
  fs.writeFileSync(file, JSON.stringify([
    {
      trace_id: "voice-1",
      status: "completed",
      durations_ms: {
        mic_to_asr: 120,
        asr_to_answer_start: 5,
        answer_generation: 300,
        answer_to_tts_first_byte: 80,
        tts_to_audio_play_start: 40,
        audio_playback: 1000,
        voice_pipeline_total: 1545,
      },
    },
  ]));
  const report = buildBrowserTraceReport(file, { writeReport: false });
  fs.rmSync(dir, { recursive: true, force: true });

  assert.equal(report.mode, "browser_real_voice");
  assert.equal(report.external_dependency_status, "browser_measured");
  assert.equal(report.latency.asr.p50_ms, 120);
  assert.equal(report.latency.answer_generation.p95_ms, 300);
  assert.equal(report.latency.first_audio_play.p50_ms, 40);
  assert.equal(report.failure_rate, 0);
});
