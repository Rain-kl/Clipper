# Wavelet Upstream Sync Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Clipper a fetch-only descendant of current Wavelet `main`, with the item-capture business and bilingual Clip/Review/Search/Library/Vault UI working on the new framework tree.

**Architecture:** In the Clipper repo only, add remote `upstream` → Wavelet, tag current `main`, and do all work on branch `sync/wavelet-upstream` created from `upstream/main` (isolated worktree). Copy Clipper-only files from the backup tag, rewrite imports to `infra` / `platform` / `shared`, register item via `v1.go` (not `custom.go`), and add a next-intl `item` namespace. Never write to the Wavelet repo.

**Tech Stack:** Git remotes/worktrees; Go 1.25 / Gin / GORM / goose / Asynq; Next.js App Router / next-intl / TypeScript / pnpm.

**Spec:** `docs/superpowers/specs/2026-08-16-wavelet-upstream-sync-design.md`

## Global Constraints

- Wavelet repo is read-only: no branch, commit, or `git push upstream`.
- Go module path stays `github.com/Rain-kl/Wavelet`.
- Framework files: Wavelet version unchanged. Item breaks → fix item.
- Business tables `c_*`; framework tables `w_*`.
- Item routes: `RegisterItemRoutes` from `internal/router/v1/v1.go`, not `custom.go`.
- API errors only via `response.Abort*` (`internal/shared/response`).
- Logics: `context.Context` only, no `*gin.Context`.
- No new hardcoded UI strings; `zh-CN.json` and `en.json` key trees must match.
- Do not localize backend `error_msg`, logs, or debug text.
- Do not edit Wavelet framework goose SQL; item SQL is a new file only.
- `w_schedules` ids 2 and 3 stay (`item_archive_pending`, `item_purge_trash`).
- Tests: `t.TempDir()` only; no hardcoded relative temp dirs.
- Commits: Conventional Commits `<type>(<scope>): <subject>`.
- Do not commit Clipper working-tree dirt (`.agent/skills/release-guide/SKILL.md`, `.codex`).
- After handler changes: `make swagger`. After code: `make format` and `make code-check`.
- Execution workspace: `superpowers:using-git-worktrees` — Task 1 creates `.worktrees/sync-wavelet-upstream` from `upstream/main`. All later tasks run **inside that worktree**.

## File map

| Path | Responsibility |
| --- | --- |
| Clipper remote `upstream` | Fetch-only Wavelet |
| tag `backup/main-pre-wavelet-sync` | Pre-sync Clipper `main` (source of item files + docs) |
| `.worktrees/sync-wavelet-upstream` | Isolated checkout of `sync/wavelet-upstream` |
| `docs/superpowers/**` | Clipper specs/plans copied from the backup tag |
| `internal/model/item.go` | Item enums + `c_items` model |
| `internal/model/item_attachment.go` | `c_item_attachments` model |
| `internal/model/system_configs.go` | Two item retention keys |
| `internal/infra/persistence/migrator/goose/{postgres,sqlite}/202607220001_create_c_items.sql` | Item tables + config + schedules |
| `internal/testhelper/test_helper.go` | AutoMigrate + seed item configs |
| `internal/apps/item/*` | Domain logics, handlers, tasks |
| `internal/router/v1/item.go` | `RegisterItemRoutes` |
| `internal/router/v1/v1.go` | Call `RegisterItemRoutes` |
| `internal/infra/task/handlers/register.go` | Register archive/purge handlers |
| `internal/cmd/banner.go` + `banner_test.go` | Clipper ASCII art + name |
| `main.go` | Clipper swagger title |
| `AGENTS.md` | Wavelet skeleton + Clipper product header |
| `README.md` / `README_zh.md` | Clipper product docs from backup tag |
| `frontend/lib/services/item/*` | Item REST client |
| `frontend/lib/services/index.ts` | Export `item` |
| `frontend/messages/{zh-CN,en}.json` | `layout.nav.*` + `item.*` |
| `frontend/components/common/item/*` | Shared list/row + option values |
| `frontend/app/(main)/{clip,review,search,library,vault}/**` | Business pages |
| `frontend/components/layout/sidebar.tsx` | Clipper nav entries |
| `.gitignore` | Restore `.worktrees/` / `.superpowers/` |

**Import rewrite (every Go file copied from the backup tag):**

| Old import | New import |
| --- | --- |
| `github.com/Rain-kl/Wavelet/internal/db` | `github.com/Rain-kl/Wavelet/internal/infra/persistence` |
| `github.com/Rain-kl/Wavelet/internal/db/idgen` | `github.com/Rain-kl/Wavelet/internal/infra/persistence/idgen` |
| `github.com/Rain-kl/Wavelet/internal/common/response` | `github.com/Rain-kl/Wavelet/internal/shared/response` |
| `github.com/Rain-kl/Wavelet/internal/task` | `github.com/Rain-kl/Wavelet/internal/infra/task` |
| `github.com/Rain-kl/Wavelet/internal/config` | `github.com/Rain-kl/Wavelet/internal/infra/config` |
| `github.com/Rain-kl/Wavelet/internal/db/migrator` | `github.com/Rain-kl/Wavelet/internal/infra/persistence/migrator` |

Package names stay `db`, `response`, `task`, `config`, `idgen`, `migrator`. Do not change `model`, `repository`, `upload`, `oauth`.

**Source of Clipper files:** `git show backup/main-pre-wavelet-sync:<path>` or `git checkout backup/main-pre-wavelet-sync -- <path>` (only for Clipper-only paths).

---

### Task 1: Upstream remote, backup tag, sync worktree

**Files:**
- Modify: Clipper git remotes (not a source file)
- Create: tag `backup/main-pre-wavelet-sync`
- Create: branch `sync/wavelet-upstream` at `upstream/main`
- Create: worktree `.worktrees/sync-wavelet-upstream`
- Create (on the new branch): `docs/superpowers/**` copied from the tag
- Modify: `.gitignore` (append Clipper local ignores)

**Interfaces:**
- Consumes: Clipper `main` including this plan; Wavelet `origin` at `https://github.com/Rain-kl/Wavelet.git` (or local `/Users/ryan/Code/Go/Wavelet` if GitHub is unreachable)
- Produces: fetch-only `upstream`; tag `backup/main-pre-wavelet-sync`; worktree on `sync/wavelet-upstream` that has Wavelet tree + Clipper docs

- [ ] **Step 1: Add fetch-only upstream (Clipper repo root, current `main`)**

```bash
cd /Users/ryan/Code/Go/Clipper
git remote remove upstream 2>/dev/null || true
git remote add upstream https://github.com/Rain-kl/Wavelet.git
git remote set-url --push upstream DISABLED
git fetch upstream
git rev-parse --verify upstream/main
```

Expected: `upstream/main` resolves to a commit (design pin was `284eec5`; a newer tip is OK). `git remote -v` shows fetch → Wavelet, push → `DISABLED`.

If GitHub fetch fails, use the local clone instead:

```bash
git remote set-url upstream /Users/ryan/Code/Go/Wavelet
git fetch upstream
```

Do **not** `cd` into `/Users/ryan/Code/Go/Wavelet` to create branches.

- [ ] **Step 2: Tag current Clipper main**

```bash
git tag -f backup/main-pre-wavelet-sync main
git log -1 --oneline backup/main-pre-wavelet-sync
```

Expected: tag points at current Clipper `main` (includes this plan).

- [ ] **Step 3: Isolated worktree from Wavelet main**

```bash
git worktree add .worktrees/sync-wavelet-upstream -b sync/wavelet-upstream upstream/main
cd .worktrees/sync-wavelet-upstream
git merge-base --is-ancestor upstream/main HEAD && echo ANCESTOR_OK
test ! -d internal/apps/item && echo NO_ITEM_YET
test -d internal/infra && echo INFRA_OK
```

Expected: `ANCESTOR_OK`, `NO_ITEM_YET`, `INFRA_OK`. All later tasks use this directory as cwd.

- [ ] **Step 4: Copy Clipper docs and restore local gitignores**

```bash
git checkout backup/main-pre-wavelet-sync -- \
  docs/superpowers/specs/2026-07-22-clipper-mvp-design.md \
  docs/superpowers/specs/2026-08-16-wavelet-upstream-sync-design.md \
  docs/superpowers/plans/2026-07-22-clipper-mvp.md \
  docs/superpowers/plans/2026-08-16-wavelet-upstream-sync.md
```

Append to `.gitignore` (Wavelet file has no these entries):

```
.worktrees/
.superpowers/
```

- [ ] **Step 5: Commit**

```bash
git add docs/superpowers .gitignore
git commit -m "docs(sync): bring Clipper specs onto Wavelet main"
```

---

### Task 2: Item models, goose migrations, config keys, testhelper

**Files:**
- Create: `internal/model/item.go`
- Create: `internal/model/item_attachment.go`
- Modify: `internal/model/system_configs.go` (append two keys after `ConfigKeyStorageConfig`)
- Create: `internal/infra/persistence/migrator/goose/postgres/202607220001_create_c_items.sql`
- Create: `internal/infra/persistence/migrator/goose/sqlite/202607220001_create_c_items.sql`
- Modify: `internal/testhelper/test_helper.go` (AutoMigrate + seed)

**Interfaces:**
- Consumes: Wavelet `model.SystemConfig` and testhelper seed pattern
- Produces: `model.Item`, `model.ItemAttachment`, `model.ItemContentType` (`text|image|file`), `model.ItemLifecycle` (`pending|active|archived|trash`), `model.ItemImportance` (`none|fragment|note|vault`), `model.ItemSourceWeb = "web"`, `model.ConfigKeyItemPendingArchiveAfterDays = "item_pending_archive_after_days"`, `model.ConfigKeyItemTrashPurgeAfterDays = "item_trash_purge_after_days"`

- [ ] **Step 1: Copy models and SQL from the backup tag**

```bash
git checkout backup/main-pre-wavelet-sync -- \
  internal/model/item.go \
  internal/model/item_attachment.go
mkdir -p internal/infra/persistence/migrator/goose/postgres \
         internal/infra/persistence/migrator/goose/sqlite
git show backup/main-pre-wavelet-sync:internal/db/migrator/goose/postgres/202607220001_create_c_items.sql \
  > internal/infra/persistence/migrator/goose/postgres/202607220001_create_c_items.sql
git show backup/main-pre-wavelet-sync:internal/db/migrator/goose/sqlite/202607220001_create_c_items.sql \
  > internal/infra/persistence/migrator/goose/sqlite/202607220001_create_c_items.sql
```

Do not edit the SQL. Confirm both files still seed `w_schedules` ids 2 and 3.

- [ ] **Step 2: Add config key constants**

In `internal/model/system_configs.go`, inside the existing `const (` block, after `ConfigKeyStorageConfig`:

```go
	ConfigKeyItemPendingArchiveAfterDays = "item_pending_archive_after_days" // pending 超时归档天数
	ConfigKeyItemTrashPurgeAfterDays     = "item_trash_purge_after_days"     // trash 硬删天数
```

- [ ] **Step 3: Extend testhelper AutoMigrate + seed**

In `internal/testhelper/test_helper.go` `AutoMigrate` list, after `&model.Schedule{}`:

```go
		&model.Item{},
		&model.ItemAttachment{},
```

In the seed-config slice that already contains `ConfigKeyStorageConfig`, append:

```go
		{
			Key:         model.ConfigKeyItemPendingArchiveAfterDays,
			Value:       "3",
			Type:        configTypeSystem,
			Visibility:  0,
			Description: "pending archive days",
		},
		{
			Key:         model.ConfigKeyItemTrashPurgeAfterDays,
			Value:       "30",
			Type:        configTypeSystem,
			Visibility:  0,
			Description: "trash purge days",
		},
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/model ./internal/testhelper ./internal/infra/persistence/migrator -count=1
```

Expected: PASS (framework tests still green; Item is only AutoMigrated).

- [ ] **Step 5: Commit**

```bash
git add internal/model/item.go internal/model/item_attachment.go \
  internal/model/system_configs.go \
  internal/infra/persistence/migrator/goose/postgres/202607220001_create_c_items.sql \
  internal/infra/persistence/migrator/goose/sqlite/202607220001_create_c_items.sql \
  internal/testhelper/test_helper.go
git commit -m "feat(item): port models and c_items migrations onto infra goose path"
```

---

### Task 3: Item domain package (tests first)

**Files:**
- Create: `internal/apps/item/logics_test.go`
- Create: `internal/apps/item/archive_task_test.go`
- Create: `internal/apps/item/errs.go`
- Create: `internal/apps/item/logics.go`
- Create: `internal/apps/item/handlers.go`
- Create: `internal/apps/item/routers.go`
- Create: `internal/apps/item/archive_task.go`
- Create: `internal/apps/item/purge_task.go`

**Interfaces:**
- Consumes: Task 2 models/keys; `db.DB(ctx)` from `internal/infra/persistence`; `idgen.NextUint64ID()`; `upload.Ingest`/`RemoveOwned`; `task.TaskMeta` / `task.TaskHandler` / `task.AppendLog` from `internal/infra/task`; `response.Abort*` from `internal/shared/response`
- Produces:
  - `CreateItem(ctx, userID uint64, in CreateItemInput) (*model.Item, error)`
  - `ListItems`, `GetItem`, `PatchItem`, `DeleteItem`, `GetTimeline`, `GetStats`
  - `ArchivePendingItems(ctx, olderThan time.Duration) (int64, error)`
  - `PurgeTrashItems(ctx, olderThan time.Duration) (int64, error)`
  - `RegisterRoutes(r *gin.RouterGroup)` → `/items` with `oauth.LoginRequired()`
  - `ArchivePendingTask = "item:archive_pending"`, `TaskTypeArchivePending = "item_archive_pending"`, `ArchivePendingMeta task.TaskMeta`, `ArchivePendingHandler`
  - `PurgeTrashTask = "item:purge_trash"`, `TaskTypePurgeTrash = "item_purge_trash"`, `PurgeTrashMeta`, `PurgeTrashHandler`

- [ ] **Step 1: Copy tests from the tag and rewrite imports**

```bash
mkdir -p internal/apps/item
git show backup/main-pre-wavelet-sync:internal/apps/item/logics_test.go > internal/apps/item/logics_test.go
git show backup/main-pre-wavelet-sync:internal/apps/item/archive_task_test.go > internal/apps/item/archive_task_test.go
```

In `logics_test.go` only, replace:

```go
	"github.com/Rain-kl/Wavelet/internal/db"
```

with:

```go
	"github.com/Rain-kl/Wavelet/internal/infra/persistence"
```

Leave the `db.DB(ctx)` selector as-is (package name is still `db`). `archive_task_test.go` has no `internal/db` / `internal/task` imports — copy unchanged.

- [ ] **Step 2: Run tests — must fail to compile**

```bash
go test ./internal/apps/item -count=1
```

Expected: FAIL compile, undefined `CreateItem` / `ArchivePendingHandler` (or similar). If it passes, the implementation was copied too early.

- [ ] **Step 3: Copy implementation and rewrite imports**

```bash
git show backup/main-pre-wavelet-sync:internal/apps/item/errs.go > internal/apps/item/errs.go
git show backup/main-pre-wavelet-sync:internal/apps/item/logics.go > internal/apps/item/logics.go
git show backup/main-pre-wavelet-sync:internal/apps/item/handlers.go > internal/apps/item/handlers.go
git show backup/main-pre-wavelet-sync:internal/apps/item/routers.go > internal/apps/item/routers.go
git show backup/main-pre-wavelet-sync:internal/apps/item/archive_task.go > internal/apps/item/archive_task.go
git show backup/main-pre-wavelet-sync:internal/apps/item/purge_task.go > internal/apps/item/purge_task.go
```

Apply this exact substitution on those six files (not the tests, already done):

```bash
perl -pi -e '
  s#github.com/Rain-kl/Wavelet/internal/db/idgen#github.com/Rain-kl/Wavelet/internal/infra/persistence/idgen#g;
  s#github.com/Rain-kl/Wavelet/internal/db"#github.com/Rain-kl/Wavelet/internal/infra/persistence"#g;
  s#github.com/Rain-kl/Wavelet/internal/common/response#github.com/Rain-kl/Wavelet/internal/shared/response#g;
  s#github.com/Rain-kl/Wavelet/internal/task#github.com/Rain-kl/Wavelet/internal/infra/task#g;
' internal/apps/item/*.go
```

Verify no old paths remain:

```bash
rg 'internal/(db|common/response|task)"' internal/apps/item || echo CLEAN
```

Expected: `CLEAN`.

- [ ] **Step 4: Run item tests**

```bash
go test ./internal/apps/item -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/apps/item
git commit -m "feat(item): port item domain onto infra persistence and task packages"
```

---

### Task 4: Register item routes and tasks

**Files:**
- Create: `internal/router/v1/item.go`
- Modify: `internal/router/v1/v1.go`
- Modify: `internal/infra/task/handlers/register.go`
- Do **not** modify `internal/router/v1/custom.go`

**Interfaces:**
- Consumes: `item.RegisterRoutes`, `item.ArchivePendingTask`, `item.ArchivePendingHandler`, `item.ArchivePendingMeta`, `item.PurgeTrashTask`, `item.PurgeTrashHandler`, `item.PurgeTrashMeta`
- Produces: `RegisterItemRoutes(apiV1Router *gin.RouterGroup)` called from `RegisterV1Routes`; item tasks registered in `handlers.Register()`

- [ ] **Step 1: Add `internal/router/v1/item.go`**

```go
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"github.com/Rain-kl/Wavelet/internal/apps/item"
	"github.com/gin-gonic/gin"
)

// RegisterItemRoutes registers Clipper item capture APIs under /api/v1/items.
func RegisterItemRoutes(apiV1Router *gin.RouterGroup) {
	item.RegisterRoutes(apiV1Router)
}
```

- [ ] **Step 2: Call it from `v1.go`**

Replace the product-domain comment block in `RegisterV1Routes` so the function is:

```go
func RegisterV1Routes(apiV1Router *gin.RouterGroup, apiGroup *gin.RouterGroup) {
	RegisterUserRoutes(apiV1Router, apiGroup)
	RegisterAdminRoutes(apiV1Router)
	RegisterItemRoutes(apiV1Router)
	RegisterCustomRoutes(apiV1Router)
}
```

Leave `custom.go` as Wavelet's `/custom/hello` demo.

- [ ] **Step 3: Register task handlers**

In `internal/infra/task/handlers/register.go`:

Add import `"github.com/Rain-kl/Wavelet/internal/apps/item"`.

At the end of `Register()`, after the push block:

```go
	// item
	task.RegisterHandler(item.ArchivePendingTask, &item.ArchivePendingHandler{})
	task.RegisterTaskMeta(item.ArchivePendingMeta)
	task.RegisterHandler(item.PurgeTrashTask, &item.PurgeTrashHandler{})
	task.RegisterTaskMeta(item.PurgeTrashMeta)
```

Change the existing `internal/infra/task` import is already correct on this Wavelet file — do not revert it to `internal/task`.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/apps/item ./internal/router/... ./internal/infra/task/... -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/router/v1/item.go internal/router/v1/v1.go \
  internal/infra/task/handlers/register.go
git commit -m "feat(item): register item routes and retention tasks on Wavelet hooks"
```

---

### Task 5: Clipper branding (banner, swagger, AGENTS, README)

**Files:**
- Modify: `internal/cmd/banner.go`
- Modify: `internal/cmd/banner_test.go`
- Modify: `main.go`
- Modify: `AGENTS.md`
- Replace: `README.md`, `README_zh.md` from backup tag
- Modify: `frontend/messages/zh-CN.json` `metadata.title` / `metadata.description`
- Modify: `frontend/messages/en.json` same keys

**Interfaces:**
- Consumes: Wavelet `formatStartupBanner` shape (already uses `infra/config` + `infra/persistence/migrator`)
- Produces: banner text contains `Clipper`; swagger `@title Clipper API`; AGENTS product header; README is Clipper

- [ ] **Step 1: Write a failing banner assertion**

In `internal/cmd/banner_test.go`, the Wavelet test expects `"Wavelet v3.2.1"`. Change that string to `"Clipper v3.2.1"` first, then run:

```bash
go test ./internal/cmd -count=1 -run Banner
```

Expected: FAIL, got `Wavelet v3.2.1`.

- [ ] **Step 2: Clipper art + name in `banner.go`**

Replace only the ASCII lines and the version sprintf. Keep Wavelet imports.

```go
		"   ____ _ _                       ",
		"  / ___| (_)_ __  _ __   ___ _ __ ",
		" | |   | | | '_ \\| '_ \\ / _ \\ '__|",
		" | |___| | | |_) | |_) |  __/ |   ",
		"  \\____|_|_| .__/| .__/ \\___|_|   ",
		"           |_|   |_|              ",
		fmt.Sprintf(" Clipper %s", buildinfo.Version),
```

- [ ] **Step 3: Re-run banner test**

```bash
go test ./internal/cmd -count=1 -run Banner
```

Expected: PASS.

- [ ] **Step 4: `main.go` swagger header**

```go
// Package main 是 Clipper 程序入口
package main

import "github.com/Rain-kl/Wavelet/internal/cmd"

// @title Clipper API
// @version 1.0
// @description Clipper 后端 API：捕获条目、用户认证、系统配置、任务调度与文件能力。
// @contact.name Clipper
// @contact.url https://github.com/Rain-kl/Clipper
```

Keep the rest of Wavelet's swagger annotations (`@license`, `@BasePath`, security) as they are on `upstream/main`.

- [ ] **Step 5: AGENTS.md Clipper header**

Keep the entire Wavelet `AGENTS.md` (paths already say `infra` / `platform` / `shared` / `.agents`). Prepend this block at the very top, then a blank line, then Wavelet's `# AGENTS.md`:

```markdown
# AGENTS.md — Clipper AI 助手工作操作手册

本文件面向 AI 开发助手，定义其职责与操作规范。

**产品：** Clipper（跨设备捕获与分级留存）。业务表前缀 `c_*`，框架表前缀 `w_*`。
**Go module：** `github.com/Rain-kl/Wavelet`
**仓库：** https://github.com/Rain-kl/Clipper
```

Do not change Wavelet skill paths back to `internal/db` or `.agent`.

- [ ] **Step 6: README + in-app product title**

```bash
git checkout backup/main-pre-wavelet-sync -- README.md README_zh.md
```

In both message files, set:

`metadata.title` → `Clipper`  
`metadata.description` → zh-CN: `跨设备捕获与分级留存` / en: `Capture and graded retention across devices`

- [ ] **Step 7: Commit**

```bash
git add internal/cmd/banner.go internal/cmd/banner_test.go main.go \
  AGENTS.md README.md README_zh.md \
  frontend/messages/zh-CN.json frontend/messages/en.json
git commit -m "chore(brand): present Wavelet tree as Clipper"
```

---

### Task 6: Frontend ItemService

**Files:**
- Create: `frontend/lib/services/item/index.ts`
- Create: `frontend/lib/services/item/item.service.ts`
- Create: `frontend/lib/services/item/types.ts`
- Modify: `frontend/lib/services/index.ts`

**Interfaces:**
- Consumes: Wavelet `BaseService` / default export pattern in `frontend/lib/services/index.ts`
- Produces: `services.item` with `list`, `timeline`, `update`, `remove`, `create` (whatever the backup client already exports)

- [ ] **Step 1: Copy the client from the backup tag**

```bash
git checkout backup/main-pre-wavelet-sync -- \
  frontend/lib/services/item/index.ts \
  frontend/lib/services/item/item.service.ts \
  frontend/lib/services/item/types.ts
```

- [ ] **Step 2: Register on the Wavelet services barrel**

In `frontend/lib/services/index.ts`:

Add `import { ItemService } from './item';`

Add `item: ItemService,` to the `services` object (next to `push`).

If the backup `index.ts` also re-exported item types, add:

```ts
export type {
  ItemContentType,
  ItemLifecycle,
  ItemImportance,
  ItemAttachment,
  Item,
  ListItemsParams,
  ListItemsResult,
  TimelineDay,
  TimelineResult,
  CreateItemPayload,
  PatchItemPayload,
} from './item';
```

only if those types are exported from `frontend/lib/services/item/index.ts` (they are).

- [ ] **Step 3: Typecheck**

```bash
cd frontend && pnpm tsc --noEmit --jsx preserve
```

Expected: PASS (pages that import `services.item` do not exist yet, so only the barrel must compile).

- [ ] **Step 4: Commit**

```bash
git add frontend/lib/services/item frontend/lib/services/index.ts
git commit -m "feat(item): port ItemService client onto Wavelet frontend"
```

---

### Task 7: i18n catalogs (failing key check first)

**Files:**
- Modify: `frontend/messages/zh-CN.json`
- Modify: `frontend/messages/en.json`

**Interfaces:**
- Consumes: Wavelet nested catalog + `frontend/scripts/check-i18n-keys.mjs`
- Produces: identical key trees including `layout.nav.clip|review|search|library|vault` and the `item` namespace below

- [ ] **Step 1: Add Chinese keys only, prove the checker fails**

In `frontend/messages/zh-CN.json` `layout.nav`, **before** `"home"`:

```json
      "clip": "Clip",
      "review": "回顾",
      "search": "搜索",
      "library": "记录",
      "vault": "密码本",
```

Add a sibling namespace `item` (top-level, next to `layout`) with this exact tree:

```json
  "item": {
    "lifecycle": {
      "pending": "待分类",
      "active": "活跃",
      "archived": "归档",
      "trash": "回收站"
    },
    "importance": {
      "none": "未标记",
      "fragment": "片段",
      "note": "笔记",
      "vault": "重要"
    },
    "contentType": {
      "text": "文本",
      "image": "图片",
      "file": "文件"
    },
    "common": {
      "refresh": "刷新",
      "all": "全部",
      "loading": "加载中…",
      "total": "共 {count} 条",
      "actionFailed": "操作失败",
      "untitledFile": "未命名文件",
      "removeAttachment": "移除附件",
      "moreActions": "更多操作",
      "imageAlt": "附件图片",
      "fileFallback": "文件 {id}",
      "noBody": "(无正文)",
      "noContent": "（无内容）",
      "image": "图片",
      "classifiedAs": "已归类为{label}",
      "markedAs": "已标记为{label}",
      "trashed": "已移入垃圾箱",
      "trashedBin": "已移入回收站",
      "archived": "已归档",
      "restored": "已恢复",
      "deletedForever": "已永久删除",
      "pendingBadge": "待处理",
      "archivedBadge": "已归档",
      "trashAction": "垃圾",
      "reclassify": "重新分类",
      "moveToTrash": "移入垃圾箱",
      "moveToTrashBin": "移入回收站",
      "restore": "恢复",
      "deleteForever": "永久删除",
      "emptyDefaultTitle": "暂无内容",
      "emptyDefaultDescription": "试试调整筛选条件，或去 Clip 捕获一条"
    },
    "clip": {
      "title": "Clip",
      "placeholder": "粘贴或输入任何内容…（支持粘贴图片，⌘/Ctrl+Enter 发送）",
      "submit": "发送",
      "submitting": "发送中…",
      "needContent": "请输入内容或添加附件",
      "captured": "已捕获",
      "submitFailed": "提交失败"
    },
    "review": {
      "title": "回顾",
      "loadFailed": "加载时间线失败",
      "empty": "暂无条目。去 Clip 捕获第一条内容吧。",
      "today": "今天",
      "yesterday": "昨天",
      "loadArchivedFailed": "加载归档失败",
      "collapseArchived": "收起 {count} 条归档",
      "expandArchived": "已折叠 {count} 条归档"
    },
    "search": {
      "title": "搜索",
      "loadFailed": "搜索失败",
      "placeholder": "关键词（匹配正文）",
      "submit": "搜索",
      "lifecycle": "生命周期",
      "importance": "重要性",
      "type": "类型",
      "includeArchived": "包含归档",
      "includeTrash": "包含回收站",
      "emptyTitle": "没有匹配结果",
      "emptyDescription": "试试调整关键词或筛选条件"
    },
    "library": {
      "title": "记录",
      "loadFailed": "加载记录失败",
      "contentType": "基础类型",
      "lifecycle": "生命周期",
      "importance": "重要性",
      "emptyTitle": "暂无记录",
      "emptyDescription": "选择筛选条件，或去 Clip 捕获内容"
    },
    "vault": {
      "title": "密码本",
      "loadFailed": "加载密码本失败",
      "warning": "密码本内容未加密，请依赖账号安全。请勿在不安全环境中保存高敏感凭证。",
      "total": "共 {count} 条重要内容",
      "emptyTitle": "密码本为空",
      "emptyDescription": "在回顾中将条目标记为「重要」后会显示在这里"
    }
  }
```

Run:

```bash
node frontend/scripts/check-i18n-keys.mjs
```

Expected: FAIL, `en.json` missing `layout.nav.clip` and `item.*`.

- [ ] **Step 2: Add matching English keys**

`layout.nav` in `en.json`, before `"home"`:

```json
      "clip": "Clip",
      "review": "Review",
      "search": "Search",
      "library": "Library",
      "vault": "Vault",
```

`item` namespace in `en.json` (same keys, English values):

```json
  "item": {
    "lifecycle": {
      "pending": "Pending",
      "active": "Active",
      "archived": "Archived",
      "trash": "Trash"
    },
    "importance": {
      "none": "Unmarked",
      "fragment": "Snippet",
      "note": "Note",
      "vault": "Important"
    },
    "contentType": {
      "text": "Text",
      "image": "Image",
      "file": "File"
    },
    "common": {
      "refresh": "Refresh",
      "all": "All",
      "loading": "Loading…",
      "total": "{count} items",
      "actionFailed": "Action failed",
      "untitledFile": "Untitled file",
      "removeAttachment": "Remove attachment",
      "moreActions": "More actions",
      "imageAlt": "Attachment image",
      "fileFallback": "File {id}",
      "noBody": "(No body)",
      "noContent": "(No content)",
      "image": "Image",
      "classifiedAs": "Classified as {label}",
      "markedAs": "Marked as {label}",
      "trashed": "Moved to trash",
      "trashedBin": "Moved to trash",
      "archived": "Archived",
      "restored": "Restored",
      "deletedForever": "Permanently deleted",
      "pendingBadge": "Pending",
      "archivedBadge": "Archived",
      "trashAction": "Trash",
      "reclassify": "Reclassify",
      "moveToTrash": "Move to trash",
      "moveToTrashBin": "Move to trash",
      "restore": "Restore",
      "deleteForever": "Delete forever",
      "emptyDefaultTitle": "Nothing here",
      "emptyDefaultDescription": "Adjust filters, or capture something in Clip"
    },
    "clip": {
      "title": "Clip",
      "placeholder": "Paste or type anything… (images supported, ⌘/Ctrl+Enter to send)",
      "submit": "Send",
      "submitting": "Sending…",
      "needContent": "Enter text or add an attachment",
      "captured": "Captured",
      "submitFailed": "Submit failed"
    },
    "review": {
      "title": "Review",
      "loadFailed": "Failed to load timeline",
      "empty": "No items yet. Capture the first one in Clip.",
      "today": "Today",
      "yesterday": "Yesterday",
      "loadArchivedFailed": "Failed to load archive",
      "collapseArchived": "Hide {count} archived",
      "expandArchived": "{count} archived folded"
    },
    "search": {
      "title": "Search",
      "loadFailed": "Search failed",
      "placeholder": "Keyword (matches body)",
      "submit": "Search",
      "lifecycle": "Lifecycle",
      "importance": "Importance",
      "type": "Type",
      "includeArchived": "Include archived",
      "includeTrash": "Include trash",
      "emptyTitle": "No matches",
      "emptyDescription": "Try another keyword or filter"
    },
    "library": {
      "title": "Library",
      "loadFailed": "Failed to load library",
      "contentType": "Type",
      "lifecycle": "Lifecycle",
      "importance": "Importance",
      "emptyTitle": "No records",
      "emptyDescription": "Pick a filter, or capture something in Clip"
    },
    "vault": {
      "title": "Vault",
      "loadFailed": "Failed to load vault",
      "warning": "Vault items are not encrypted. Rely on account security. Do not store highly sensitive secrets on an untrusted device.",
      "total": "{count} important items",
      "emptyTitle": "Vault is empty",
      "emptyDescription": "Mark an item as Important in Review to see it here"
    }
  }
```

- [ ] **Step 3: Re-run key checker**

```bash
node frontend/scripts/check-i18n-keys.mjs
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add frontend/messages/zh-CN.json frontend/messages/en.json
git commit -m "feat(i18n): add Clipper item and nav message catalogs"
```

---

### Task 8: Shared item components (i18n)

**Files:**
- Create: `frontend/components/common/item/labels.ts`
- Create: `frontend/components/common/item/item-list.tsx`
- Create: `frontend/components/common/item/item-row.tsx`

**Interfaces:**
- Consumes: Task 6 types; Task 7 `item.lifecycle|importance|contentType|common` keys; `useTranslations` / `useLocale` from `next-intl`; `formatDateTime` from `@/i18n/format`
- Produces: `LIFECYCLE_OPTIONS`, `IMPORTANCE_OPTIONS`, `CONTENT_TYPE_OPTIONS` (values only); `ItemList`, `ItemRow`

- [ ] **Step 1: Copy files from the backup tag**

```bash
git checkout backup/main-pre-wavelet-sync -- \
  frontend/components/common/item/labels.ts \
  frontend/components/common/item/item-list.tsx \
  frontend/components/common/item/item-row.tsx
```

- [ ] **Step 2: Rewrite `labels.ts` to values-only**

Replace the file with:

```ts
import type {
  ItemContentType,
  ItemImportance,
  ItemLifecycle,
} from '@/lib/services/item/types';

export const LIFECYCLE_OPTIONS: ItemLifecycle[] = [
  'pending',
  'active',
  'archived',
  'trash',
];

export const IMPORTANCE_OPTIONS: ItemImportance[] = [
  'none',
  'fragment',
  'note',
  'vault',
];

export const CONTENT_TYPE_OPTIONS: ItemContentType[] = [
  'text',
  'image',
  'file',
];
```

- [ ] **Step 3: i18n `item-list.tsx`**

Add `import { useTranslations } from 'next-intl';`

Inside `ItemList`, resolve defaults via translations (do not keep Chinese default props):

```tsx
  const t = useTranslations('item.common');
  const resolvedTitle = emptyTitle ?? t('emptyDefaultTitle');
  const resolvedDescription = emptyDescription ?? t('emptyDefaultDescription');
```

Pass `resolvedTitle` / `resolvedDescription` to `EmptyStateWithBorder`. Callers may still override.

- [ ] **Step 4: i18n `item-row.tsx`**

- `import { useLocale, useTranslations } from 'next-intl';`
- `import { formatDateTime } from '@/i18n/format';`
- `import type { AppLocale } from '@/i18n/config';`
- Remove imports of `CONTENT_TYPE_LABELS`, `IMPORTANCE_LABELS`, `LIFECYCLE_LABELS`.
- In the component: `const t = useTranslations('item');` `const locale = useLocale() as AppLocale;`
- Labels: `t(\`contentType.${item.content_type}\`)`, `t(\`lifecycle.${item.lifecycle}\`)`, `t(\`importance.${item.importance}\`)`.
- Replace `formatRelativeTime` usage with `formatDateTime(item.created_at, locale)`.
- `previewText`: use `t('common.noContent')` instead of `'（无内容）'`.
- Toasts / buttons / aria-labels use `t('common.*')` keys from Task 7 (`actionFailed`, `markedAs` with `{ label: t(\`importance.${importance}\`) }`, `trashedBin`, `archived`, `restored`, `deletedForever`, `trashAction`, `moveToTrashBin`, `restore`, `deleteForever`, `moreActions`, `image`).
- Delete the `toLocaleString('zh-CN', …)` helper.

- [ ] **Step 5: Typecheck**

```bash
cd frontend && pnpm tsc --noEmit --jsx preserve
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/components/common/item
git commit -m "feat(item): port shared item UI and switch labels to next-intl"
```

---

### Task 9: Business pages (i18n)

**Files:**
- Create: `frontend/app/(main)/clip/page.tsx`
- Create: `frontend/app/(main)/clip/components/composer.tsx`
- Create: `frontend/app/(main)/review/page.tsx`
- Create: `frontend/app/(main)/review/components/day-section.tsx`
- Create: `frontend/app/(main)/review/components/item-block.tsx`
- Create: `frontend/app/(main)/search/page.tsx`
- Create: `frontend/app/(main)/library/page.tsx`
- Create: `frontend/app/(main)/vault/page.tsx`

**Interfaces:**
- Consumes: `services.item`, Task 7 keys, Task 8 `ItemList` / option arrays
- Produces: five routes that render with `useTranslations('item.clip'|…)` and no Chinese literals

- [ ] **Step 1: Copy pages from the backup tag**

```bash
git checkout backup/main-pre-wavelet-sync -- \
  frontend/app/\(main\)/clip \
  frontend/app/\(main\)/review \
  frontend/app/\(main\)/search \
  frontend/app/\(main\)/library \
  frontend/app/\(main\)/vault
```

- [ ] **Step 2: `clip/page.tsx` + `composer.tsx`**

`page.tsx`: `const t = useTranslations('item.clip');` heading `{t('title')}`.

`composer.tsx`: `useTranslations('item.clip')` and `useTranslations('item.common')`.

| Old literal | Key |
| --- | --- |
| `请输入内容或添加附件` | `item.clip.needContent` |
| `已捕获` | `item.clip.captured` |
| `提交失败` | `item.clip.submitFailed` |
| placeholder | `item.clip.placeholder` |
| `未命名文件` | `item.common.untitledFile` |
| `移除附件` | `item.common.removeAttachment` |
| `发送中…` / `发送` | `item.clip.submitting` / `item.clip.submit` |

- [ ] **Step 3: Review (`page.tsx`, `day-section.tsx`, `item-block.tsx`)**

`review/page.tsx`: `useTranslations('item.review')` + `item.common.refresh` / `item.review.loadFailed` / `item.review.empty` / `item.review.title`.

`day-section.tsx`: `item.review.today` / `yesterday`; archived toggle `t('collapseArchived', { count })` / `t('expandArchived', { count })`; `loadArchivedFailed`. Keep `toLocaleDateString(undefined, …)` (locale-aware). Do not hardcode `'zh-CN'`.

`item-block.tsx`: remove the local `importanceLabel` map; use `t('importance.' + key)`. All buttons/toasts/aria from `item.common` and `item.importance.*` (`pendingBadge`, `archivedBadge`, `classifiedAs`, `trashed`, `archived`, `actionFailed`, `trashAction`, `reclassify`, `moveToTrash`, `moreActions`, `noBody`, `imageAlt`, `fileFallback`).

- [ ] **Step 4: Search / Library / Vault**

Remove `*_LABELS` imports; map options with `t('item.lifecycle.' + v)` etc.

| Page | Keys |
| --- | --- |
| Search | `item.search.*`, `item.common.refresh|loading|total|all` |
| Library | `item.library.*`, `item.common.refresh|loading|total|all` |
| Vault | `item.vault.*`, `item.common.refresh|loading` (vault total uses `item.vault.total`) |

`ItemList` `emptyTitle` / `emptyDescription` take `t('emptyTitle')` / `t('emptyDescription')` from that page namespace.

- [ ] **Step 5: Hunt leftover Chinese / hardcoded locale**

```bash
rg -n '[\p{Han}]' \
  frontend/app/\(main\)/clip \
  frontend/app/\(main\)/review \
  frontend/app/\(main\)/search \
  frontend/app/\(main\)/library \
  frontend/app/\(main\)/vault \
  frontend/components/common/item
rg -n "zh-CN|zhCN" \
  frontend/app/\(main\)/clip \
  frontend/app/\(main\)/review \
  frontend/app/\(main\)/search \
  frontend/app/\(main\)/library \
  frontend/app/\(main\)/vault \
  frontend/components/common/item
```

Expected: no Han characters and no `zh-CN` / `zhCN` in those trees (comments in English only).

- [ ] **Step 6: Typecheck + i18n keys**

```bash
node frontend/scripts/check-i18n-keys.mjs
cd frontend && pnpm tsc --noEmit --jsx preserve
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend/app/\(main\)/clip frontend/app/\(main\)/review \
  frontend/app/\(main\)/search frontend/app/\(main\)/library \
  frontend/app/\(main\)/vault
git commit -m "feat(item): port Clipper pages and wire next-intl strings"
```

---

### Task 10: Sidebar nav entries

**Files:**
- Modify: `frontend/components/layout/sidebar.tsx`

**Interfaces:**
- Consumes: Task 7 `layout.nav.clip|review|search|library|vault`
- Produces: those five links before `home` / `myFiles`

- [ ] **Step 1: Extend Wavelet `navMainItems`**

Add lucide imports: `ClipboardPaste`, `History`, `Search`, `Library`, `Lock`.

Replace `navMainItems` with:

```ts
const navMainItems: NavItem[] = [
  { titleKey: 'clip', url: '/clip', icon: ClipboardPaste },
  { titleKey: 'review', url: '/review', icon: History },
  { titleKey: 'search', url: '/search', icon: Search },
  { titleKey: 'library', url: '/library', icon: Library },
  { titleKey: 'vault', url: '/vault', icon: Lock },
  { titleKey: 'home', url: '/home', icon: Home },
  { titleKey: 'myFiles', url: '/files', icon: FolderOpen },
];
```

Do not put Chinese in `titleKey`. Wavelet already does `t(\`nav.${item.titleKey}\`)`.

- [ ] **Step 2: Typecheck**

```bash
cd frontend && pnpm tsc --noEmit --jsx preserve
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/components/layout/sidebar.tsx
git commit -m "feat(item): add Clipper nav entries to Wavelet sidebar"
```

---

### Task 11: Repo-wide gates

**Files:**
- Modify: `docs/swagger.json`, `docs/swagger.yaml`, `docs/docs.go` (via `make swagger`)
- Possibly formatted sources from `make format`

**Interfaces:**
- Consumes: all previous tasks
- Produces: green gates listed in the spec

- [ ] **Step 1: Go tests**

```bash
go test ./... -count=1
```

Expected: PASS, including `internal/apps/item`.

- [ ] **Step 2: Swagger**

```bash
make swagger
```

Expected: `docs/` updates mention Clipper / `/api/v1/items`.

- [ ] **Step 3: Format + code-check + i18n**

```bash
make format
node frontend/scripts/check-i18n-keys.mjs
make code-check
```

Expected: all exit 0. `code-check` runs `golangci-lint` and `cd frontend && pnpm tsc --noEmit --jsx preserve && npx eslint . --max-warnings 0`.

- [ ] **Step 4: Registration sanity**

```bash
rg -n 'RegisterItemRoutes' internal/router/v1/v1.go
rg -n 'item.RegisterRoutes' internal/router/v1/custom.go && echo UNEXPECTED_CUSTOM || echo CUSTOM_CLEAN
rg -n 'ArchivePendingTask|PurgeTrashTask' internal/infra/task/handlers/register.go
test -f internal/infra/persistence/migrator/goose/postgres/202607220001_create_c_items.sql
test -f internal/infra/persistence/migrator/goose/sqlite/202607220001_create_c_items.sql
```

Expected: `RegisterItemRoutes` in `v1.go`; `CUSTOM_CLEAN`; both task names present; both SQL files exist.

- [ ] **Step 5: Commit generated / format-only diffs if any**

```bash
git add docs frontend internal
git status
git commit -m "chore(sync): swagger and format after Wavelet upstream port"
```

If `git status` is clean, skip the commit.

- [ ] **Step 6: Fetch Wavelet again before calling it done**

```bash
git fetch upstream
git merge --ff-only upstream/main || git merge upstream/main
```

If this creates conflicts, resolve with the spec table (Wavelet wins on framework files; keep item registration). Re-run Steps 1–3. Commit the merge if one was created:

```bash
git commit -m "merge(upstream): refresh Wavelet main before cutover"
```

(Only if merge made a commit.)

---

### Task 12: Point Clipper `main` at the sync branch

**Files:** none (git refs only)

**Interfaces:**
- Consumes: green Task 11 on `sync/wavelet-upstream`
- Produces: local `main` == `sync/wavelet-upstream`; `origin/main` updated only after an explicit force-push decision

- [ ] **Step 1: Move local `main`**

From the Clipper repo root (not only the worktree):

```bash
git checkout main
git reset --hard sync/wavelet-upstream
git merge-base --is-ancestor upstream/main main && echo UPSTREAM_ANCESTOR
git log -1 --oneline
```

Expected: `UPSTREAM_ANCESTOR`. Do **not** run `git push origin main` yet if the user has not confirmed the force-push. When they do:

```bash
git push --force-with-lease origin main
```

Mention `backup/main-pre-wavelet-sync` in the push / chat note so old clones can recover.

Ongoing sync (document in the cutover commit message if you create one; otherwise leave as operator notes):

```bash
git fetch upstream
git merge upstream/main
```

Never `git push upstream`.

---

## Spec coverage (self-review)

| Spec section | Task |
| --- | --- |
| 3.1 remotes / tag / branch from `upstream/main` | Task 1 |
| 3.1 no Wavelet writes | Task 1 (push URL `DISABLED`) |
| 3.2 conflict policy | Task 11 Step 6 |
| 4.1 package mapping | Tasks 3–4 import rewrite |
| 4.2 models / SQL / config keys / docs | Tasks 1–2 |
| 4.2 routes not in `custom.go` | Task 4 |
| 4.2 task registration | Task 4 |
| 4.2 banner + swagger | Task 5 |
| 5.1 Wavelet i18n kept | implicit (branch starts at Wavelet) |
| 5.2–5.4 pages + catalogs | Tasks 6–9 |
| 5.3 sidebar | Task 10 |
| 5.5 product name | Task 5 |
| 6 error handling | Task 3 (copied Abort* handlers) |
| 7 gates | Task 11 |
| 8 future merge | Task 12 notes |
| 9 rollout / force-push | Task 12 |
| 10 schedule id / skill path | Task 2 SQL untouched; Wavelet `.agents` comes with the branch |

No TBD / “handle later” steps. Symbol names (`RegisterItemRoutes`, `ArchivePendingTask`, config keys) are consistent across tasks.
