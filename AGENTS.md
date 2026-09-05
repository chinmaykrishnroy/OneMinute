# Project instructions

- Product name and current branding: OneMinute. Keep neutral internal module, schema and infrastructure names so branding can change.
- Build in the milestone order documented in README.md and docs/architecture. Preserve PostgreSQL durability, Redis distributed runtime state, Go application identity and browser-to-browser media.
- Run application builds, containers and tests on `ssh llm-04`, in `/home/roy/OneMinute`. Do not run the application stack or test workloads on the user's local workstation. Local source edits and Git operations are fine.
- Remote Docker commands require `sudo -n`. Use `compose.test.yaml` for Go race/integration and frontend verification without installing host toolchains.
- Copy source changes to the remote workspace without .env, .git, node_modules, .next, caches or artifacts. Preserve the remote .env and durable data.
- Inspect an older deployment before cleaning it; do not remove unrelated services, model files, images or volumes.
- Commit and push coherent verified checkpoints to `origin/main` at `https://github.com/chinmaykrishnroy/OneMinute.git`.
- Do not add AI attribution or co-author trailers to commits, code or documentation.
- Keep verification claims precise. Record unexecuted browser/hardware/NAT tests in docs/verification.md.
