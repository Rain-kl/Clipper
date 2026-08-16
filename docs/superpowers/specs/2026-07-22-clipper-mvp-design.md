# Clipper MVP Design

**Date:** 2026-07-22  
**Status:** Approved for implementation planning  
**Product:** Clipper — multi-user capture & graded retention (WeChat File Transfer–like)

## 1. Goal

Users capture ephemeral content (ideas, temporary passwords, cross-device files) into Clipper. Every capture is recorded, classified on two axes (lifecycle + importance), auto-archived or purged by policy, and browsed via Clip / Review / Search / Library / Vault.

### MVP scope (in)

- Multi-user SaaS: each user owns isolated items (`user_id`)
- Web capture: text + images + files (via existing `upload.Ingest`)
- Dual-axis state: `lifecycle` × `importance`
- Manual classify (trash / fragment / note / vault)
- Review timeline (by day, archived folded)
- Search with keyword + filters (default exclude archived/trash)
- Library: browse by content type / lifecycle / importance
- Vault: filter view for `importance=vault` (no field encryption)
- System-configurable auto rules: pending → archive; trash → hard delete
- Product is first-class domain (`item`), not a demo nested under a platform plugin mindset

### MVP scope (out)

- Tags
- AI auto-classify / auto-tag
- Telegram / QQ / other providers
- Vault field-level encryption
- PostgreSQL `tsvector` full-text
- Shared items across users
- Realtime WebSocket feed
- Chat-style multi-turn history on Clip page

### Later hooks (design only)

- `source` column (`web` now; `telegram` etc. later)
- Provider adapters write the same `c_items` table
- AI async task updates lifecycle/importance (+ future tags)
- Per-user retention overrides of system defaults

## 2. Architecture approach

**Chosen:** First-class Item domain on Clipper platform capabilities.

| Layer | Choice |
| --- | --- |
| Domain package | `internal/apps/item` (product core, not scaffold demo) |
| Tables | Business prefix `c_*` (`c_items`, `c_item_attachments`); framework keeps `w_*` |
| Files | Only `upload.Ingest` / `upload.RemoveOwned`; no direct `w_uploads` writes |
| Jobs | Asynq handlers + scheduler; register via `internal/bootstrap`, no `init()` side effects |
| Routes | Register only through `internal/router/v1` → `item.RegisterRoutes` |
| Config | System settings: archive/purge day thresholds |
| Frontend | Next.js App Router pages under `(main)`; `ItemService` in `frontend/lib/services/item` |

**Rejected alternatives**

- Full inbox message bus before Web MVP (too heavy)
- Minimal single-table CRUD with no lifecycle tasks (hard to extend)

## 3. Domain model

### 3.1 Entity: Item

One **Item** = one capture unit.

| Field | Type | Notes |
| --- | --- | --- |
| `id` | snowflake | PK |
| `user_id` | snowflake | Owner; required on every query |
| `content_type` | enum | `text` \| `image` \| `file` |
| `title` | string, optional | Nullable; UI may derive from body prefix |
| `body` | text, optional | Main text; may be empty for pure media |
| `lifecycle` | enum | see below |
| `importance` | enum | see below |
| `source` | string | MVP always `web` |
| `created_at` | timestamptz | |
| `updated_at` | timestamptz | |
| `archived_at` | timestamptz, null | Set when entering `archived` |
| `trashed_at` | timestamptz, null | Set when entering `trash`; drives purge |

### 3.2 Attachments

Table `c_item_attachments`:

| Field | Notes |
| --- | --- |
| `id` | snowflake |
| `item_id` | FK-like index to `c_items` (no physical FK) |
| `upload_id` | Existing upload record |
| `sort` | Display order |
| `created_at` | |

**`content_type` assignment rules on create**

| Input | `content_type` |
| --- | --- |
| body only, no uploads | `text` |
| no body, single image upload | `image` |
| no body, single non-image upload | `file` |
| body + one or more uploads | `text`; uploads → attachments |
| multiple uploads, no body | primary from first MIME (`image`/`file`); rest attachments |
| empty body and no uploads | **reject 400** |

### 3.3 Dual-axis state

**`lifecycle`**

| Value | Meaning |
| --- | --- |
| `pending` | Unprocessed (default on create) |
| `active` | Classified and in the normal library |
| `archived` | Archived; excluded from default search |
| `trash` | Soft-deleted / pending purge; excluded from default search |

**`importance`**

| Value | Meaning |
| --- | --- |
| `none` | Unspecified; required while `lifecycle=pending` at create |
| `fragment` | Snippet / light keep |
| `note` | Normal note |
| `vault` | Important / password-book |

### 3.4 Transitions (user)

```
pending  --classify fragment|note|vault-->  active (+ importance)
pending  --mark trash------------------->  trash  (set trashed_at)
active   --mark trash------------------->  trash  (set trashed_at)
active   --manual archive--------------->  archived (set archived_at)
trash    --restore---------------------->  active (keep importance;
                                          if importance was none → pending)
archived --unarchive-------------------->  active
```

Invalid combinations (e.g. `lifecycle=pending` with `importance=note` on create) are rejected with 400.

### 3.5 Automatic transitions (jobs)

| Job | Default | Effect |
| --- | --- | --- |
| `item:archive-pending` | 3 days after `created_at` while still `pending` | `lifecycle=archived`, `importance=fragment`, set `archived_at` |
| `item:purge-trash` | 30 days after `trashed_at` while `trash` | Hard delete item, attachment rows, `upload.RemoveOwned` for owned uploads |

System settings (keys illustrative):

- `item.pending_archive_after_days` (default `3`)
- `item.trash_purge_after_days` (default `30`)

### 3.6 Tags

Not in MVP. Future: `c_tags` + `c_item_tags`.

## 4. API

Base: authenticated (`oauth.LoginRequired()`). All handlers scope by current `user_id`. Missing or other-user id → **404** (no existence leak).

| Method | Path | Behavior |
| --- | --- | --- |
| `POST` | `/v1/items` | Create; default `pending` + `importance=none`, `source=web` |
| `GET` | `/v1/items` | Paginated list; filters + `q` (body `ILIKE`); default lifecycle ∈ {pending, active} |
| `GET` | `/v1/items/:id` | Detail + attachment metadata |
| `PATCH` | `/v1/items/:id` | Update title/body; transition lifecycle/importance |
| `DELETE` | `/v1/items/:id` | Move to `trash` (set `trashed_at`) |
| `DELETE` | `/v1/items/:id?force=1` | Immediate hard delete + attachment cleanup |
| `GET` | `/v1/items/timeline` | Review: grouped by calendar day; archived count per day; optional expand archived |
| `GET` | `/v1/items/stats` | Optional counts for nav badges |

### 4.1 Create body (example)

```json
{
  "body": "an idea",
  "upload_ids": ["..."],
  "title": "optional"
}
```

### 4.2 Classify body (example)

```json
{ "lifecycle": "active", "importance": "note" }
```

```json
{ "lifecycle": "trash" }
```

### 4.3 List / search defaults

- Default: `lifecycle IN (pending, active)`
- Opt-in: `include_archived=true`, `include_trash=true`
- Filters: `importance`, `content_type`, date range, `q`

### 4.4 Response envelope

Platform standard:

```json
{ "error_msg": "", "data": ... }
```

Failures via `response.Abort*` only (never `c.JSON` + `response.Err` in handlers).

### 4.5 Error mapping

| Case | Response |
| --- | --- |
| Bind / validation / illegal transition | 400 |
| Unauthenticated | 401 |
| Not found / not owner | 404 |
| Internal | 500 + logged |

## 5. Backend structure

```
internal/apps/item/
  routers.go      # RegisterRoutes
  handlers.go     # HTTP bind + Abort/OK
  logics.go       # context.Context business (API + worker share)
  errs.go         # camelCase user-facing string constants
  task/
    archive_pending.go
    purge_trash.go
internal/model/item.go
internal/model/item_attachment.go
internal/db/migrator/goose/...  # c_items, c_item_attachments
```

- Complex SQL stays in model / item logics, not fat handlers
- Swagger comments on every public handler; run `make swagger` after API changes
- After code: `make code-check`, `make prettier`

## 6. Frontend information architecture

| Nav | Route (draft) | Role |
| --- | --- | --- |
| Clip | `/clip` (or product home) | Large composer; paste image / pick files → upload → `POST /v1/items` |
| Review | `/review` | Day waterfall; Notion-like blocks; pending hover classify |
| Search | `/search` | Keyword + filters |
| Library | `/library` | Faceted by type / lifecycle / importance |
| Vault | `/vault` | `importance=vault` and `lifecycle IN (pending, active)` only |

Admin (`/admin`) remains platform admin; Clipper user app is separate nav in main layout.

### 6.1 Clip

- Large multi-line input; attach images/files via existing upload APIs
- Success: clear composer + toast; no chat history in MVP

### 6.2 Review

- `GET /v1/items/timeline` returns chronological items (and per-day archived counts); **client groups into calendar days in the viewer’s local timezone**
- Within day: newest first, block UI
- **Pending:** hover (desktop) or overflow menu (touch) → Trash / Fragment / Note / Vault
- **Active:** change importance, trash, manual archive
- **Archived:** bottom of day card “N archived collapsed”; expand loads archived for that day
- **Trash:** hidden from Review (visible via Search/Library with filters)

### 6.3 Search

- Shared list API semantics; default excludes archived/trash
- Result list (not day-grouped); row actions same as Review

### 6.4 Library

- Chips/tabs: content_type, lifecycle, importance
- Grid or list of filtered items

### 6.5 Vault

- Fixed filter vault + non-trash active/pending
- Visual cue (lock); copy that security depends on account (no encryption)

### 6.6 Client service

```
frontend/lib/services/item/
  types.ts
  item.service.ts
  index.ts
```

Register on exported `services` object. Page chrome follows project title-bar rules (`py-6`, `h1` standards, Lucide `size-5 text-primary`).

## 7. Security & multi-tenancy

- Login required for all item routes
- Every read/write filters `user_id = current user`
- Uploads must belong to current user before attach
- Hard delete cleans owned uploads to avoid orphans
- Vault content stored like other items (plaintext at rest in DB for MVP)

## 8. Testing strategy

| Layer | Coverage |
| --- | --- |
| logics | create rules, transitions, user isolation |
| tasks | pending→archived+fragment; trash purge + RemoveOwned |
| handlers | 401 unauthenticated; 404 cross-user id |
| frontend E2E | optional after API stable |

Tests must call bootstrap registration explicitly when depending on tasks; use `t.TempDir()` only (no hardcoded upload paths in-repo).

## 9. Implementation phases (for planning skill)

1. DB migration (`c_items`, `c_item_attachments`) + models  
2. logics + handlers + routes + swagger  
3. System settings for day thresholds  
4. Asynq tasks + scheduler + bootstrap register  
5. Frontend service + Clip page  
6. Review timeline UI + classify actions  
7. Search + Library + Vault pages  
8. `make code-check` / `make prettier` / smoke verification  

## 10. Success criteria

- Logged-in user can capture text/image/file and see it in Review the same day  
- Pending item can be classified to fragment/note/vault or trash  
- Search default omits archived/trash; opt-in works  
- After configured days, pending becomes archived+fragment; trash hard-deletes with files  
- User A cannot read/write User B’s items  
- No bypass of upload ingest/remove APIs  

## 11. Open decisions resolved in brainstorming

| Topic | Decision |
| --- | --- |
| MVP scope | Web core loop only |
| Tenancy | Multi-user SaaS isolation |
| State model | Dual-axis lifecycle + importance |
| Tags | Deferred |
| Media | Text + image + file; text may have multi-attachments |
| Vault | Filter view, no encryption |
| Search | Keyword ILIKE + filters |
| Package name | `item` (product core) |
| Table prefix | `c_*` business / `w_*` framework |
| Pending timeout | → `archived` + `importance=fragment` |
| DELETE | soft trash; `force=1` hard delete |
| Implementation style | First-class domain + platform reuse |
