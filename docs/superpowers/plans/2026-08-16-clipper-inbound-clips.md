# Clipper Inbound Bot → Clip Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merge current Wavelet `upstream/main` into Clipper, then persist bound Telegram/QQ private messages as clips with a rolling 60-second merge window.

**Architecture:** Isolated Clipper worktree. Fast-forward/merge Wavelet. Item domain owns `IngestInbound`. Worker emit returns handler errors so the bot ACK matches save success. Dedup table `c_item_ingest_keys`. Attachments only via `upload.Ingest`.

**Tech Stack:** Git worktrees; Go 1.25 / GORM / goose; Wavelet `pkg/message_gateway` + `internal/listener`; Clipper `internal/apps/item`; `upload.Ingest` `PolicyResolveExisting`.

**Spec:** `docs/superpowers/specs/2026-08-16-clipper-inbound-clips-design.md`

## Global Constraints

- Clipper repo only. Never commit or push to Wavelet. `upstream` push URL stays DISABLED.
- Do not reuse `.worktrees/sync-wavelet-upstream`. New branch: `feat/inbound-clips`.
- Do not stage `frontend/lib/theme/themes.json` (unrelated dirty file on main).
- Framework files: Wavelet wins on conflict; fix item/branding, not framework shape — except Task 3 (emit must return errors; required by spec ACK).
- Business tables `c_*`. No physical FKs. Dual PG + SQLite goose.
- `IngestInbound` takes `context.Context` only (no Gin).
- Web `CreateItem` always `source=web` and never merges with bot clips.
- Merge key: `user_id` + `source`; only `lifecycle=pending` and `time.Since(updated_at) < 60s`.
- Title on bot clips is always empty.
- `upload.Ingest` only; `Type: "clip"`; `SkipExtensionCheck: true`; `PolicyResolveExisting`.
- Tests: `t.TempDir()` only. Conventional Commits.
- After code: `make format` and `make code-check` (Task 8). No swagger unless a handler comment changes.

## File map

| Path | Responsibility |
| --- | --- |
| `.worktrees/feat-inbound-clips` | Isolated checkout of `feat/inbound-clips` |
| `internal/model/item.go` | `ItemSourceTelegram`, `ItemSourceQQ` |
| `internal/model/item_ingest_key.go` | `ItemIngestKey` → `c_item_ingest_keys` |
| goose `202608160004_create_c_item_ingest_keys.sql` | PG + SQLite |
| `internal/testhelper/test_helper.go` | AutoMigrate `ItemIngestKey` |
| `internal/apps/item/inbound.go` | `IngestInbound` |
| `internal/apps/item/inbound_test.go` | Create / merge / expire / source split / dedup / media / web isolation |
| `internal/listener/message_gateway.go` | Handler returns `error`; Emit returns first error |
| `internal/apps/message_gateway/runner/runner.go` | Emit wrapper returns emit error |
| `internal/platform/bootstrap/bootstrap.go` | Register `item.IngestInbound` on worker |

---

### Task 1: Worktree and merge Wavelet

**Files:** none created until merge lands

**Interfaces:**
- Produces: worktree at `.worktrees/feat-inbound-clips` on `feat/inbound-clips` containing Wavelet message-gateway + Clipper item

- [ ] **Step 1: Create worktree from Clipper main**

From `/Users/ryan/Code/Go/Clipper` (main repo, not the stale sync worktree):

```bash
git fetch upstream
git worktree add -b feat/inbound-clips .worktrees/feat-inbound-clips main
cd .worktrees/feat-inbound-clips
```

- [ ] **Step 2: Merge Wavelet**

```bash
git merge upstream/main -m "merge(upstream): Wavelet message-gateway into Clipper"
```

Conflict policy: keep Wavelet for framework paths (`internal/platform`, `internal/cmd`, `pkg/`, `go.mod` unless Clipper-only require is lost). Keep Clipper for `internal/apps/item/**`, `c_*` goose, `frontend/app/(main)/{clip,review,search,library,vault}`, `frontend/lib/services/item`, Clipper branding. Re-apply item hooks if Wavelet rewrote `v1.go` / bootstrap / sidebar.

If `go.mod` conflicts: take Wavelet module set, keep Clipper-only requires if any (there should be none extra).

- [ ] **Step 3: Compile and smoke item tests**

```bash
go test ./internal/apps/item ./pkg/message_gateway ./internal/listener ./internal/apps/message_gateway/... -count=1
```

Expected: pass. If item tests fail due to merge, fix item only.

- [ ] **Step 4: Commit conflict resolutions if the merge commit is not clean**

If merge already committed and tree is clean, skip. Otherwise:

```bash
git add -A
git status   # confirm themes.json is NOT listed
git commit --no-edit
```

---

### Task 2: Source constants and ingest-key model

**Files:**
- Modify: `internal/model/item.go`
- Create: `internal/model/item_ingest_key.go`

**Interfaces:**
- Produces: `model.ItemSourceTelegram = "telegram"`, `model.ItemSourceQQ = "qq"`
- Produces: `model.ItemIngestKey` with `TableName() == "c_item_ingest_keys"`

- [ ] **Step 1: Add constants next to `ItemSourceWeb`**

```go
// ItemSourceTelegram is a clip captured from a bound Telegram DM.
const ItemSourceTelegram = "telegram"

// ItemSourceQQ is a clip captured from a bound QQ C2C message.
const ItemSourceQQ = "qq"
```

- [ ] **Step 2: Add model**

```go
// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package model

import "time"

// ItemIngestKey makes one platform message idempotent.
type ItemIngestKey struct {
	ChannelID uint64    `json:"channel_id,string" gorm:"primaryKey"`
	MessageID string    `json:"message_id" gorm:"primaryKey;size:128"`
	UserID    uint64    `json:"user_id,string" gorm:"not null"`
	ItemID    uint64    `json:"item_id,string" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName returns c_item_ingest_keys.
func (ItemIngestKey) TableName() string { return "c_item_ingest_keys" }
```

- [ ] **Step 3: Commit**

```bash
git add internal/model/item.go internal/model/item_ingest_key.go
git commit -m "feat(item): add telegram/qq sources and ingest-key model"
```

---

### Task 3: Listener emit returns errors

**Files:**
- Modify: `internal/listener/message_gateway.go`
- Modify: `internal/listener/message_gateway_test.go`
- Modify: `internal/platform/bootstrap/bootstrap.go` (log handler `return nil`)
- Modify: `internal/apps/message_gateway/runner/runner.go` (Emit returns emit error)
- Modify: `internal/apps/message_gateway/runner/inbound_test.go` only if it stubs Emit

**Interfaces:**
- Produces: `type MessageGatewayInboundHandler func(ctx context.Context, event MessageGatewayInbound) error`
- Produces: `func EmitMessageGatewayInbound(ctx context.Context, msg message_gateway.InboundMessage) error`

- [ ] **Step 1: Change handler type and Emit**

```go
type MessageGatewayInboundHandler func(ctx context.Context, event MessageGatewayInbound) error

func EmitMessageGatewayInbound(ctx context.Context, msg message_gateway.InboundMessage) error {
	if msg.BindingUserID == nil {
		return nil
	}
	event := MessageGatewayInbound{Msg: msg}
	for _, handler := range messageGatewayInboundHandlers {
		if err := handler(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
```

Update the existing skip-unbound test: handlers must return `nil`. Add:

```go
func TestEmitMessageGatewayInbound_ReturnsHandlerError(t *testing.T) {
	// reset is not exported; register a failing handler and assert Emit returns it
	want := errors.New("save failed")
	OnMessageGatewayInbound(func(ctx context.Context, ev MessageGatewayInbound) error {
		return want
	})
	uid := uint64(1)
	err := EmitMessageGatewayInbound(context.Background(), message_gateway.InboundMessage{BindingUserID: &uid, Text: "x"})
	if !errors.Is(err, want) {
		t.Fatalf("Emit() error = %v, want %v", err, want)
	}
}
```

Note: handlers persist on the package slice. Existing test already appends. This test may see prior handlers; if the first registered handler returns nil, the new one still runs. Fine.

If tests pollute later tests, document that order is append-only (same as today).

- [ ] **Step 2: Wavelet log handler returns nil**

In `RegisterMessageGatewayListeners`, change the func to `error` and `return nil`.

- [ ] **Step 3: Runner production Emit**

```go
Emit: func(ctx context.Context, msg message_gateway.InboundMessage) error {
	return listener.EmitMessageGatewayInbound(ctx, msg)
},
```

Do **not** register `item.IngestInbound` yet (Task 7).

- [ ] **Step 4: Tests**

```bash
go test ./internal/listener ./internal/apps/message_gateway/runner -count=1
```

- [ ] **Step 5: Commit**

```bash
git commit -m "fix(message-gateway): propagate inbound handler errors to bot ACK"
```

---

### Task 4: Goose + AutoMigrate

**Files:**
- Create: `internal/infra/persistence/migrator/goose/postgres/202608160004_create_c_item_ingest_keys.sql`
- Create: `internal/infra/persistence/migrator/goose/sqlite/202608160004_create_c_item_ingest_keys.sql`
- Modify: `internal/testhelper/test_helper.go` AutoMigrate list
- Modify: `internal/infra/persistence/migrator/migrator_test.go` only if a table-count assertion breaks

**Interfaces:**
- Produces: `c_item_ingest_keys` in both dialects

- [ ] **Step 1: Postgres**

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS c_item_ingest_keys (
    channel_id BIGINT NOT NULL,
    message_id VARCHAR(128) NOT NULL,
    user_id BIGINT NOT NULL,
    item_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (channel_id, message_id)
);

-- +goose Down
DROP TABLE IF EXISTS c_item_ingest_keys;
```

- [ ] **Step 2: SQLite** — same, `DATETIME` for `created_at`.

- [ ] **Step 3: AutoMigrate `&model.ItemIngestKey{}`**

- [ ] **Step 4:**

```bash
go test ./internal/infra/persistence/migrator ./internal/apps/item -count=1
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(item): add c_item_ingest_keys goose migrations"
```

---

### Task 5: IngestInbound text create + rolling merge (TDD)

**Files:**
- Test: `internal/apps/item/inbound_test.go`
- Create: `internal/apps/item/inbound.go`
- Modify: `internal/apps/item/logics.go` only if create-with-source must be extracted

**Interfaces:**
- Consumes: `model.ItemSourceTelegram`, `model.ItemSourceQQ`, `CreateItem` helpers, `detectContentType`
- Produces: `func IngestInbound(ctx context.Context, msg message_gateway.InboundMessage) error`

- [ ] **Step 1: Write tests first**

```go
func uidPtr(id uint64) *uint64 { return &id }

func tgMsg(userID uint64, mid, text string) message_gateway.InboundMessage {
	return message_gateway.InboundMessage{
		ChannelID:      10,
		ChannelType:    model.ItemSourceTelegram,
		PlatformUserID: "tg-1",
		MessageID:      mid,
		Text:           text,
		BindingUserID:  uidPtr(userID),
	}
}

func TestIngestInbound_CreatesTelegramText(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	if err := IngestInbound(ctx, tgMsg(1001, "m1", "hello")); err != nil {
		t.Fatalf("IngestInbound() error = %v", err)
	}
	list, err := ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if err != nil || list.Total != 1 {
		t.Fatalf("ListItems() total = %d err = %v, want 1", list.Total, err)
	}
	got := list.Results[0]
	if got.Source != model.ItemSourceTelegram || got.Body != "hello" || got.Title != "" {
		t.Fatalf("item = %+v", got)
	}
}

func TestIngestInbound_MergesWithin60s(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	if err := IngestInbound(ctx, tgMsg(1001, "m1", "a")); err != nil {
		t.Fatal(err)
	}
	if err := IngestInbound(ctx, tgMsg(1001, "m2", "b")); err != nil {
		t.Fatal(err)
	}
	list, _ := ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if list.Total != 1 || list.Results[0].Body != "a\nb" {
		t.Fatalf("merge = %+v", list)
	}
}

func TestIngestInbound_NewClipAfter60s(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	if err := IngestInbound(ctx, tgMsg(1001, "m1", "a")); err != nil {
		t.Fatal(err)
	}
	list, _ := ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if err := db.DB(ctx).Model(&model.Item{}).Where("id = ?", list.Results[0].ID).
		Update("updated_at", time.Now().Add(-61*time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	if err := IngestInbound(ctx, tgMsg(1001, "m2", "b")); err != nil {
		t.Fatal(err)
	}
	list, _ = ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if list.Total != 2 {
		t.Fatalf("total = %d, want 2", list.Total)
	}
}

func TestIngestInbound_SkipsNonPending(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	_ = IngestInbound(ctx, tgMsg(1001, "m1", "a"))
	list, _ := ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	active := model.ItemLifecycleActive
	if _, err := PatchItem(ctx, 1001, list.Results[0].ID, PatchItemInput{Lifecycle: &active}); err != nil {
		t.Fatal(err)
	}
	_ = IngestInbound(ctx, tgMsg(1001, "m2", "b"))
	list, _ = ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10, IncludeArchived: true})
	if list.Total != 2 {
		t.Fatalf("total = %d, want 2", list.Total)
	}
}

func TestIngestInbound_DoesNotMergeQQWithTelegram(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	_ = IngestInbound(ctx, tgMsg(1001, "m1", "a"))
	qq := tgMsg(1001, "m2", "b")
	qq.ChannelType = model.ItemSourceQQ
	_ = IngestInbound(ctx, qq)
	list, _ := ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if list.Total != 2 {
		t.Fatalf("total = %d, want 2", list.Total)
	}
}

func TestIngestInbound_WebCreateDoesNotMerge(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	if _, err := CreateItem(ctx, 1001, CreateItemInput{Body: "web"}); err != nil {
		t.Fatal(err)
	}
	if err := IngestInbound(ctx, tgMsg(1001, "m1", "bot")); err != nil {
		t.Fatal(err)
	}
	list, _ := ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if list.Total != 2 {
		t.Fatalf("total = %d, want 2", list.Total)
	}
}

func TestIngestInbound_EmptyRejected(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	err := IngestInbound(context.Background(), tgMsg(1001, "m1", "  "))
	if err == nil || err.Error() != errEmptyContent {
		t.Fatalf("error = %v, want %s", err, errEmptyContent)
	}
}

func TestIngestInbound_NilBindingRejected(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	msg := tgMsg(1001, "m1", "x")
	msg.BindingUserID = nil
	if err := IngestInbound(context.Background(), msg); err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Step 2: `go test ./internal/apps/item -run TestIngestInbound -count=1` — fail compile**

- [ ] **Step 3: Implement `IngestInbound`**

```go
const inboundMergeWindow = 60 * time.Second

func IngestInbound(ctx context.Context, msg message_gateway.InboundMessage) error {
	if msg.BindingUserID == nil {
		return errors.New(errEmptyContent)
	}
	userID := *msg.BindingUserID
	source := strings.TrimSpace(msg.ChannelType)
	if source != model.ItemSourceTelegram && source != model.ItemSourceQQ {
		return errors.New(errEmptyContent)
	}
	uploadIDs, err := ingestInboundAttachments(ctx, userID, msg.Attachments)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(msg.Text)
	if text == "" && len(uploadIDs) == 0 {
		return errors.New(errEmptyContent)
	}
	if msg.MessageID != "" {
		var existing model.ItemIngestKey
		err := db.DB(ctx).Where("channel_id = ? AND message_id = ?", msg.ChannelID, msg.MessageID).First(&existing).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(errInternal)
		}
	}
	// find last pending clip; merge or create (Task 6 fills attachments)
	...
}
```

`ingestInboundAttachments` may return nil, nil in this task (attachments in Task 6).

Merge SQL:

```go
var last model.Item
err := db.DB(ctx).Where("user_id = ? AND source = ? AND lifecycle = ?", userID, source, model.ItemLifecyclePending).
	Order("updated_at DESC").First(&last).Error
```

If `err == nil && time.Since(last.UpdatedAt) < inboundMergeWindow` → append body, save, write ingest key.

Else call a local `createItemWithSource(ctx, userID, source, text, uploadIDs)` copied from `CreateItem` but `item.Source = source` and `Title = ""`.

Dedup insert in the same transaction as create/append:

```go
if msg.MessageID != "" {
	key := model.ItemIngestKey{ChannelID: msg.ChannelID, MessageID: msg.MessageID, UserID: userID, ItemID: item.ID}
	if err := tx.Create(&key).Error; err != nil {
		return err
	}
}
```

If unique conflict, return `nil` (check SQLite/Postgres error or `First` before write as above).

- [ ] **Step 4: Tests pass**

```bash
go test ./internal/apps/item -run TestIngestInbound -count=1
```

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(item): ingest bound telegram/qq text into clips"
```

---

### Task 6: Dedup + attachments

**Files:**
- Modify: `internal/apps/item/inbound.go`
- Modify: `internal/apps/item/inbound_test.go`

**Interfaces:**
- Consumes: `upload.Ingest`, `upload.PolicyResolveExisting`
- Produces: attachment rows on create/merge; idempotent `message_id`

- [ ] **Step 1: Tests**

```go
func TestIngestInbound_DedupSameMessageID(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	msg := tgMsg(1001, "same", "hello")
	if err := IngestInbound(ctx, msg); err != nil {
		t.Fatal(err)
	}
	if err := IngestInbound(ctx, msg); err != nil {
		t.Fatal(err)
	}
	list, _ := ListItems(ctx, 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if list.Total != 1 || list.Results[0].Body != "hello" {
		t.Fatalf("dedup = %+v", list)
	}
}

func TestIngestInbound_ImageCaption(t *testing.T) {
	// copy objectstore.MockStorage setup from ingest_test.go
	restore, disable := setupInboundMockStorage(t)
	defer restore()
	defer disable()
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	dir := t.TempDir()
	path := filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(path, []byte("jpeg-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	msg := tgMsg(1001, "img1", "caption")
	msg.Attachments = []message_gateway.Attachment{{Path: path, FileName: "photo.jpg", MIME: "image/jpeg"}}
	if err := IngestInbound(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	list, _ := ListItems(context.Background(), 1001, ListItemsQuery{Page: 1, PageSize: 10})
	if list.Total != 1 || len(list.Results[0].Attachments) != 1 {
		t.Fatalf("attachments = %+v", list.Results[0])
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("temp file must be removed")
	}
}
```

`setupInboundMockStorage` is the same mock as `internal/apps/upload/ingest/ingest_test.go` (`objectstore.MockStorage` + `IsEnabledFunc = true`). Duplicate the helper in `inbound_test.go` (do not export from ingest tests).

- [ ] **Step 2: Implement attachment ingest**

```go
func ingestInboundAttachments(ctx context.Context, userID uint64, atts []message_gateway.Attachment) ([]uint64, error) {
	var ids []uint64
	for _, att := range atts {
		path := att.Path
		if att.Error != "" || path == "" {
			logger.WarnF(ctx, "item inbound skip attachment: %s", att.Error)
			continue
		}
		data, err := os.ReadFile(path)
		_ = os.Remove(path)
		if err != nil {
			logger.WarnF(ctx, "item inbound read attachment: %v", err)
			continue
		}
		sum := sha256.Sum256(data)
		hash := hex.EncodeToString(sum[:])
		res, err := upload.Ingest(ctx, upload.IngestRequest{
			UserID:              userID,
			Reader:              bytes.NewReader(data),
			Size:                int64(len(data)),
			FileName:            att.FileName,
			MimeType:            att.MIME,
			Hash:                hash,
			Type:                "clip",
			Policy:              upload.PolicyResolveExisting,
			SkipExtensionCheck:  true,
		})
		if err != nil {
			logger.WarnF(ctx, "item inbound ingest: %v", err)
			continue
		}
		ids = append(ids, res.Upload.ID)
	}
	return ids, nil
}
```

On merge, load existing attachments, append new ones with `sort = max+1, max+2, ...`.

- [ ] **Step 3:** `go test ./internal/apps/item -run TestIngestInbound -count=1`

- [ ] **Step 4: Commit**

```bash
git commit -m "feat(item): dedup inbound messages and ingest bot attachments"
```

---

### Task 7: Register Clipper handler

**Files:**
- Modify: `internal/platform/bootstrap/bootstrap.go`

**Interfaces:**
- Consumes: `item.IngestInbound`, `listener.OnMessageGatewayInbound`

- [ ] **Step 1: In `RegisterWorker` and `RegisterAll` (after Wavelet `RegisterMessageGatewayListeners`)**

```go
func registerClipperInbound() {
	listener.OnMessageGatewayInbound(func(ctx context.Context, event listener.MessageGatewayInbound) error {
		return item.IngestInbound(ctx, event.Msg)
	})
}
```

Call from `RegisterWorker` and `RegisterAll` inside the existing `sync.Once` blocks or a new `registerClipperInboundOnce`. Do not call from `RegisterAPI`. Keep the Wavelet log handler.

- [ ] **Step 2:** `go test ./internal/platform/bootstrap ./internal/apps/item -count=1`

- [ ] **Step 3: Commit**

```bash
git commit -m "feat(item): register inbound clip writer on worker"
```

---

### Task 8: Repo gates

**Files:** format leftovers only

- [ ] **Step 1:** `go test ./... -count=1`
- [ ] **Step 2:** `make format && node frontend/scripts/check-i18n-keys.mjs && make code-check`
- [ ] **Step 3:** Confirm `CreateItem` still sets `source=web`; no `init()` in `apps/item`; Wavelet remotes untouched
- [ ] **Step 4: Commit leftover format**

```bash
git commit -m "chore(item): format after inbound clip ingest"
```

(Skip if clean.)

Then use `superpowers:finishing-a-development-branch` — do not merge/push without asking.

---

## Spec coverage

| Spec | Task |
|---|---|
| Merge Wavelet | 1 |
| Source constants + ingest key model | 2 |
| ACK on ingest error | 3 |
| `c_item_ingest_keys` goose | 4 |
| Text create + 60s merge + pending + QQ split + web isolation | 5 |
| Dedup + `upload.Ingest` + temp delete | 6 |
| Bootstrap register | 7 |
| Gates | 8 |
| UI source badge | **out of scope** |
