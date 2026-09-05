# Media and object storage

Status: architecture for Milestones 5.5 and 7; no user-media upload path is implemented.

Persistent connection media uses an S3-compatible object-storage interface. MinIO is the first deployment implementation, not a business-logic dependency. Buckets remain private. PostgreSQL stores ownership, object key, MIME type, byte size, dimensions/duration, lifecycle/moderation status and timestamps; it does not store binary image, audio or video payloads.

The planned upload flow is browser → Go authorization → short-lived presigned URL → direct object-store upload. Go first validates the signed-in user, active connection, allowed type, size and quota. The client-supplied MIME type and metadata are hints only. Finalization verifies magic bytes, extracts metadata and moves the database record from pending to usable. Failed, abandoned and unreferenced objects need scheduled cleanup.

Downloads use short-lived signed URLs or an authenticated authorization path. A message may reference a ready media object only when the sender owns it and belongs to the active conversation. Block and connection removal policies apply before upload authorization and download access.

Hardening includes bounded sizes, MIME allowlists, image dimensions, audio/video duration, per-user quotas, abuse reports, moderation state and object cleanup. Provider-specific SDK details stay in the storage adapter so MinIO can be replaced by AWS S3, Cloudflare R2 or another compatible service without changing messaging rules.
