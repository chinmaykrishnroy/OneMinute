import { chromium, expect } from "@playwright/test";
import { readFile } from "node:fs/promises";
import { createServer } from "node:http";
import ts from "typescript";

// Production media helpers, real browser WebRTC and synthetic devices.
// No account credentials, production messages or physical cameras are used.
const modules = new Map();
for (const name of ["call-media", "processed-camera"])
  modules.set(
    `/${name}.js`,
    ts.transpileModule(await readFile(`lib/webrtc/${name}.ts`, "utf8"), {
      compilerOptions: {
        target: ts.ScriptTarget.ES2022,
        module: ts.ModuleKind.ES2022,
      },
    }).outputText,
  );
const server = createServer((request, response) => {
  response.setHeader(
    "Content-Type",
    modules.has(request.url) ? "text/javascript" : "text/html",
  );
  response.end(
    modules.get(request.url) ??
      '<video id="a" autoplay muted playsinline></video><video id="b" autoplay muted playsinline></video>',
  );
});
await new Promise((resolve) => server.listen(3111, "127.0.0.1", resolve));
const browser = await chromium.launch({
  headless: true,
  args: [
    "--no-sandbox",
    "--use-fake-device-for-media-stream",
    "--use-fake-ui-for-media-stream",
    "--autoplay-policy=no-user-gesture-required",
  ],
});
try {
  for (const compatibility of [false, true]) {
    const context = await browser.newContext({
      permissions: ["camera", "microphone"],
    });
    const page = await context.newPage();
    await page.goto("http://127.0.0.1:3111");
    await page.evaluate(async (compatibility) => {
      if (compatibility) {
        window.MediaStreamTrackProcessor = undefined;
        window.MediaStreamTrackGenerator = undefined;
      }
      const { attachCallMedia, receiveTrack } = await import("/call-media.js");
      const { processCamera } = await import("/processed-camera.js");
      const streams = await Promise.all(
        [0, 1].map(() =>
          navigator.mediaDevices.getUserMedia({
            audio: true,
            video: { width: 1920, height: 1080, frameRate: 30 },
          }),
        ),
      );
      const captures = await Promise.all(
        streams.map((raw) => processCamera(raw, () => true)),
      );
      const media = streams.map(
        (raw, i) =>
          new MediaStream([...raw.getAudioTracks(), captures[i].track]),
      );
      const a = new RTCPeerConnection(),
        b = new RTCPeerConnection();
      const ra = new MediaStream(),
        rb = new MediaStream();
      a.ontrack = (event) =>
        receiveTrack(ra, event, document.querySelector("#a"));
      b.ontrack = (event) =>
        receiveTrack(rb, event, document.querySelector("#b"));
      const gather = async (pc, description) => {
        const complete = new Promise((resolve) => {
          if (pc.iceGatheringState === "complete") resolve();
          else
            pc.addEventListener("icegatheringstatechange", () => {
              if (pc.iceGatheringState === "complete") resolve();
            });
        });
        await pc.setLocalDescription(description);
        await complete;
        return pc.localDescription;
      };
      const senderA = await attachCallMedia(a, media[0], false);
      await b.setRemoteDescription(await gather(a, await a.createOffer()));
      const senderB = await attachCallMedia(b, media[1], true);
      await a.setRemoteDescription(await gather(b, await b.createAnswer()));
      window.fixture = { a, b, media, captures, senderA, senderB, streams };
    }, compatibility);
    await expect
      .poll(
        () =>
          page.evaluate(async () => {
            const stats = await Promise.all(
              [window.fixture.a, window.fixture.b].map(async (pc) => {
                const values = [...(await pc.getStats()).values()];
                return values
                  .filter((s) => s.type === "inbound-rtp" && s.kind === "video")
                  .reduce((n, s) => n + (s.framesDecoded || 0), 0);
              }),
            );
            return Math.min(...stats);
          }),
        { timeout: 30000 },
      )
      .toBeGreaterThan(20);
    await expect
      .poll(() =>
        page.evaluate(() =>
          [...document.querySelectorAll("video")].every(
            (v) => v.videoWidth > 0 && !v.paused,
          ),
        ),
      )
      .toBe(true);
    await page.evaluate(async () => {
      await window.fixture.senderB.replaceTrack(null);
    });
    await page.evaluate(async () => {
      await window.fixture.senderB.replaceTrack(
        window.fixture.media[1].getVideoTracks()[0],
      );
    });
    const before = await page.evaluate(
      () =>
        document.querySelector("#a").getVideoPlaybackQuality().totalVideoFrames,
    );
    await expect
      .poll(
        () =>
          page.evaluate(
            () =>
              document.querySelector("#a").getVideoPlaybackQuality()
                .totalVideoFrames,
          ),
        { timeout: 10000 },
      )
      .toBeGreaterThan(before + 15);
    // A busy UI must not leave a multi-second queue of old frames afterward.
    await page.evaluate(() => {
      const until = performance.now() + 700;
      while (performance.now() < until) {
        /* intentional CPU stall */
      }
    });
    const latency = await page.evaluate(async () => {
      const video = document.querySelector("#a");
      return new Promise((resolve) =>
        video.requestVideoFrameCallback((now, metadata) =>
          resolve(metadata.captureTime ? now - metadata.captureTime : null),
        ),
      );
    });
    if (latency !== null) expect(latency).toBeLessThan(2000);
    console.log(
      `PASS: ${compatibility ? "frame callback fallback" : "bounded timestamp-preserving pipeline"}; both peers receive video, camera resumes, post-stall latency=${latency ?? "not exposed"}ms`,
    );
    await page.evaluate(() => {
      window.fixture.a.close();
      window.fixture.b.close();
      window.fixture.captures.forEach((c) => c.stop());
      window.fixture.streams.forEach((s) =>
        s.getTracks().forEach((t) => t.stop()),
      );
    });
    await context.close();
  }
} finally {
  await browser.close();
  server.close();
}
