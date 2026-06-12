# Adding a Feature

A checklist for adding a new feature page or backend capability. The layered
architecture (see `architecture.md`) makes most additions mechanical.

## Frontend: a new page

1. **Route** — add `app/(dashboard)/<feature>/page.tsx`. It is automatically
   wrapped in the AppShell.
2. **Nav entry** — add one item to `config/nav.ts` (`mainNav` or `supportNav`).
   That is the only wiring the sidebar and command palette need.
3. **Feature folder** — put components in `features/<feature>/`, composed from
   `components/shared/` and `components/ui/`. Do not re-implement a stat card,
   table, badge, or chart — reuse the shared primitives.
4. **Data** — add typed client functions in `lib/api/client.ts` (and DTOs in
   `lib/api/types.ts`) for any new endpoint, then a hook in `lib/hooks/` using
   TanStack Query. If the data is live, patch the cache from the WS manager
   rather than holding a second store.
5. **States** — every widget renders explicit loading (skeleton), empty
   (`EmptyState`), and error states.
6. **Verify** — `npm run typecheck`, `npm run lint`, `npm run build`.

## Backend: a new endpoint or capability

1. **Domain** — add types and any sentinel errors to `internal/domain` (it
   depends on nothing).
2. **Migration** — if you need a table or column, append a numbered entry to
   `internal/storage/sqlite/migrations.go`. Never edit or reorder a shipped
   migration. Add the query methods to the sqlite `Store`.
3. **Service** — put business logic in `internal/service/<area>`. Define the
   narrow repository interface the service needs in that package (the `Store`
   satisfies it). Inject the `Clock` for time-dependent logic so it is testable.
4. **Transport** — add a thin handler in `internal/transport/http` (decode ->
   service -> encode). Map domain errors through `fail`/`writeError`; do not
   write status codes ad hoc. Register the route in `router.go` under
   `/api/v1`; if it mutates state, wrap it in `RequireAuth`. Add a legacy
   `/api` alias only if an existing client needs it.
5. **Wiring** — construct the service in `cmd/server/main.go` and inject it.
6. **Tests** — table-driven service tests with a fake repo + fake clock, and an
   httptest handler test for the endpoint.
7. **Docs** — if the endpoint is part of the public contract, add it to the API
   table in `server-monitor-go/README.md` and `PRD.md`.
8. **Verify** — `go vet ./...`, `gofmt -l .`, `go test -race ./...`,
   `scripts/build.sh`.

## Invariants to preserve

- Deployed agents cannot be updated atomically: keep the ingest DTO and the
  legacy `/api/...` paths compatible.
- An existing `servers.db` must upgrade in place (versioned migrations only).
- Anonymous viewers never see real public or Tailscale IPs (REST or WS); admin
  JWT reveals them. Masking changes go in `internal/masking` only.
