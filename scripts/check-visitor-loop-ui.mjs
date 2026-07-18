import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const files = {
  api: path.join(root, "web-vue/src/services/api.ts"),
  digitalHuman: path.join(root, "web-vue/src/views/DigitalHumanView.vue"),
  dashboard: path.join(root, "web-vue/src/views/DashboardView.vue"),
  zh: path.join(root, "web-vue/src/locales/zh-CN.json"),
  en: path.join(root, "web-vue/src/locales/en-US.json"),
};

const source = Object.fromEntries(
  Object.entries(files).map(([key, filePath]) => [
    key,
    fs.readFileSync(filePath, "utf8"),
  ]),
);
const zh = JSON.parse(source.zh);
const en = JSON.parse(source.en);

const requiredSnippets = [
  ["api.ts", source.api, "visitorExperienceApi"],
  ["api.ts", source.api, "/visitor/routes/recommend"],
  ["DigitalHumanView.vue", source.digitalHuman, "visitor-loop-panel"],
  ["DigitalHumanView.vue", source.digitalHuman, "selectedVisitorProfile"],
  ["DigitalHumanView.vue", source.digitalHuman, "parseRouteSpotNames"],
  ["DigitalHumanView.vue", source.digitalHuman, "resolveRouteSpotIds"],
  ["DigitalHumanView.vue", source.digitalHuman, "submitSpotRating"],
  [
    "DashboardView.vue",
    source.dashboard,
    "/admin/dashboard/visitor-experience",
  ],
  ["DashboardView.vue", source.dashboard, "visitorExperienceSummary"],
];

const requiredLocaleKeys = [
  "dh.visitorLoop.title",
  "dh.visitorLoop.profileTitle",
  "dh.visitorLoop.recommendRoutes",
  "dh.visitorLoop.ratingTitle",
  "dh.visitorLoop.ratingSubmit",
  "dashboard.sections.visitorExperience",
  "dashboard.labels.totalRatings",
  "dashboard.labels.negativeRatings",
  "dashboard.labels.routePreference",
];

function getPath(obj, dottedPath) {
  return dottedPath.split(".").reduce((current, key) => {
    if (!current || typeof current !== "object") return undefined;
    return current[key];
  }, obj);
}

const failures = [];
for (const [fileName, content, snippet] of requiredSnippets) {
  if (!content.includes(snippet)) {
    failures.push(`${fileName} missing ${snippet}`);
  }
}
for (const key of requiredLocaleKeys) {
  if (typeof getPath(zh, key) !== "string")
    failures.push(`zh-CN missing ${key}`);
  if (typeof getPath(en, key) !== "string")
    failures.push(`en-US missing ${key}`);
}

if (failures.length > 0) {
  console.error("Visitor loop UI check failed:");
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}

console.log("Visitor loop UI check passed.");
