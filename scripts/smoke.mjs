import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
const env = Object.fromEntries(readFileSync(".env", "utf8").split(/\r?\n/).filter(l => l && !l.startsWith("#")).map(l => { const i=l.indexOf("="); return [l.slice(0,i),l.slice(i+1)]; }));
for (const [name,url] of [
  ["API liveness", `http://localhost:${env.API_PORT || 8080}/healthz`],
  ["API readiness", `http://localhost:${env.API_PORT || 8080}/readyz`],
  ["web health", `http://localhost:${env.WEB_PORT || 3000}/healthz`],
  ["TURN health", "http://localhost:8081/healthz"],
]) {
  const response = await fetch(url, { signal: AbortSignal.timeout(5000) });
  assert.equal(response.status,200, name);
  console.log(name + ": " + JSON.stringify(await response.json()));
}
const home = await fetch(`http://localhost:${env.WEB_PORT || 3000}`);
assert.equal(home.status,200);
assert.match(await home.text(), /OneMinute/);
console.log("Landing page: OK");
