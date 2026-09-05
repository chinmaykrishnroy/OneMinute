# Authentication boundary
Status: Milestone 2 implemented. Public Google sign-in, application sessions, current-user lookup and logout endpoints exist.

Go is the application identity authority. The browser sends a Google identity credential to Go; Go verifies the signature, issuer, audience, expiry and nonce before linking the stable provider subject. Email is never the identity key. Never expose provider account information beyond the chosen public profile.

The browser uses Google Identity Services to obtain an ID token with a server-issued nonce. Go validates the Google signature, issuer, audience, expiry and nonce, then links the stable `sub` claim. Basic display name, avatar and email-verification signal initialize application identity; email is not stored or used as the identity key. This flow needs the OAuth client ID and does not use an OAuth client secret.

Use a Secure, HttpOnly, SameSite cookie for an opaque application session. Store only a cryptographic hash of the random session secret in PostgreSQL with expiry and revocation. An opaque session checked in PostgreSQL works with disposable compute; short-lived access credentials and rotation metadata can be added if measurements justify them. Returning users reuse their application session rather than repeating Google login.

State-changing HTTP endpoints require strict Origin checks and CSRF protection appropriate to the cookie deployment. Milestone 3 WebSocket handshakes must validate the exact configured origin and the Go session before presence or queue state is created. Reconnects repeat identity and match-membership checks. Do not place tokens in URLs, localStorage or logs.

The networking lab is explicitly enabled only in development, uses short-lived room capabilities, and is forbidden in production. It is not application authentication.
