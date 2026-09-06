// Runs against an isolated build with synthetic identities and media. Every
// application API and WebSocket is intercepted; no production accounts/data.
import { chromium, expect } from "@playwright/test";
import { spawn } from "node:child_process";
import { mkdir } from "node:fs/promises";
const origin = "http://127.0.0.1:3101";
const server = spawn(
  process.execPath,
  ["node_modules/next/dist/bin/next", "start", "-p", "3101"],
  { stdio: "inherit" },
);
const browser = await chromium.launch({
  args: [
    "--use-fake-device-for-media-stream",
    "--use-fake-ui-for-media-stream",
    "--autoplay-policy=no-user-gesture-required",
  ],
});
const sockets = new Map();
const settings = {
  theme: "system",
  notifications: true,
  typing: true,
  readReceipts: true,
};
let call;
const people = { a: "Alex Morgan", b: "Sam Rivers" };
const connectionId = "11111111-1111-4111-8111-111111111111";
const emit = (id, value) => sockets.get(id)?.send(JSON.stringify(value));
const errors = [];
async function fixture(id) {
  const context = await browser.newContext({
    permissions: ["camera", "microphone"],
    viewport: { width: 393, height: 852 },
    colorScheme: "light",
  });
  const other = id === "a" ? "b" : "a";
  await context.addInitScript(() => {
    window.testPeers = [];
    const Native = window.RTCPeerConnection;
    window.RTCPeerConnection = class extends Native {
      constructor(...args) {
        super(...args);
        window.testPeers.push(this);
      }
    };
  });
  await context.routeWebSocket("**/v1/events/ws", (socket) => {
    sockets.set(id, socket);
    socket.send(
      JSON.stringify({
        type: "ready",
        user: { id },
        ice: { iceServers: [] },
        settings,
      }),
    );
  });
  await context.routeWebSocket("**/v1/discovery/ws", (socket) =>
    socket.send(
      JSON.stringify({
        version: 1,
        type: "connection.ready",
        payload: { ice: { iceServers: [] } },
      }),
    ),
  );
  await context.route("**/v1/**", async (route) => {
    const request = route.request(),
      path = new URL(request.url()).pathname;
    let body = {};
    try {
      body = request.postDataJSON() || {};
    } catch {}
    const profile = {
      id,
      displayName: people[id],
      avatarUrl: "",
      bio: "Good conversations, music and weekend walks.",
      countryCode: "IN",
      interests: ["music", "technology"],
      languages: ["en"],
      discoveryIntent: "new_friends",
    };
    const connections = [
      {
        id: connectionId,
        preview: "That was a great conversation. Same time tomorrow?",
        unread: 1,
        person: { id: other, displayName: people[other], avatarUrl: "" },
      },
    ];
    let result;
    if (path === "/v1/profile") result = profile;
    else if (path === "/v1/blocks") result = { blocks: [] };
    else if (path === "/v1/conversations" || path === "/v1/connections")
      result = { connections };
    else if (path === "/v1/settings")
      result = request.method() === "PUT" ? body : settings;
    else if (path === "/v1/notifications") result = [];
    else if (path === "/v1/moments")
      result = [
        {
          id: "moment-1",
          userId: other,
          name: people[other],
          avatarUrl: "",
          body: "Found a tiny bookshop with a whole shelf of travel journals. Where would you go next?",
          tone: "lilac",
          createdAt: new Date().toISOString(),
          expiresAt: new Date(Date.now() + 86400000).toISOString(),
          connectionId,
        },
      ];
    else if (path.endsWith("/messages"))
      result = {
        messages: [
          {
            id: 1,
            clientId: "fixture-1",
            senderId: other,
            body: "That was a great conversation. Same time tomorrow?",
            createdAt: new Date().toISOString(),
          },
        ],
        readId: 0,
        deliveredId: 0,
      };
    else if (path.endsWith("/calls") && path.startsWith("/v1/connections/")) {
      call = {
        id: crypto.randomUUID(),
        connectionId,
        caller: id,
        callee: other,
        name: people[id],
        video: body.video,
      };
      result = call;
      emit(other, { type: "call.invited", call });
    } else if (path.startsWith("/v1/calls/")) {
      result = { ok: true };
      const event = { type: `call.${body.type}`, call, payload: body.payload };
      if (
        body.type === "accept" ||
        body.type === "end" ||
        body.type === "decline"
      ) {
        emit("a", event);
        emit("b", event);
      } else if (body.type !== "heartbeat") emit(other, event);
    } else if (
      path.endsWith("/receipt") ||
      path.endsWith("/typing") ||
      path.endsWith("/read")
    )
      result = { ok: true };
    else throw new Error(`Unmocked application request ${path}`);
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(result),
    });
  });
  const page = await context.newPage();
  page.on("pageerror", (error) => errors.push(String(error)));
  return page;
}
async function videoFlows(page) {
  await expect
    .poll(
      () =>
        page.evaluate(async () => {
          const pc = window.testPeers.at(-1);
          if (!pc) return false;
          const stats = [...(await pc.getStats()).values()];
          const video = document.querySelector(".call-remote-video");
          return (
            stats.some(
              (s) =>
                s.type === "inbound-rtp" &&
                s.kind === "video" &&
                s.framesDecoded > 15,
            ) &&
            video.videoWidth > 0 &&
            !video.paused
          );
        }),
      { timeout: 25000 },
    )
    .toBe(true);
}
try {
  for (let n = 0; n < 100; n++) {
    try {
      if ((await fetch(origin + "/healthz")).ok) break;
    } catch {}
    await new Promise((r) => setTimeout(r, 250));
  }
  await mkdir("/artifacts", { recursive: true });
  const a = await fixture("a"),
    b = await fixture("b");
  for (const [width, height, label] of [
    [393, 852, "phone"],
    [820, 1180, "tablet"],
    [1440, 1000, "desktop"],
  ]) {
    await a.setViewportSize({ width, height });
    for (const [route, title] of [
      ["discover", "Discover"],
      ["messages", "Messages"],
      ["moments", "Moments"],
      ["profile", "You"],
      ["settings", "Settings"],
    ]) {
      await a.goto(`${origin}/app/${route}`);
      await expect(a.locator(".header-heading h1")).toHaveText(title);
      await expect(a).toHaveTitle(new RegExp(`${title}.*OneMinute`));
      await expect
        .poll(() =>
          a
            .locator(".product-header")
            .evaluate((el) => Math.round(el.getBoundingClientRect().height)),
        )
        .toBe(72);
      await expect
        .poll(() =>
          a.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
        )
        .toBe(true);
      await expect(
        a.getByRole("button", { name: "Go back", exact: true }),
      ).toBeVisible();
      if (route === "discover") {
        await expect(
          a.getByRole("button", { name: /Notifications/ }),
        ).toBeVisible();
      }
      await a.screenshot({
        path: `/artifacts/${label}-${route}.png`,
        fullPage: true,
      });
    }
  }
  await a.emulateMedia({ colorScheme: "dark" });
  await a.goto(`${origin}/app/moments`);
  await expect(a.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect
    .poll(() =>
      a.locator("html").evaluate((el) => getComputedStyle(el).backgroundColor),
    )
    .toBe("rgb(3, 16, 9)");
  await a.screenshot({ path: "/artifacts/desktop-dark.png", fullPage: true });
  await a.emulateMedia({ colorScheme: "light" });
  await a.setViewportSize({ width: 393, height: 852 });
  await a.goto(`${origin}/app/profile`);
  await expect(a.locator("#preferences")).toBeVisible();
  await expect
    .poll(() =>
      a.evaluate(
        () => getComputedStyle(document.documentElement).scrollbarWidth,
      ),
    )
    .toBe("none");
  await a.mouse.move(200, 400);
  await a.mouse.wheel(0, 500);
  await expect.poll(() => a.evaluate(() => scrollY)).toBeGreaterThan(100);
  await a.evaluate(() => scrollTo(0, 0));
  await a.keyboard.press("PageDown");
  await expect.poll(() => a.evaluate(() => scrollY)).toBeGreaterThan(100);
  await a.screenshot({ path: "/artifacts/phone-scroll.png" });
  await a.goto(`${origin}/app/messages`);
  await a.getByRole("button", { name: /Sam Rivers/ }).click();
  await expect(a.locator(".header-heading h1")).toHaveText("Sam Rivers");
  const draft = a.locator("#message-draft");
  const send = a.getByRole("button", { name: "Send", exact: true });
  await draft.fill("One line");
  await expect
    .poll(() => draft.evaluate((element) => Math.round(element.getBoundingClientRect().height)))
    .toBe(await send.evaluate((element) => Math.round(element.getBoundingClientRect().height)));
  await draft.fill(Array.from({ length: 8 }, (_, index) => `line ${index + 1}`).join("\n"));
  await expect(a.locator(".message-overflow-cue")).toBeVisible();
  await expect
    .poll(() => draft.evaluate((element) => element.getBoundingClientRect().height))
    .toBeLessThan(170);
  await a.getByRole("button", { name: "Go back", exact: true }).click();
  await expect(a.locator(".header-heading h1")).toHaveText("Messages");
  await a.getByRole("button", { name: /Sam Rivers/ }).click();
  await b.goto(`${origin}/app/messages?connection=${connectionId}`);
  await a.getByRole("button", { name: "Start video call" }).click();
  await b.getByRole("button", { name: "Accept video call" }).click();
  await Promise.all([videoFlows(a), videoFlows(b)]);
  await b.getByRole("button", { name: "Camera off", exact: true }).click();
  await expect(a.getByText("Camera is off", { exact: true })).toBeVisible();
  const before = await a.evaluate(
    async () =>
      [...(await window.testPeers.at(-1).getStats()).values()].find(
        (s) => s.type === "inbound-rtp" && s.kind === "video",
      ).framesDecoded,
  );
  await b.getByRole("button", { name: "Camera on", exact: true }).click();
  await expect
    .poll(
      () =>
        a.evaluate(
          async () =>
            [...(await window.testPeers.at(-1).getStats()).values()].find(
              (s) => s.type === "inbound-rtp" && s.kind === "video",
            ).framesDecoded,
        ),
      { timeout: 20000 },
    )
    .toBeGreaterThan(before + 15);
  await a.screenshot({ path: "/artifacts/phone-call.png" });
  await a.getByRole("button", { name: "End call", exact: true }).click();
  await b.getByRole("button", { name: "Close", exact: true }).click();
  await a.getByRole("button", { name: "Start audio call" }).click();
  await b.getByRole("button", { name: "Accept audio call" }).click();
  await expect(a.getByText("Connected", { exact: true })).toBeVisible({
    timeout: 20000,
  });
  await a.getByRole("button", { name: "Camera on", exact: true }).click();
  await b.getByRole("button", { name: "Camera on", exact: true }).click();
  await Promise.all([videoFlows(a), videoFlows(b)]);
  await a.getByRole("button", { name: "End call", exact: true }).click();
  expect(errors).toEqual([]);
  console.log(
    "PASS: responsive shell, titles, back navigation, system dark theme, real call UI two-way video, camera resume and audio-to-video upgrade",
  );
} finally {
  await browser.close();
  server.kill("SIGTERM");
}
