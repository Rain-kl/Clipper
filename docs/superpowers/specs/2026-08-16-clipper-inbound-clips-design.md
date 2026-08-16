# Clipper Inbound Bot → Clip Design

**Date:** 2026-08-16  
**Status:** Approved for implementation planning  
**Product:** Clipper  
**Depends on:** Wavelet message gateway already on `upstream/main` (`a8fcf60`)

## 1. Goal

Bring current Wavelet into Clipper, then turn bound Telegram / QQ private messages into Clipper clips: one message is one capture, consecutive messages within 60 seconds append to the same pending clip.

### In scope

- Clipper-only `git fetch upstream && git merge upstream/main` (Wavelet remains fetch-only; never write Wavelet).
- Register a Clipper product listener for `message_gateway.inbound`.
- `item.IngestInbound`: attachments via `upload.Ingest`, then create or append a `c_items` row.
- Rolling 60-second merge on the last **pending** clip for the same user + `source`.
- Idempotency on `(channel_id, message_id)` so a retried DM does not double-write.
- Tests for create / merge / expiry / classified skip / media / dedup / web isolation.

### Out of scope

- Editing Wavelet or pushing Clipper `origin` unless asked.
- Group / guild messages (gateway already drops them).
- Changing pairing, admin channel UI, or profile bind card (already in Wavelet).
- Tags, AI classify, UI source badges (list already exposes `source`; no new frontend required for v1).
- Merging across Telegram and QQ, or a per-`channel_id` split (v1: one window per user per `source`).
- Burst merge for web-created clips.

## 2. Locked decisions

| Topic | Choice |
|---|---|
| Mapping | One inbound message is one capture unit |
| Merge | Rolling 60s append into the last matching **pending** clip |
| Match key | `user_id` + `source` (`telegram` or `qq`) |
| Clock | Last matching clip `updated_at` within 60 seconds |
| Classified clips | `lifecycle != pending` never receives appends |
| Files | Only `upload.Ingest` / no direct `w_uploads` writes |
| Ingest policy | `PolicyResolveExisting` |
| Empty payload | No text and no successful attachment → error (bot “could not save”) |
| Web create | Still `source=web`; never enters the merge window |
| Wavelet ACK | Unchanged: ingest error → “could not save your message”; success → “received” |

## 3. Architecture

```
git fetch upstream && git merge upstream/main     (Clipper worktree)

Bound DM → Worker adapter → pairing
        → EmitMessageGatewayInbound
        → Wavelet log handler (keep)
        → Clipper item.IngestInbound          (new)
              ├─ ingest attachments → upload IDs
              ├─ dedup (channel_id, message_id)
              ├─ find last pending clip for user+source, updated_at >= now-60s
              │     yes → append body + attachments, bump updated_at
              │     no  → CreateItem-equivalent with source=telegram|qq
              └─ delete local temp files
```

Work happens on a new Clipper branch in `.worktrees/` (do not reuse stale `sync/wavelet-upstream`). Resolve conflicts with the existing rule: Wavelet wins framework files; Clipper item/branding stays and adapts.

After merge, Wavelet already registers a log-only `OnMessageGatewayInbound` from bootstrap. Clipper **adds** a second handler in `RegisterWorker` / `RegisterAll` (events only fire on Worker). Do not remove the log handler. Do not use `init()`.

### 3.1 Packages

| Path | Role |
|---|---|
| `internal/apps/item/inbound.go` | `IngestInbound(ctx, msg) error` — no Gin |
| `internal/apps/item/logics.go` | Extract or reuse create/append helpers; web `CreateItem` stays `source=web` |
| `internal/model/item.go` | `ItemSourceTelegram = "telegram"`, `ItemSourceQQ = "qq"` (already have `ItemSourceWeb`) |
| goose `c_item_ingest_keys` | Dedup rows (PG + SQLite) |
| `internal/platform/bootstrap` | `OnMessageGatewayInbound(item.IngestInbound)` from worker registration |
| tests | `inbound_test.go` with `t.TempDir()` / upload mocks |

`pkg/message_gateway` stays framework. Item must not import adapters.

## 4. IngestInbound

Signature:

```go
func IngestInbound(ctx context.Context, msg message_gateway.InboundMessage) error
```

`userID` is `*msg.BindingUserID` (listener already skips nil). `source` is `msg.ChannelType` and must be `telegram` or `qq`; anything else returns error.

### 4.1 Attachments

For each `msg.Attachments` entry:

- Skip if `Error != ""` or `Path == ""` (log at warn).
- Read bytes from `Path`, SHA-256 hex, `upload.Ingest` with:
  - `UserID: userID`
  - `FileName` from attachment
  - `MimeType` if present
  - `Type: "clip"`
  - `Policy: upload.PolicyResolveExisting`
  - `SkipExtensionCheck: true` (bot filenames are untrusted; hash + mime are enough)
- Collect successful upload IDs.
- Always `os.Remove(Path)` in a defer/cleanup pass (success or fail).

### 4.2 Empty check

After attachment ingest: if `strings.TrimSpace(msg.Text) == ""` and there are no upload IDs → return a sentinel the runner already treats as failure (`errors.New` with a stable item-domain message, e.g. existing `errEmptyContent`).

### 4.3 Dedup

Table `c_item_ingest_keys`:

| Column | Notes |
|---|---|
| `channel_id` | `msg.ChannelID` |
| `message_id` | `msg.MessageID`; empty string is allowed but then skip insert (cannot dedup) |
| `user_id` | owner |
| `item_id` | clip that absorbed this message |
| `created_at` | |

Primary / unique: `(channel_id, message_id)` where `message_id != ''`.

Insert **before** create/append (same transaction as the item write). Unique violation → return `nil` (idempotent success). Optionally delete rows older than 24 hours in the worker reload loop or on ingest; not required for correctness.

### 4.4 Merge vs create

Load the latest `c_items` row where:

- `user_id = BindingUserID`
- `source = ChannelType`
- `lifecycle = pending`
- order by `updated_at DESC`, limit 1

If found and `time.Since(item.UpdatedAt) < 60*time.Second`:

- Append text: if new text non-empty, `body = old + "\n" + new` when old is non-empty, else `body = new`.
- Never change `title` on merge (bot clips are body-first).
- Attach new upload IDs with `sort` continuing after existing attachments.
- Recompute `content_type` with existing `detectContentType(body, allUploads)` (today: any non-empty body → `text`, even with files).
- Save item so `updated_at` refreshes (rolling window).

Else: insert a new item like `CreateItem`, but `Source` is `telegram` or `qq` and `Title` is empty (body holds the text).

### 4.5 Errors

| Situation | Result |
|---|---|
| Some attachments fail, text or other files remain | Success; failed files logged |
| No text and zero successful attachments | Error → bot “could not save” |
| Dedup hit | `nil` → bot “received” |
| Last clip not pending or older than 60s | New clip |
| Web `CreateItem` | Unchanged |

Do not surface internal errors to the bot.

## 5. Data model additions

`c_items.source` already exists (`web` default). No item schema change except constants.

New goose (both dialects), no physical FKs:

```sql
CREATE TABLE IF NOT EXISTS c_item_ingest_keys (
    channel_id BIGINT NOT NULL,
    message_id VARCHAR(128) NOT NULL,
    user_id BIGINT NOT NULL,
    item_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (channel_id, message_id)
);
```

SQLite: same columns, `DATETIME` for `created_at` to match existing Clipper sqlite style.

Add models to AutoMigrate in `internal/testhelper`.

## 6. Testing

Use `testhelper.SetupTestEnvironment` and `t.TempDir()` only. Mock storage via existing upload test helpers.

| Case | Expect |
|---|---|
| First telegram text | New item, `source=telegram`, body set |
| Second text 59s later same user/source | Same `id`, body `"a\nb"` |
| Second text 61s later | New item |
| Last item `lifecycle=active` | New item |
| QQ vs telegram | Do not merge |
| Image + caption | `upload.Ingest` + attachment row |
| Repeat same `message_id` | No second item / no double append |
| Web create | `source=web`; not merged with bot clip |

Repo gates after implementation: `go test ./...`, `make swagger` if handlers change (none expected), `make format`, `make code-check`.

## 7. Manual check

1. Merge Wavelet, run Clipper worker + API.
2. Admin: add Telegram (or QQ) channel.
3. Profile: bind with pairing code.
4. DM “hello” → new clip on `/`.
5. DM “world” within 60s → same clip body has both lines.
6. Wait 61s, DM again → new clip.
7. Send a photo → attachment visible on the clip.

## 8. Risks

- Merge conflicts on bootstrap / go.mod / frontend i18n after 18 Wavelet commits; resolve per existing sync policy.
- Clipper `main` has an uncommitted `frontend/lib/theme/themes.json` change; do not include it in this work unless the user asks.
- Stale worktree `.worktrees/sync-wavelet-upstream` must not be reused.
- `detectContentType` treats any non-empty body as `text`; bot image+caption will be `text`. Keep this consistent with web; do not special-case.
