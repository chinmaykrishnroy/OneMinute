# OneMinute product navigation

OneMinute helps people find a person worth keeping in touch with. Discover is the entry point after onboarding; messaging and private updates serve the relationships created there. Preserve `DISCOVER → TALK → EXTEND → CONNECT → KEEP` and the approved soft neobrutalist + modern minimal social UI hybrid.

## Primary destinations

| Destination | Purpose and layout | Header action |
| --- | --- | --- |
| Discover | One clear invitation to meet. Saved preferences appear as context. During an encounter use equal participant tiles. | Discovery preferences |
| Messages | A searchable list of mutual connections and conversation previews. Opening a person shows their conversation; desktop retains the list beside it. | Connections in the inbox; audio and video call in a conversation |
| Moments | Up to three short text updates, each visible for 24 hours to connections present when shared. A reply opens the existing private conversation. | Create moment |
| You | Identity and editable profile/discovery preferences. Connection and inbox shortcuts. | Settings & activity drawer |

Mobile uses four icons with accessible names and separated touch areas. Tablet uses a compact rail. Desktop uses a labeled rail and branding. Destinations share the same 72-pixel sticky header, title typography and back control; only the contextual action changes. Back uses navigation history, falling back to Discover when none exists. Browser titles describe each destination followed by OneMinute. Old `/app/posts` links redirect to Moments.

## Account and secondary screens

You → Settings & activity is the single account hub:

- Activity & notifications: durable connection, message and call events with mark-as-read. A single activity dot on You replaces repeated header bells.
- App settings: System default / Light / Dark appearance, notification preference, browser permission, typing visibility and read receipts. Settings persist per account. Browser alerts currently require the site to remain open; closed-app push is a separate delivery milestone.
- Edit profile and Discovery preferences: anchors in You. First-time account setup remains a dedicated onboarding flow.
- Connections: manage relationships, start a conversation, remove, report or block.
- Blocked people: inspect and unblock within the account drawer. Unblocking does not restore a relationship.
- Log out: end the current application session.

Connected calls are global overlays so navigation does not accidentally terminate a call. Remote video fills the available space while preserving its aspect ratio; the local preview is movable. Discover remains a separate encounter flow with a timer, private Extend/Connect votes and temporary DataChannel chat.

## Surface and content rules

Use spacing, typography and dividers to organize pages. Reserve bounded surfaces for controls, message bubbles, dialogs, camera tiles and selected navigation. Do not nest a card around every feature. Preserve visible focus, at least 44-pixel controls, accessible icon names, dark-mode contrast and mobile safe-area spacing.

Moments is a conversation prompt: no public audience, discovery ranking boost, likes, followers, reposts or call recordings. Audience membership is snapshotted at publication and checked against the current relationship and blocks when read. Expired content is excluded by the server immediately and removed by a periodic worker; backups follow their own retention. New connections cannot view an earlier moment. Deleting a moment removes its audience records.

## Implementation sequence

1. Stabilize two-way camera negotiation and bounded frame processing; regress actual call UI and camera toggles on the remote test machine.
2. Apply the shared shell, flat inbox, single account hub and private Moments slice with privacy/expiry tests.
3. Complete Milestone 5.5 private object storage and authorized attachment transfer before Milestone 6 intelligent discovery.
4. Add closed-app Web Push/native push, physical device acceptance and Milestone 7 moderation/operational hardening through explicit acceptance boundaries.

This guide is the navigation source of truth for subsequent web and native work. Keep the roadmap and verification ledger explicit about implemented, automated-tested and physically accepted behavior.
