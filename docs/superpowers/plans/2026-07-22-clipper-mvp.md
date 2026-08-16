# Clipper MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship Clipper Web core loop — multi-user item capture (text/image/file), dual-axis lifecycle/importance, Review/Search/Library/Vault UI, and auto archive/purge jobs.

**Architecture:** First-class `item` domain (`internal/apps/item`) on Clipper platform. Business tables use `c_*` prefix. Files only via `upload.Ingest` / `upload.RemoveOwned`. Jobs register through `internal/task/handlers` + bootstrap (no `init()`). Frontend pages under `(main)` with `ItemService`.

**Tech Stack:** Go 1.25 / Gin / GORM / goose / Asynq / PostgreSQL+SQLite dual migrations; Next.js App Router / TypeScript / shadcn / existing BaseService upload APIs.

**Spec:** `docs/superpowers/specs/2026-07-22-clipper-mvp-design.md`

## Global Constraints

- Table prefix: business `c_*`, framework `w_*`
- All item queries filter `user_id = current user`; cross-user → 404
- API errors only via `response.Abort*`; success via `response.OK` / `OKNil`
- Routes only via `internal/router/v1` → package `RegisterRoutes`
- Tasks: `task.RegisterHandler` + `RegisterTaskMeta` in `internal/task/handlers/register.go`
- No hard-coded test dirs — use `t.TempDir()`
- After code: `make prettier`, `make code-check`; API changes: `make swagger`
- Commit style: `<type>(<scope>): <subject>`
- Module path: `github.com/Rain-kl/Clipper`
- IDs: `idgen.NextUint64ID()`; JSON ids as `json:"id,string"`
- Pending auto-archive → `lifecycle=archived` + `importance=fragment`
- `DELETE /items/:id` → trash; `?force=1` → hard delete
- Tags / AI / Provider / vault encryption: **out of scope**

## File map

| Path | Responsibility |
| --- | --- |
| `internal/model/item.go` | Item + enums + TableName |
| `internal/model/item_attachment.go` | Attachment row |
| `internal/model/system_configs.go` | Add config key constants |
| `internal/db/migrator/goose/postgres/202607220001_create_c_items.sql` | PG tables + indexes + config seed + schedules |
| `internal/db/migrator/goose/sqlite/202607220001_create_c_items.sql` | SQLite twin |
| `internal/apps/item/errs.go` | User-facing error strings |
| `internal/apps/item/logics.go` | Create/list/get/patch/delete/timeline/stats/archive/purge |
| `internal/apps/item/logics_test.go` | Domain tests |
| `internal/apps/item/handlers.go` | HTTP handlers + swagger |
| `internal/apps/item/routers.go` | `RegisterRoutes` |
| `internal/apps/item/task/archive_pending.go` | Job: archive pending |
| `internal/apps/item/task/purge_trash.go` | Job: purge trash |
| `internal/task/handlers/register.go` | Register item tasks |
| `internal/router/v1/custom.go` or new `item.go` | Wire item routes under v1 |
| `internal/testhelper/test_helper.go` | AutoMigrate Item models + seed item configs |
| `frontend/lib/services/item/*` | Client API |
| `frontend/lib/services/index.ts` | Export item service |
| `frontend/components/layout/sidebar.tsx` | Clipper nav entries |
| `frontend/app/(main)/clip/page.tsx` | Capture UI |
| `frontend/app/(main)/review/page.tsx` | Timeline UI |
| `frontend/app/(main)/search/page.tsx` | Search UI |
| `frontend/app/(main)/library/page.tsx` | Faceted browse |
| `frontend/app/(main)/vault/page.tsx` | Vault filter view |
| `frontend/app/(main)/review/components/*` | Day card / item block (if page grows) |

---

### Task 1: Models + goose migrations + config keys

**Files:**
- Create: `internal/model/item.go`
- Create: `internal/model/item_attachment.go`
- Modify: `internal/model/system_configs.go` (append keys)
- Create: `internal/db/migrator/goose/postgres/202607220001_create_c_items.sql`
- Create: `internal/db/migrator/goose/sqlite/202607220001_create_c_items.sql`
- Modify: `internal/testhelper/test_helper.go` (AutoMigrate + seed)

**Interfaces:**
- Produces: `model.Item`, `model.ItemAttachment`, lifecycle/importance/content_type constants, `ConfigKeyItemPendingArchiveAfterDays`, `ConfigKeyItemTrashPurgeAfterDays`

- [ ] **Step 1: Add model files**

`internal/model/item.go`:

```go
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

type ItemContentType string

const (
	ItemContentTypeText  ItemContentType = "text"
	ItemContentTypeImage ItemContentType = "image"
	ItemContentTypeFile  ItemContentType = "file"
)

type ItemLifecycle string

const (
	ItemLifecyclePending  ItemLifecycle = "pending"
	ItemLifecycleActive   ItemLifecycle = "active"
	ItemLifecycleArchived ItemLifecycle = "archived"
	ItemLifecycleTrash    ItemLifecycle = "trash"
)

type ItemImportance string

const (
	ItemImportanceNone     ItemImportance = "none"
	ItemImportanceFragment ItemImportance = "fragment"
	ItemImportanceNote     ItemImportance = "note"
	ItemImportanceVault    ItemImportance = "vault"
)

const ItemSourceWeb = "web"

// Item is one capture unit owned by a user.
type Item struct {
	ID          uint64         `json:"id,string" gorm:"primaryKey"`
	UserID      uint64         `json:"user_id,string" gorm:"index;not null"`
	ContentType ItemContentType `json:"content_type" gorm:"size:16;not null;index"`
	Title       string         `json:"title" gorm:"size:255"`
	Body        string         `json:"body" gorm:"type:text"`
	Lifecycle   ItemLifecycle  `json:"lifecycle" gorm:"size:16;not null;index"`
	Importance  ItemImportance `json:"importance" gorm:"size:16;not null;index"`
	Source      string         `json:"source" gorm:"size:32;not null;default:web"`
	ArchivedAt  *time.Time     `json:"archived_at"`
	TrashedAt   *time.Time     `json:"trashed_at"`
	CreatedAt   time.Time      `json:"created_at" gorm:"autoCreateTime;index"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
}

func (Item) TableName() string { return "c_items" }
```

`internal/model/item_attachment.go`:

```go
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

// ItemAttachment links an item to an upload record.
type ItemAttachment struct {
	ID        uint64    `json:"id,string" gorm:"primaryKey"`
	ItemID    uint64    `json:"item_id,string" gorm:"index;not null"`
	UploadID  uint64    `json:"upload_id,string" gorm:"index;not null"`
	Sort      int       `json:"sort" gorm:"not null;default:0"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (ItemAttachment) TableName() string { return "c_item_attachments" }
```

Append to `internal/model/system_configs.go` constants:

```go
ConfigKeyItemPendingArchiveAfterDays = "item_pending_archive_after_days" // pending 超时归档天数
ConfigKeyItemTrashPurgeAfterDays     = "item_trash_purge_after_days"     // trash 硬删天数
```

- [ ] **Step 2: Write dual goose migrations**

Postgres `internal/db/migrator/goose/postgres/202607220001_create_c_items.sql`:

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS c_items (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    content_type VARCHAR(16) NOT NULL,
    title VARCHAR(255) NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    lifecycle VARCHAR(16) NOT NULL,
    importance VARCHAR(16) NOT NULL,
    source VARCHAR(32) NOT NULL DEFAULT 'web',
    archived_at TIMESTAMPTZ NULL,
    trashed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_c_items_user_created ON c_items (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_c_items_user_lifecycle_created ON c_items (user_id, lifecycle, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_c_items_user_importance_lifecycle ON c_items (user_id, importance, lifecycle);
CREATE INDEX IF NOT EXISTS idx_c_items_pending_created ON c_items (lifecycle, created_at);
CREATE INDEX IF NOT EXISTS idx_c_items_trash_trashed ON c_items (lifecycle, trashed_at);

CREATE TABLE IF NOT EXISTS c_item_attachments (
    id BIGINT PRIMARY KEY,
    item_id BIGINT NOT NULL,
    upload_id BIGINT NOT NULL,
    sort INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_c_item_attachments_item ON c_item_attachments (item_id);
CREATE INDEX IF NOT EXISTS idx_c_item_attachments_upload ON c_item_attachments (upload_id);

INSERT INTO w_system_configs (key, value, type, visibility, description, created_at, updated_at)
VALUES
  ('item_pending_archive_after_days', '3', 'system', 0, '未处理 Item 自动归档天数', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  ('item_trash_purge_after_days', '30', 'system', 0, '垃圾箱 Item 彻底删除天数', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (key) DO NOTHING;

INSERT INTO w_schedules (id, name, task_type, cron, payload, is_active, created_at, updated_at)
VALUES
  (2, 'Item 未处理归档', 'item_archive_pending', '15 * * * *', '{}', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
  (3, 'Item 垃圾箱清理', 'item_purge_trash', '45 * * * *', '{}', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM w_schedules WHERE id IN (2, 3);
DELETE FROM w_system_configs WHERE key IN ('item_pending_archive_after_days', 'item_trash_purge_after_days');
DROP TABLE IF EXISTS c_item_attachments;
DROP TABLE IF EXISTS c_items;
```

SQLite twin: same SQL but use `DATETIME` instead of `TIMESTAMPTZ` for timestamp columns (match `202606170002` style). Keep `ON CONFLICT` if SQLite schedules/config use unique keys (config key is unique; schedules id PK).

- [ ] **Step 3: Extend testhelper AutoMigrate**

In `SetupTestEnvironment` AutoMigrate list add:

```go
&model.Item{},
&model.ItemAttachment{},
```

In seed configs add:

```go
{Key: model.ConfigKeyItemPendingArchiveAfterDays, Value: "3", Type: configTypeSystem, Visibility: 0, Description: "pending archive days"},
{Key: model.ConfigKeyItemTrashPurgeAfterDays, Value: "30", Type: configTypeSystem, Visibility: 0, Description: "trash purge days"},
```

- [ ] **Step 4: Verify models compile**

Run: `go test ./internal/model/ -count=1`  
Expected: PASS (or no tests, compile OK)

- [ ] **Step 5: Commit**

```bash
git add internal/model/item.go internal/model/item_attachment.go internal/model/system_configs.go \
  internal/db/migrator/goose/postgres/202607220001_create_c_items.sql \
  internal/db/migrator/goose/sqlite/202607220001_create_c_items.sql \
  internal/testhelper/test_helper.go
git commit -m "feat(item): add c_items models, migrations, and retention config keys"
```

---

### Task 2: Item logics — create, get, list, transitions (TDD)

**Files:**
- Create: `internal/apps/item/errs.go`
- Create: `internal/apps/item/logics.go`
- Create: `internal/apps/item/logics_test.go`

**Interfaces:**
- Consumes: `model.Item*`, `idgen.NextUint64ID`, `db.DB`, `repository.GetActiveUploadByID` (or equivalent), `upload.RemoveOwned`
- Produces:
  - `CreateItem(ctx, userID, CreateItemInput) (*ItemDTO, error)`
  - `GetItem(ctx, userID, id) (*ItemDTO, error)`
  - `ListItems(ctx, userID, ListItemsQuery) (*ListItemsResult, error)`
  - `PatchItem(ctx, userID, id, PatchItemInput) (*ItemDTO, error)`
  - `DeleteItem(ctx, userID, id, force bool) error`
  - `Timeline(ctx, userID, TimelineQuery) (*TimelineResult, error)`
  - `Stats(ctx, userID) (*ItemStats, error)`
  - `ArchivePendingItems(ctx, olderThan time.Duration) (int, error)`
  - `PurgeTrashItems(ctx, olderThan time.Duration) (int, error)`

- [ ] **Step 1: Write failing tests first** (`logics_test.go`)

Use `testhelper.SetupTestEnvironment(t)`. Cover at minimum:

1. `TestCreateItem_TextOnly` — body set, lifecycle pending, importance none, content_type text  
2. `TestCreateItem_EmptyRejected` — no body no uploads → error  
3. `TestCreateItem_CrossUserUploadRejected` — upload owned by other user → error  
4. `TestClassifyPendingToNote` — patch active+note  
5. `TestTrashAndRestore` — delete soft trash; restore with importance none → pending  
6. `TestListDefaultExcludesArchivedAndTrash`  
7. `TestGetItem_OtherUserNotFound`  
8. `TestArchivePendingJob` — create old pending, run ArchivePendingItems, expect archived+fragment  
9. `TestPurgeTrashJob` — trash with old trashed_at, purge hard-deletes row  

Skeleton for create test:

```go
func TestCreateItem_TextOnly(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()

	userID := uint64(1001)
	item, err := CreateItem(ctx, userID, CreateItemInput{Body: "hello"})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if item.Lifecycle != model.ItemLifecyclePending || item.Importance != model.ItemImportanceNone {
		t.Fatalf("state = %s/%s", item.Lifecycle, item.Importance)
	}
	if item.ContentType != model.ItemContentTypeText || item.Body != "hello" {
		t.Fatalf("content mismatch: %+v", item)
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL**

Run: `go test ./internal/apps/item/ -count=1 -v`  
Expected: fail (package or CreateItem undefined)

- [ ] **Step 3: Implement errs + logics**

`errs.go` string constants (Chinese user-facing, camelCase names):

```go
const (
	errBindParamsFailed     = "参数绑定失败"
	errItemNotFound         = "记录不存在"
	errEmptyContent         = "内容不能为空"
	errInvalidTransition    = "非法的状态变更"
	errInvalidUpload        = "附件无效或不属于当前用户"
	errInternal             = "内部错误"
)
```

Core create rules in `CreateItem`:

```go
type CreateItemInput struct {
	Title     string
	Body      string
	UploadIDs []uint64
}

// 1. Trim body; if body=="" && len(uploads)==0 → error empty
// 2. Load each upload via repository.GetActiveUploadByID; must UserID==userID
// 3. Detect content_type per spec
// 4. Insert Item with idgen.NextUint64ID(), pending/none/web
// 5. Insert attachments with sort order; mark uploads used if project has status update helper
//    (if upload status stays pending from HTTP upload, set Status=used on attach — update via gorm)
```

Patch transition matrix (enforce):

| From | Request | Result |
| --- | --- | --- |
| pending | active + fragment\|note\|vault | active + importance |
| pending/active | trash | trash + trashed_at=now; clear archived_at optional |
| active | archived | archived + archived_at; keep importance |
| trash | active (or restore) | if importance none → pending else active; clear trashed_at |
| archived | active | active; clear archived_at |
| any | title/body only | update fields without state change |

`DeleteItem(force=false)` → same as patch trash.  
`DeleteItem(force=true)` → load attachments, `upload.RemoveOwned` each, delete attachment rows, delete item.

`ListItemsQuery`:

```go
type ListItemsQuery struct {
	Page             int
	PageSize         int
	Q                string
	Lifecycle        string // optional single
	Importance       string
	ContentType      string
	IncludeArchived  bool
	IncludeTrash     bool
	// optional CreatedFrom/To time.Time
}
```

Default filter: `lifecycle IN ('pending','active')` unless include flags.

`ArchivePendingItems`:  
`WHERE lifecycle='pending' AND created_at < now-days`  
→ set archived, importance=fragment, archived_at=now.

`PurgeTrashItems`:  
`WHERE lifecycle='trash' AND trashed_at < now-days`  
→ hard delete each via shared internal `hardDeleteItem`.

Read days: `repository.GetIntByKey(ctx, model.ConfigKeyItemPendingArchiveAfterDays)` default 3 on error.

- [ ] **Step 4: Run tests — expect PASS**

Run: `go test ./internal/apps/item/ -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/apps/item/
git commit -m "feat(item): implement item domain logics with lifecycle rules"
```

---

### Task 3: HTTP handlers, routes, swagger

**Files:**
- Create: `internal/apps/item/handlers.go`
- Create: `internal/apps/item/routers.go`
- Modify: `internal/router/v1/custom.go` **or** create `internal/router/v1/item.go` and call from `v1.go`
- Prefer: `internal/router/v1/item.go` + `RegisterV1Routes` call `RegisterItemRoutes`

**Interfaces:**
- Consumes: logics above; `oauth.LoginRequired`, `oauth.GetUserIDFromContext`
- Produces: REST under `/api/v1/items`

- [ ] **Step 1: routers.go**

```go
func RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/items")
	g.Use(oauth.LoginRequired())
	{
		g.POST("", CreateItem)
		g.GET("", ListItems)
		g.GET("/timeline", GetTimeline)
		g.GET("/stats", GetStats)
		g.GET("/:id", GetItem)
		g.PATCH("/:id", PatchItem)
		g.DELETE("/:id", DeleteItem)
	}
}
```

Note: register static paths (`timeline`, `stats`) **before** `/:id`.

- [ ] **Step 2: handlers**

Patterns:

```go
func CreateItem(c *gin.Context) {
	var req createItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortBadRequest(c, errBindParamsFailed)
		return
	}
	userID := oauth.GetUserIDFromContext(c)
	// parse upload_ids strings to uint64 if JSON uses string ids
	dto, err := CreateItem(c.Request.Context(), userID, ...)
	if err != nil {
		// map domain errors: empty/invalid upload/transition → AbortBadRequest
		// not found → AbortNotFound
		// else log + AbortInternal
		return
	}
	c.JSON(http.StatusOK, response.OK(dto))
}
```

Request structs (JSON camelCase matching platform style used elsewhere — check existing handlers: many use snake_case in JSON tags like `page_size`. **Match upload handlers: snake_case json tags.**):

```go
type createItemRequest struct {
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	UploadIDs []string `json:"upload_ids"`
}
```

List query:

```go
type listItemsRequest struct {
	Page            int    `form:"page"`
	PageSize        int    `form:"page_size"`
	Q               string `form:"q"`
	Lifecycle       string `form:"lifecycle"`
	Importance      string `form:"importance"`
	ContentType     string `form:"content_type"`
	IncludeArchived bool   `form:"include_archived"`
	IncludeTrash    bool   `form:"include_trash"`
}
```

DELETE: `force := c.Query("force") == "1"`

Every handler needs full swagger comments (`@Tags item`, `@Security` if project uses it — mirror upload).

- [ ] **Step 3: Wire routes**

`internal/router/v1/item.go`:

```go
func RegisterItemRoutes(apiV1Router *gin.RouterGroup) {
	item.RegisterRoutes(apiV1Router)
}
```

In `RegisterV1Routes` add `RegisterItemRoutes(apiV1Router)`.

- [ ] **Step 4: Swagger + compile**

Run: `make swagger && go test ./internal/apps/item/ ./internal/router/... -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/apps/item/handlers.go internal/apps/item/routers.go \
  internal/router/v1/item.go internal/router/v1/v1.go docs/
git commit -m "feat(item): expose authenticated item REST API"
```

---

### Task 4: Asynq tasks for archive + purge

**Files:**
- Create: `internal/apps/item/task/archive_pending.go`
- Create: `internal/apps/item/task/purge_trash.go`
- Create: `internal/apps/item/task/exports.go` (re-export constants for handlers register) **or** export from package task and import path `itemtask`
- Modify: `internal/task/handlers/register.go`
- Optional: `internal/apps/item/exports.go` if register imports root item package

**Interfaces:**
- Produces: `item:archive_pending` / meta type `item_archive_pending`; `item:purge_trash` / `item_purge_trash`
- Schedule seeds already in Task 1 migration (cron hourly)

- [ ] **Step 1: Implement handlers mirroring upload cleanup**

```go
const (
	ArchivePendingTask     = "item:archive_pending"
	TaskTypeArchivePending = "item_archive_pending"
)

var ArchivePendingMeta = task.TaskMeta{
	Type:         TaskTypeArchivePending,
	AsynqTask:    ArchivePendingTask,
	Name:         "Item 未处理归档",
	Description:  "将超时未处理的 Item 归档为片段",
	SupportsTime: false,
	MaxRetry:     task.DefaultMaxRetry,
	Queue:        task.QueueDefault,
	Retryable:    true,
}

type ArchivePendingHandler struct{}

func (h *ArchivePendingHandler) Execute(ctx context.Context, _ []byte) (*task.TaskResult, error) {
	days, err := repository.GetIntByKey(ctx, model.ConfigKeyItemPendingArchiveAfterDays)
	if err != nil || days <= 0 {
		days = 3
	}
	n, err := item.ArchivePendingItems(ctx, time.Duration(days)*24*time.Hour)
	// avoid import cycle: put ArchivePendingItems in logics; task package imports parent carefully
	// If cycle: move ArchivePendingItems to internal/apps/item/logics.go and call via same package
	// Prefer task files in package `item` subfolder `task` calling `itemlogics` —
	// Simplest: put task handlers in package `item` files archive_task.go to avoid cycle.
}
```

**Avoid import cycle:** implement task handlers as `package item` files `archive_task.go` / `purge_task.go` in `internal/apps/item/` (same package as logics), not a child package that imports parent. Export task constants from package `item`.

Revised files:
- `internal/apps/item/archive_task.go`
- `internal/apps/item/purge_task.go`

- [ ] **Step 2: Register in handlers/register.go**

```go
task.RegisterHandler(item.ArchivePendingTask, &item.ArchivePendingHandler{})
task.RegisterTaskMeta(item.ArchivePendingMeta)
task.RegisterHandler(item.PurgeTrashTask, &item.PurgeTrashHandler{})
task.RegisterTaskMeta(item.PurgeTrashMeta)
```

- [ ] **Step 3: Test job wrappers**

Add `TestArchivePendingHandler_Execute` that inserts old pending row and runs handler.

Run: `go test ./internal/apps/item/ -count=1`  
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/apps/item/archive_task.go internal/apps/item/purge_task.go internal/task/handlers/register.go
git commit -m "feat(item): register archive-pending and purge-trash tasks"
```

---

### Task 5: Frontend ItemService

**Files:**
- Create: `frontend/lib/services/item/types.ts`
- Create: `frontend/lib/services/item/item.service.ts`
- Create: `frontend/lib/services/item/index.ts`
- Modify: `frontend/lib/services/index.ts`

- [ ] **Step 1: types.ts**

```ts
export type ItemContentType = 'text' | 'image' | 'file';
export type ItemLifecycle = 'pending' | 'active' | 'archived' | 'trash';
export type ItemImportance = 'none' | 'fragment' | 'note' | 'vault';

export interface ItemAttachment {
  id: string;
  item_id: string;
  upload_id: string;
  sort: number;
  // optional denormalized from API:
  file_name?: string;
  mime_type?: string;
  file_size?: number;
}

export interface Item {
  id: string;
  user_id: string;
  content_type: ItemContentType;
  title: string;
  body: string;
  lifecycle: ItemLifecycle;
  importance: ItemImportance;
  source: string;
  archived_at?: string | null;
  trashed_at?: string | null;
  created_at: string;
  updated_at: string;
  attachments?: ItemAttachment[];
}

export interface ListItemsParams {
  page?: number;
  page_size?: number;
  q?: string;
  lifecycle?: ItemLifecycle;
  importance?: ItemImportance;
  content_type?: ItemContentType;
  include_archived?: boolean;
  include_trash?: boolean;
}

export interface ListItemsResult {
  total: number;
  results: Item[];
}

export interface TimelineDay {
  date: string; // YYYY-MM-DD server-suggested; client may regroup
  items: Item[];
  archived_count: number;
  archived_items?: Item[];
}

export interface TimelineResult {
  days: TimelineDay[];
}

export interface CreateItemPayload {
  title?: string;
  body?: string;
  upload_ids?: string[];
}

export interface PatchItemPayload {
  title?: string;
  body?: string;
  lifecycle?: ItemLifecycle;
  importance?: ItemImportance;
}
```

- [ ] **Step 2: item.service.ts**

```ts
import { BaseService } from '../core/base.service';
import type {
  CreateItemPayload,
  Item,
  ListItemsParams,
  ListItemsResult,
  PatchItemPayload,
  TimelineResult,
} from './types';

export class ItemService extends BaseService {
  protected static readonly basePath = '/api/v1/items';

  static create(payload: CreateItemPayload) {
    return this.post<Item>('', payload);
  }

  static list(params: ListItemsParams = {}) {
    return this.get<ListItemsResult>('', params as Record<string, unknown>);
  }

  static getById(id: string) {
    return this.get<Item>(`/${id}`);
  }

  static patch(id: string, payload: PatchItemPayload) {
    return this.patch<Item>(`/${id}`, payload);
  }

  static remove(id: string, force = false) {
    const q = force ? '?force=1' : '';
    return this.delete<void>(`/${id}${q}`);
  }

  static timeline(params?: { expand_archived?: boolean; before?: string; limit?: number }) {
    return this.get<TimelineResult>('/timeline', params as Record<string, unknown>);
  }

  static stats() {
    return this.get<Record<string, number>>('/stats');
  }
}
```

If `BaseService` has no `patch` method, use existing `put` or add `protected static patch` mirroring post (check base.service.ts — implement patch if missing).

- [ ] **Step 3: Register in index.ts**

```ts
import { ItemService } from './item';
// services object:
item: ItemService,
```

- [ ] **Step 4: Typecheck**

Run: `cd frontend && pnpm tsc --noEmit --jsx preserve`  
Expected: no errors related to item service

- [ ] **Step 5: Commit**

```bash
git add frontend/lib/services/item frontend/lib/services/index.ts frontend/lib/services/core/base.service.ts
git commit -m "feat(item): add frontend ItemService client"
```

---

### Task 6: Sidebar nav + Clip capture page

**Files:**
- Modify: `frontend/components/layout/sidebar.tsx` (`data.navMain`)
- Create: `frontend/app/(main)/clip/page.tsx`
- Optional components under `frontend/app/(main)/clip/components/composer.tsx` if page exceeds ~200 lines

- [ ] **Step 1: Nav entries**

Replace/extend `navMain`:

```ts
import { ClipboardPaste, History, Search, Library, Lock } from 'lucide-react';

navMain: [
  { title: 'Clip', url: '/clip', icon: ClipboardPaste },
  { title: '回顾', url: '/review', icon: History },
  { title: '搜索', url: '/search', icon: Search },
  { title: '记录', url: '/library', icon: Library },
  { title: '密码本', url: '/vault', icon: Lock },
  { title: '我的文件', url: '/files', icon: FolderOpen },
],
```

Keep or demote 首页 `/home` as product prefers — default: keep Home under document or leave first; **product-first: Clip first**.

- [ ] **Step 2: Clip page**

Structure:

```tsx
'use client';
import { RequireAuth } from '@/components/auth/require-auth';
import { ClipboardPaste } from 'lucide-react';
import services from '@/lib/services';
import { toast } from 'sonner';
// Textarea + Button from shadcn
// File input + paste handler → services.upload.uploadFile(file, 'clip')
// Then services.item.create({ body, upload_ids })
```

Behavior:
- Multi-line textarea
- Attach button / paste images into pending file list
- Submit: upload each pending File → collect ids → create item → clear → toast success
- On empty submit: toast error
- Title bar: `py-6`, icon `size-5 text-primary`, `h1` `text-2xl font-semibold tracking-tight`

- [ ] **Step 3: Manual smoke (if server running)**

Login → `/clip` → send text → network 200

- [ ] **Step 4: Commit**

```bash
git add frontend/components/layout/sidebar.tsx frontend/app/(main)/clip
git commit -m "feat(item): add Clip capture page and sidebar navigation"
```

---

### Task 7: Review timeline + classify actions

**Files:**
- Create: `frontend/app/(main)/review/page.tsx`
- Create: `frontend/app/(main)/review/components/day-section.tsx`
- Create: `frontend/app/(main)/review/components/item-block.tsx`

- [ ] **Step 1: Backend timeline DTO** (if not finished in Task 2)

`Timeline` logics:
- Fetch non-trash items for user ordered by created_at desc (limit window e.g. 90 days or page by cursor)
- Group by **UTC date string** in API is OK; client regroups to local TZ when rendering
- For each day: `items` = pending+active; `archived_count` = count archived that day
- Query `expand_archived=1` fills `archived_items` for requested day(s)

- [ ] **Step 2: Item block UI**

- Show body (clamp long text), image thumbs via `/f/:id` or upload download URL pattern used by files page
- Pending: hover toolbar — 垃圾 / 片段 / 笔记 / 重要  
  - Trash → `services.item.remove(id)` or patch lifecycle trash  
  - Fragment/Note/Vault → `patch({ lifecycle: 'active', importance })`
- Active: menu for reclassify / archive / trash
- After action: refresh timeline

- [ ] **Step 3: Day section**

- Header date label (今天/昨天/日期)
- Map items to ItemBlock
- Footer: if archived_count > 0 → button “已折叠 N 条归档” expands and fetches with expand

- [ ] **Step 4: Commit**

```bash
git add frontend/app/(main)/review
git commit -m "feat(item): add Review timeline with classify actions"
```

---

### Task 8: Search, Library, Vault pages

**Files:**
- Create: `frontend/app/(main)/search/page.tsx`
- Create: `frontend/app/(main)/library/page.tsx`
- Create: `frontend/app/(main)/vault/page.tsx`
- Shared: `frontend/components/common/item/item-list.tsx` (optional reuse)

- [ ] **Step 1: Search**

- Input `q` + checkboxes include_archived / include_trash
- Filters: lifecycle, importance, content_type select
- `services.item.list({...})`
- Result rows with same classify actions as Review

- [ ] **Step 2: Library**

- Chip groups for content_type / lifecycle / importance
- Always allow selecting archived/trash via lifecycle chips (when lifecycle=archived, set include_archived)
- List/grid of items

- [ ] **Step 3: Vault**

- Fixed: `importance=vault`, `include_archived=false`, `include_trash=false`
- Copy banner: “密码本内容未加密，请依赖账号安全”
- List items

- [ ] **Step 4: Commit**

```bash
git add frontend/app/(main)/search frontend/app/(main)/library frontend/app/(main)/vault frontend/components/common/item
git commit -m "feat(item): add Search, Library, and Vault pages"
```

---

### Task 9: Quality gate

**Files:** none new; fix whatever fails

- [ ] **Step 1: Format**

Run: `make prettier`  
Expected: reformatted sources

- [ ] **Step 2: Backend tests**

Run: `go test ./internal/apps/item/ ./internal/task/... -count=1`  
Expected: PASS

- [ ] **Step 3: code-check**

Run: `make code-check`  
Expected: golangci-lint clean; frontend tsc + eslint clean

- [ ] **Step 4: swagger committed**

Run: `make swagger` && git status — commit docs if dirty:

```bash
git add docs/
git commit -m "docs(swagger): regenerate after item API" || true
```

- [ ] **Step 5: Final commit if remaining fixes**

```bash
git add -A
git status
# commit only intentional fixes
git commit -m "fix(item): address lint and typecheck after MVP UI"
```

---

## Spec coverage checklist

| Spec requirement | Task |
| --- | --- |
| `c_items` / `c_item_attachments` | 1 |
| Dual-axis state + transitions | 2 |
| Create text/image/file + multi attach | 2–3, 6 |
| List default excludes archived/trash | 2–3, 8 |
| DELETE trash + force hard delete | 2–3 |
| Timeline review | 2–3, 7 |
| Stats (optional) | 2–3 |
| System days config | 1, 4 |
| Archive pending → archived+fragment | 2, 4 |
| Purge trash + RemoveOwned | 2, 4 |
| Multi-user isolation | 2 tests |
| Clip / Review / Search / Library / Vault | 6–8 |
| Sidebar | 6 |
| No tags/AI/provider | — out of scope |

## Placeholder / consistency notes

- Task constants: `item:archive_pending` / meta type `item_archive_pending` (must match schedule seed `task_type`)
- Schedule ids `2` and `3` — if conflict in env, adjust migration ids before merge
- Prefer task handlers in package `item` (not nested package) to avoid import cycles
- Frontend JSON field names: snake_case to match backend GORM json tags
- Upload business type for captures: use `type: 'clip'` on `uploadFile` for easier cleanup filters later

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-07-22-clipper-mvp.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks  
2. **Inline Execution** — this session with executing-plans checkpoints  

Which approach?
