# OneMinute UI direction

See [product navigation](product-navigation.md) for the current tab, page, action and settings map. Use it as the shared web/native UX guide.

Owner preference, recorded 2026-09-05:

> **soft neobrutalist + modern minimal social UI hybrid**

This is the persistent visual direction for OneMinute. Keep the product name in the branding module and keep internal identifiers neutral.

Owner refinements, 2026-09-07: dark page backgrounds use `#031009`; keep surfaces lighter for separation. Hide visible scrollbars throughout the app while preserving natural scrolling and keyboard access. Do not reserve a scrollbar gutter.

## Visual principles

- Use clear borders, gently rounded cards and controls, and short offset shadows to give interactive elements definition. Keep these details restrained.
- Pair warm neutral surfaces with a small set of soft accent colors. Mint, lilac and pale yellow are possible starting points, not a finalized palette.
- Use clean typography, generous spacing and a quiet page structure. Make people and live video the focus.
- Use familiar social patterns: compact avatars, interest pills, clear call controls and lightweight profile cards.
- Give each screen a clear primary action. During a call, keep the timer, Next, Extend, Connect and safety actions easy to understand and reach.
- Preserve readable contrast, visible keyboard focus, reduced-motion support and touch targets of at least 44 CSS pixels. State changes must be understandable without color alone.

Avoid heavy decoration, dense dashboard layouts and oversized shadows that compete with faces or controls. Do not add fabricated social activity or nonfunctional controls for appearance.

## Implementation timing

Build functional vertical slices in the agreed milestone order. The networking lab remains a diagnostic screen. Product screens use warm neutral surfaces, mint and lilac controls, rounded borders, compact shadows and visible keyboard focus. Discovery, encounter, profile and connection screens share this system across desktop, tablet and bottom-tab mobile layouts.

Discovery is conversation-first. Before a call, show minimal identity plus useful shared context such as compatible current intent and a few shared interests. Do not use swipe cards, photo-heavy profile judgment or public-feed patterns. Richer profile detail becomes appropriate after mutual Connect. Keep private one-sided Extend and Connect choices visually private until a mutual result exists.

The app shell uses four future-mobile destinations: Discover, Messages, Posts and You. Discover is the ready-to-meet home after profile setup. You is the account hub for identity, discovery preferences, connections, blocked people and settings. Messages and Posts may show clear coming-soon states until their milestones are implemented; they should not fabricate activity. Use icons with accessible labels in compact mobile navigation, a labeled rail on desktop, and a compact icon rail on tablet.

## Encounter layout across screens

Use square video surfaces as the default visual frame, with `object-fit: cover` and face-centered composition. Provide a later “Fit full frame” option for users who prefer uncropped video. Keep the remote participant visually primary even when both panes have equal layout area.

- Phone portrait: remote video occupies the upper half and the local preview the lower half. Name, status, timer and call controls sit in a readable scrim over the bottom of the local preview. Leave enough safe-area space for a persistent bottom tab navigator that can carry into a future mobile app.
- Tablet portrait: use a stacked layout with a modestly larger remote pane. Tablet landscape: switch to a side-by-side layout while keeping touch-sized controls and avoiding desktop-density UI.
- Desktop and wide screens: place the remote participant in the left half and the local participant in the right half. Center square video surfaces within each pane rather than stretching video to the viewport shape.
- Very short or unusually wide screens: preserve both faces and controls by reducing gaps and moving secondary metadata into an overlay; never let primary call controls leave the viewport.

The local preview has a camera-settings button that opens an accessible overlay. The encounter defaults to a selfie-style mirrored camera and input-device selection. The preview and transmitted track use the same canvas-processed video, so changing the mirror setting changes what both people see. Advanced blur, effects and playful filters may later extend this processed-video-track boundary (for example WebGL/MediaPipe), with an easy “None” state and performance fallback.

Request an ideal 1080p camera source without a minimum resolution. WebRTC senders use balanced degradation so congestion control can reduce bitrate and spatial resolution as network conditions change instead of locking the call to one fixed quality. Reconnect recovery rebuilds both peers' `RTCPeerConnection` while reusing any still-live local media pipeline; the stable server-provided offerer role starts renegotiation.

## Onboarding, communication and appearance

First-time setup is a dedicated three-step welcome flow, never a redirect into the You editor. Explain that users can change these details in You later. Keep every destination's document title and visible header meaningful. Headers remain anchored while content scrolls. Mobile hover and selected states have a visible gap between tabs.

Use a conversation list and chat pane on desktop/tablet, and a single-pane inbox with a back control on phones. Show real last-message previews, unread state, typing and delivered/read status. Connected-user calls have their own full-screen remote-video layout with a movable, aspect-aware self-preview and reachable audio/video/end controls. Use full-frame containment when a participant changes camera orientation.

Appearance defaults to System and offers Light/Dark overrides. Use shared ink, muted, paper, canvas, accent, line and shadow tokens for readable surfaces in both modes. User settings include notification, typing and read-receipt choices; notification permissions are requested only through the enable button. The header notification center uses durable, genuine activity.

## App icon

The canonical mark is `apps/web/app/icon.svg`: a text-free 1:00 clock with the product palette and geometry that remains clear at 32×32. `scripts/generate-icons.py` generates the PNG, Apple icon and multi-size ICO derivatives from the same geometry. Navigation titles use the root template `%s · OneMinute`, with `OneMinute` alone on the home page.

