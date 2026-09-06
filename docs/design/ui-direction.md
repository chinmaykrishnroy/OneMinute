# OneMinute UI direction

Owner preference, recorded 2026-09-05:

> **soft neobrutalist + modern minimal social UI hybrid**

This is the persistent visual direction for OneMinute. Keep the product name in the branding module and keep internal identifiers neutral.

## Visual principles

- Use clear borders, gently rounded cards and controls, and short offset shadows to give interactive elements definition. Keep these details restrained.
- Pair warm neutral surfaces with a small set of soft accent colors. Mint, lilac and pale yellow are possible starting points, not a finalized palette.
- Use clean typography, generous spacing and a quiet page structure. Make people and live video the focus.
- Use familiar social patterns: compact avatars, interest pills, clear call controls and lightweight profile cards.
- Give each screen a clear primary action. During a call, keep the timer, Next and Extend easy to understand and reach.
- Preserve readable contrast, visible keyboard focus, reduced-motion support and touch targets of at least 44 CSS pixels. State changes must be understandable without color alone.

Avoid heavy decoration, dense dashboard layouts and oversized shadows that compete with faces or controls. Do not add fabricated social activity or nonfunctional controls for appearance.

## Implementation timing

Build functional vertical slices in the agreed milestone order. The current networking lab is a diagnostic screen, not the finished social experience. The shared stylesheet now establishes the first visual foundation: warm neutral surfaces, mint and lilac controls, rounded borders, compact shadows and visible keyboard focus. The complete social screen design remains part of future feature slices.

Discovery is conversation-first. Before a call, show minimal identity plus useful shared context such as compatible current intent and a few shared interests. Do not use swipe cards, photo-heavy profile judgment or public-feed patterns. Richer profile detail becomes appropriate after mutual Connect. Keep private one-sided Extend and Connect choices visually private until a mutual result exists.

## Encounter layout across screens

Use square video surfaces as the default visual frame, with `object-fit: cover` and face-centered composition. Provide a later “Fit full frame” option for users who prefer uncropped video. Keep the remote participant visually primary even when both panes have equal layout area.

- Phone portrait: remote video occupies the upper half and the local preview the lower half. Name, status, timer and call controls sit in a readable scrim over the bottom of the local preview. Leave enough safe-area space for a persistent bottom tab navigator that can carry into a future mobile app.
- Tablet portrait: use a stacked layout with a modestly larger remote pane. Tablet landscape: switch to a side-by-side layout while keeping touch-sized controls and avoiding desktop-density UI.
- Desktop and wide screens: place the remote participant in the left half and the local participant in the right half. Center square video surfaces within each pane rather than stretching video to the viewport shape.
- Very short or unusually wide screens: preserve both faces and controls by reducing gaps and moving secondary metadata into an overlay; never let primary call controls leave the viewport.

The local preview has a camera-settings button that opens an accessible overlay. The encounter defaults to a selfie-style mirrored camera and input-device selection. The preview and transmitted track use the same canvas-processed video, so changing the mirror setting changes what both people see. Advanced blur, effects and playful filters may later extend this processed-video-track boundary (for example WebGL/MediaPipe), with an easy “None” state and performance fallback.

## App icon

The canonical mark is `apps/web/app/icon.svg`: a text-free 1:00 clock with the product palette and geometry that remains clear at 32×32. `scripts/generate-icons.py` generates the PNG, Apple icon and multi-size ICO derivatives from the same geometry. Navigation titles use the root template `%s · OneMinute`, with `OneMinute` alone on the home page.

