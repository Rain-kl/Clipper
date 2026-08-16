# Clipper ← Wavelet Upstream Sync Design

**Date:** 2026-08-16  
**Status:** Approved for implementation planning  
**Product:** Clipper (business) on Wavelet (scaffold)  
**Wavelet pin at design time:** `main` @ `284eec5` (`chore: guideline`). Implementation fetches current `upstream/main` at sync time; if Wavelet `main` has moved, take that tip and apply the same rules.

## 1. Goal

Reconnect Clipper to Wavelet as a fetch-only git upstream so:

1. Clipper's framework tree matches current Wavelet `main` (package layout, i18n, CI, skills).
2. Clipper's item-capture business keeps working on that framework.
3. Later Wavelet updates are `git fetch upstream && git merge upstream/main` inside the **Clipper** repo.

Clipper and Wavelet stay independent GitHub repositories. The Wavelet repo is never written to (no branch, no commit, no push).

### In scope

- Add `upstream` remote on Clipper; rebuild Clipper history so Wavelet `main` is an ancestor.
- Adopt Wavelet `internal/infra`, `internal/platform`, `internal/shared`, `.agents`, frontend i18n infrastructure, CI/Docker/Makefile.
- Re-home item domain (Go, SQL, frontend, i18n) onto the new tree.
- Bilingual UI for Clip / Review / Search / Library / Vault and their sidebar entries.
- Keep Clipper product identity (name, banner, swagger title, README).

### Out of scope

- Any write to the Wavelet working tree or `Rain-kl/Wavelet` remotes.
- Backend API `error_msg` / email / push localization.
- New Clipper features (tags, AI classify, providers, vault encryption).
- Changing the Go module path (stays `github.com/Rain-kl/Wavelet`).
- Re-translating Wavelet framework pages that already have keys.

## 2. Current state

Clipper is not a git fork. Its root `6aac943` (`Initial commit`) has the **same tree** as Wavelet `5fd056f` (`chore(release): bump version to v1.4.1`, 2026-07-13). After that:

- Clipper added item domain, Clipper branding, and some overlapping 7/22 polish (biome / Makefile / AGENTS).
- Wavelet added the 7/24 `internal/` regroup (`fbbb750`), next-intl (`1625cfb`), CI/Docker, `.agent` → `.agents`, and push webhook logging.

There is no shared git ancestor today, so a naive merge treats almost every file as added on both sides.

## 3. Architecture

```
Wavelet GitHub (read-only)
        │ fetch only
        ▼
Clipper remote `upstream`
        │
        ▼
Clipper branch `sync/wavelet-upstream`  ← starts at upstream/main
        │  port item + branding + item i18n
        ▼
Clipper `main` (after backup tag + verify)
```

**Chosen method:** In the Clipper repo only, fetch Wavelet and create `sync/wavelet-upstream` from `upstream/main`. Copy Clipper-only business onto that Wavelet tree. Do **not** rebase Clipper's 7/22 polish commits (they collide with Wavelet's later biome/i18n/structure work).

**Rejected:** rebase Clipper commits onto v1.4.1 then merge (double conflict: polish + import rewrite). Graft/`git replace` on existing Clipper `main` (local-only, still rewrites history, worse future merge).

### 3.1 Git operations (Clipper repo only)

1. `git remote add upstream <Wavelet fetch URL>` if missing. Never set a push URL that can update Wavelet (use `git remote set-url --push upstream DISABLED` or leave push unused).
2. `git fetch upstream`
3. `git tag backup/main-pre-wavelet-sync main` so the pre-sync Clipper history remains reachable.
4. `git checkout -b sync/wavelet-upstream upstream/main`
5. Port business (sections 4–6) as Clipper commits on this branch.
6. After gates pass: fast-forward or reset Clipper `main` to this branch. Updating GitHub `origin/main` requires a force-push because histories diverge; the backup tag is the rollback.

After this, merge-base with a later Wavelet commit is the Wavelet commit that `sync/wavelet-upstream` started from (or a later merge commit).

### 3.2 Conflict policy

| Situation | Resolution |
|---|---|
| Same path exists in both; file is framework | Wavelet version unchanged |
| Clipper-only path (item, Clipper docs, branding) | Keep and adapt |
| Framework file Clipper edited (e.g. `initial_schema.sql`, `custom.go`) | Wavelet version; re-apply business via item files / `v1.go` hook |
| Import or API break in item | Change item, not the framework shape |
| Future `git merge upstream/main` | Same table |

## 4. Backend

### 4.1 Framework mapping (Wavelet is source of truth)

| Clipper today | Wavelet now |
|---|---|
| `internal/config` | `internal/infra/config` |
| `internal/db` | `internal/infra/persistence` (import path; package name stays `db`) |
| `internal/db/batchwriter` | `internal/infra/persistence/batchwriter` |
| `internal/db/idgen` | `internal/infra/persistence/idgen` |
| `internal/db/migrator` | `internal/infra/persistence/migrator` |
| `internal/diskcache` | `internal/infra/diskcache` |
| `internal/storage` | `internal/infra/objectstore` |
| `internal/task` | `internal/infra/task` |
| `internal/bootstrap` | `internal/platform/bootstrap` |
| `internal/lifecycle` | `internal/platform/lifecycle` |
| `internal/common` | `internal/shared` |
| `internal/common/response` | `internal/shared/response` |
| `.agent/` | `.agents/` |

Take Wavelet's skills, AGENTS.md skeleton, CI, Docker, Makefile, `go.mod`/`go.sum`. Then write a Clipper header on `AGENTS.md` (product name Clipper, tables `c_*` vs `w_*`, repo `https://github.com/Rain-kl/Clipper`) without reverting Wavelet path names.

### 4.2 Business files to port

Copy from pre-sync Clipper (backup tag or current `main`) and rewrite imports:

**Go**

- `internal/apps/item/*.go` — change `internal/db` → `internal/infra/persistence`, `internal/db/idgen` → `internal/infra/persistence/idgen`, `internal/common/response` → `internal/shared/response`, `internal/task` → `internal/infra/task`. `upload`, `model`, `repository` stay.
- `internal/model/item.go`, `internal/model/item_attachment.go`
- Config key constants `ConfigKeyItemPendingArchiveAfterDays` and `ConfigKeyItemTrashPurgeAfterDays` on `internal/model/system_configs.go` (Wavelet file + two Clipper constants)
- Clipper-only docs from the backup tag: `docs/superpowers/**` (MVP spec/plan and this sync spec). Wavelet `main` does not contain them.

**SQL** (do not edit Wavelet framework migrations)

- `internal/infra/persistence/migrator/goose/postgres/202607220001_create_c_items.sql`
- `internal/infra/persistence/migrator/goose/sqlite/202607220001_create_c_items.sql`

Keep existing contents: `c_items`, `c_item_attachments`, the two `w_system_configs` keys, and `w_schedules` ids **2** and **3** (`item_archive_pending`, `item_purge_trash`). Wavelet only seeds schedule id 1; 2 and 3 stay free.

**Registration (Wavelet `new-api` shape, not `custom.go`)**

- Add `RegisterItemRoutes(apiV1Router)` in `internal/router/v1/v1.go` at the commented "Product domain routes" slot. Implementation can live as `internal/router/v1/item.go` calling `item.RegisterRoutes`.
- Leave `custom.go` as Wavelet's scaffold demo.
- Register `item.ArchivePending*` and `item.PurgeTrash*` in `internal/infra/task/handlers/register.go` next to the other domain handlers.

**Branding**

- Keep Clipper ASCII banner and `Clipper %s` version line in `internal/cmd` (Wavelet banner file + Clipper art/name; imports use `infra/config` and `infra/persistence/migrator`).
- Keep Clipper swagger title/description in `main.go`.

Item HTTP contract stays `/api/v1/items` (create/list/timeline/stats/get/patch/delete), auth via `oauth.LoginRequired()`.

## 5. Frontend

### 5.1 Framework

Take Wavelet frontend as-is: `next-intl` non-routing provider, `frontend/i18n/*`, `frontend/messages/{zh-CN,en}.json`, language switcher, `check-i18n-keys.mjs`, already-translated layout/auth/settings/admin.

### 5.2 Business files to port

- Pages: `frontend/app/(main)/clip/**`, `review/**`, `search/**`, `library/**`, `vault/**`
- Shared UI: `frontend/components/common/item/**`
- Client: `frontend/lib/services/item/**`

Replace every user-visible literal in those files with `useTranslations` / `getTranslations`. Delete hardcoded maps in `labels.ts`; keep option **value** arrays (`pending`, `vault`, …) and resolve labels through `item.lifecycle.*` / `item.importance.*` / `item.contentType.*`.

### 5.3 Sidebar

Start from Wavelet `sidebar.tsx` (`titleKey` + `useTranslations('layout')`). Insert Clipper entries **before** `home` / `myFiles`:

| `titleKey` | url |
|---|---|
| `clip` | `/clip` |
| `review` | `/review` |
| `search` | `/search` |
| `library` | `/library` |
| `vault` | `/vault` |

Add matching keys under `layout.nav` in both message files.

### 5.4 Message catalog

Add an `item` namespace. `zh-CN.json` and `en.json` must have the same key tree. Required groups:

- `layout.nav.clip|review|search|library|vault`
- `item.lifecycle.pending|active|archived|trash`
- `item.importance.none|fragment|note|vault`
- `item.contentType.text|image|file`
- `item.clip.*` — page title, composer placeholders, submit, validation toasts
- `item.review.*` — title, day grouping, classify actions, empty state
- `item.search.*` — title, filters, empty state
- `item.library.*` — title, filters, empty state
- `item.vault.*` — title, empty state

Copy every string that exists in the current Clipper TSX/labels into `zh-CN`; write real English in `en` (not Chinese placeholders).

Do not localize backend `error_msg`, logs, or debug text. Dates/numbers use Wavelet `frontend/i18n/format.ts` helpers; do not hardcode `'zh-CN'` or `date-fns` `zhCN` on new/changed paths.

### 5.5 Product name

User-visible chrome says Clipper: startup banner, swagger `@title`, README, sidebar/home product title, `html`/docs titles that name the running product. Framework docs copy that says "Wavelet Platform" as a generic scaffold description may stay unless it is the in-app product title.

## 6. Error handling

- Item handlers keep `response.Abort*` / `ErrorHandlerMiddleware`; no `c.JSON` error envelopes.
- Logics stay `context.Context` + `(result, error)` with no gin dependency.
- Missing Wavelet symbols after the move are fixed in item or in the thin registration files (`v1/item.go`, `handlers/register.go`), not by restoring `internal/db` or `internal/common`.
- If a Clipper-edited framework test fails after taking Wavelet, prefer Wavelet's test; add item coverage next to `internal/apps/item`.

## 7. Testing and done criteria

| Gate | Command / check | Pass |
|---|---|---|
| Go tests | `go test ./...` from Clipper repo root | exit 0, including `internal/apps/item` |
| Frontend typecheck | `cd frontend && pnpm tsc --noEmit --jsx preserve` (also part of `make code-check`) | exit 0 |
| i18n keys | `node frontend/scripts/check-i18n-keys.mjs` | zh-CN / en trees match |
| Swagger | `make swagger` if handlers changed | `docs/` regenerates |
| Lint | `make code-check` and `make format` per AGENTS.md | exit 0 |
| Routes | item routes registered from `v1.go`, not `custom.go` | `/api/v1/items` present |
| Migrations | both dialects have `202607220001_create_c_items.sql` under `internal/infra/persistence/migrator/goose/` | files exist |
| Manual | login → Clip / Review / Search / Library / Vault; switch `zh-CN` / `en`; banner says Clipper | no hardcoded Chinese left on those five pages |

Do not include unrelated Clipper working-tree dirt (current `release-guide` skill edit, deleted `.codex`) in sync commits.

## 8. Ongoing Wavelet updates

Always inside Clipper:

```bash
git fetch upstream
git merge upstream/main
```

Resolve with section 3.2. Never `git push upstream`. After each merge, re-run section 7 gates and re-attach item registration if Wavelet rewrote `v1.go` or `handlers/register.go`.

## 9. Rollout

1. Backup tag on current Clipper `main`.
2. Build `sync/wavelet-upstream` from `upstream/main`.
3. Port backend item + registration + branding; make `go test ./...` green.
4. Port frontend pages/services; add i18n keys; make frontend gates green.
5. Swagger + format + code-check.
6. Point Clipper `main` at the branch; force-push `origin/main` only after local gates pass.

## 10. Risks

- **Force-push `main`:** anyone with a clone of old Clipper `main` must reset to the backup tag or reclone. Document the tag name in the sync commit message.
- **Wavelet `main` moves during the work:** fetch again and merge `upstream/main` into the sync branch before declaring done.
- **Schedule id clash:** if a future Wavelet migration inserts `w_schedules` id 2 or 3, change Clipper's item schedule ids in a new Clipper migration — do not edit Wavelet's SQL.
- **Skill path:** agents and docs must point at `.agents/`, not `.agent/`.
