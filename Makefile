.PHONY: init up down logs test check smoke migrate
init:
	node scripts/init-env.mjs
up:
	docker compose up --build --detach --wait --wait-timeout 180
down:
	docker compose down
logs:
	docker compose logs --tail 100 --follow
test:
	go test ./...
check: test
	go vet ./...
	cd apps/web && npm run typecheck && npm run lint && npm run build
smoke:
	node scripts/smoke.mjs
migrate:
	docker compose run --rm migrate
