# Authentication boundary
Status: design for Milestone 2; no public login or session endpoints exist yet.

Go is the application identity authority. The browser sends a Google identity credential to Go; Go verifies the signature, issuer, audience, expiry and nonce before linking the stable provider subject. Email is never the identity key. Never expose provider account information beyond the chosen public profile.

Use a Secure, HttpOnly, SameSite cookie for an opaque application session. Store only a cryptographic hash of the random session/refresh secret in PostgreSQL with expiry/revocation and rotation metadata. An opaque session checked in PostgreSQL works with disposable compute; short-lived access credentials can be added if measurements justify them. Returning users reuse their application session rather than repeating Google OAuth.

State-changing HTTP endpoints require strict Origin checks and CSRF protection appropriate to the cookie deployment. WebSocket handshakes validate the exact configured origin and the Go session. Do not place tokens in URLs, localStorage or logs.

The upcoming networking lab must be explicitly enabled only in development, use short-lived room capabilities, and be forbidden in production. It is not application authentication.
