# Message Gateway Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship Wavelet’s inbound message gateway (Telegram + QQ private chat), admin channel cards, profile pairing, and `message_gateway.inbound` events — no Clipper clip ingest.

**Architecture:** `pkg/message_gateway` defines Channel/Registry/pairing with zero Gin/GORM. Worker runs adapters (telebot long poll, official botgo C2C). API does CRUD and bind/unbind. Bound inbound messages emit `listener.EmitMessageGatewayInbound`.

**Tech Stack:** Go 1.25, Gin, GORM, goose, `gopkg.in/telebot.v4`, official QQ `botgo`, Next.js, next-intl.

**Spec:** `docs/superpowers/specs/2026-08-16-message-gateway-design.md`

## Global Constraints

- Wavelet repo only. Do not edit Clipper.
- `pkg/message_gateway` must not import Gin, GORM, sessions, or `internal/apps`.
- Tables `w_*`. No physical FKs. Dual PG + SQLite goose.
- API errors only `response.Abort*`.
- No `init()` cross-module registration; wire in `internal/platform/bootstrap` and `internal/cmd`.
- Routes only via `internal/router/v1`.
- v1: private chat / C2C only; groups logged and dropped.
- Pairing: alphabet `ABCDEFGHJKLMNPQRSTUVWXYZ23456789`, length 8, display `XXXX-XXXX`, TTL 15 minutes, one-time.
- Event name string is exactly `message_gateway.inbound`.
- Channel types: `telegram`, `qq`. `owner_scope` always `system` in v1.
- Encrypt credentials with `pkg/util.Encrypt` using `hex.EncodeToString(sha256(session_secret))` as the 64-char key.
- After handlers: `make swagger`. After code: `make format` and `make code-check`.
- Commits: Conventional Commits.
- Tests: `t.TempDir()` only.
- Execution: `superpowers:using-git-worktrees` in Wavelet.

## File map

| Path | Responsibility |
| --- | --- |
| `pkg/message_gateway/types.go` | Capability, Attachment, InboundMessage, OutboundMessage, Recipient |
| `pkg/message_gateway/channel.go` | `Channel` interface, `Handler` |
| `pkg/message_gateway/registry.go` | Register / Lookup factories |
| `pkg/message_gateway/pairing.go` | `GenerateCode`, `NormalizeCode`, `FormatCode` |
| `pkg/message_gateway/channel/telegram` | telebot private-chat adapter |
| `pkg/message_gateway/channel/qq` | botgo C2C adapter |
| `internal/model/message_gateway.go` | GORM models |
| goose `202608160003_create_message_gateway.sql` | PG + SQLite |
| `internal/repository/message_gateway.go` | persistence |
| `internal/listener/message_gateway.go` | event + emit |
| `internal/apps/message_gateway` | user bind logics + handlers |
| `internal/apps/admin/message_gateway` | admin CRUD |
| `internal/apps/message_gateway/runner` | Worker Start / reload / inbound handler |
| `internal/cmd/worker.go` | start runner before Asynq |
| `internal/router/v1/admin.go` + new files | HTTP routes |
| frontend admin + profile + i18n | UI |

---

### Task 1: Pairing helpers and Channel types

**Files:**
- Create: `pkg/message_gateway/types.go`
- Create: `pkg/message_gateway/channel.go`
- Create: `pkg/message_gateway/registry.go`
- Create: `pkg/message_gateway/pairing.go`
- Test: `pkg/message_gateway/pairing_test.go`
- Test: `pkg/message_gateway/registry_test.go`

**Interfaces:**
- Produces: `GenerateCode() (string, error)` returns 8 chars from the alphabet; `NormalizeCode(s string) string` strips `-` and uppercases; `FormatCode(s string) string` → `XXXX-XXXX`; `Register(typ string, fn Factory)`; `Lookup(typ string) (Factory, bool)`; types `Channel`, `Capability`, `InboundMessage`, `OutboundMessage`, `Recipient`, `Attachment`, `Handler`

- [ ] **Step 1: Write pairing tests**

```go
func TestGenerateCode_AlphabetAndLength(t *testing.T) {
	code, err := GenerateCode()
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 8 {
		t.Fatalf("len=%d", len(code))
	}
	for _, r := range code {
		if !strings.ContainsRune(CodeAlphabet, r) {
			t.Fatalf("bad rune %q", r)
		}
	}
}

func TestNormalizeAndFormat(t *testing.T) {
	if got := NormalizeCode("ab-cd-ef-gh"); got != "ABCDEFGH" {
		t.Fatalf("got %q", got)
	}
	if got := FormatCode("ABCDEFGH"); got != "ABCD-EFGH" {
		t.Fatalf("got %q", got)
	}
}
```

- [ ] **Step 2: Run — must fail compile**

```bash
go test ./pkg/message_gateway -count=1
```

Expected: undefined `GenerateCode`.

- [ ] **Step 3: Implement types, pairing, registry**

`pairing.go`: `const CodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"`, `CodeLength = 8`. `GenerateCode` reads `crypto/rand` indexes.

`channel.go`:

```go
type Channel interface {
	Type() string
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Send(ctx context.Context, to Recipient, msg OutboundMessage) error
	Capabilities() Capability
}

type Handler func(ctx context.Context, msg InboundMessage) error
type Factory func(cfg ChannelConfig, onInbound Handler) (Channel, error)
```

`ChannelConfig` in types: `ID uint64`, `Type string`, `Name string`, `Credentials map[string]string`, `Extra map[string]string`.

`Attachment`: `Path, FileName, MIME, Error string`.

Registry: mutex map[string]Factory.

- [ ] **Step 4: Tests pass**

```bash
go test ./pkg/message_gateway -count=1
```

- [ ] **Step 5: Commit**

```bash
git add pkg/message_gateway
git commit -m "feat(message-gateway): add channel types, registry, and pairing codes"
```

---

### Task 2: Models and goose migrations

**Files:**
- Create: `internal/model/message_gateway.go`
- Create: `internal/infra/persistence/migrator/goose/postgres/202608160003_create_message_gateway.sql`
- Create: `internal/infra/persistence/migrator/goose/sqlite/202608160003_create_message_gateway.sql`
- Modify: `internal/testhelper/test_helper.go` AutoMigrate list
- Modify: `internal/infra/persistence/migrator/migrator_test.go` only if it asserts table counts that break (do not change `w_system_configs` count)

**Interfaces:**
- Produces: `model.MessageChannel`, `model.MessageBinding`, `model.MessagePairingCode` with `TableName()` `w_message_channels`, `w_message_bindings`, `w_message_pairing_codes`
- Constants: `MessageChannelTypeTelegram = "telegram"`, `MessageChannelTypeQQ = "qq"`, `MessageOwnerScopeSystem = "system"`

- [ ] **Step 1: Add models**

```go
type MessageChannel struct {
	ID          uint64 `json:"id,string" gorm:"primaryKey"`
	Name        string `json:"name" gorm:"size:128;not null"`
	Type        string `json:"type" gorm:"size:32;not null;index"`
	OwnerScope  string `json:"owner_scope" gorm:"size:16;not null;default:system"`
	OwnerID     *uint64 `json:"owner_id,string"`
	Enabled     bool   `json:"enabled" gorm:"not null;default:true"`
	Credentials string `json:"-" gorm:"type:text"`
	Extra       string `json:"extra" gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

Binding: `UserID`, `ChannelID`, `PlatformUserID`. Pairing: `Code` PK, `ChannelID`, `PlatformUserID`, `ExpiresAt`.

- [ ] **Step 2: Goose SQL (both dialects)**

Postgres up:

```sql
CREATE TABLE IF NOT EXISTS w_message_channels (
    id BIGINT PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    type VARCHAR(32) NOT NULL,
    owner_scope VARCHAR(16) NOT NULL DEFAULT 'system',
    owner_id BIGINT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    credentials TEXT NOT NULL DEFAULT '',
    extra TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_w_message_channels_type ON w_message_channels (type);

CREATE TABLE IF NOT EXISTS w_message_bindings (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    channel_id BIGINT NOT NULL,
    platform_user_id VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_w_message_bindings_channel_platform
    ON w_message_bindings (channel_id, platform_user_id);
CREATE INDEX IF NOT EXISTS idx_w_message_bindings_user ON w_message_bindings (user_id);

CREATE TABLE IF NOT EXISTS w_message_pairing_codes (
    code VARCHAR(16) PRIMARY KEY,
    channel_id BIGINT NOT NULL,
    platform_user_id VARCHAR(128) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_w_message_pairing_lookup
    ON w_message_pairing_codes (channel_id, platform_user_id);
```

SQLite twin: `INTEGER`/`TEXT`/`DATETIME` as in existing Wavelet sqlite migrations. Down: drop the three tables.

- [ ] **Step 3: AutoMigrate the three models in testhelper**

- [ ] **Step 4:**

```bash
go test ./internal/model ./internal/infra/persistence/migrator ./internal/testhelper -count=1
```

Expected: PASS (migrator applies `202608160003`).

- [ ] **Step 5: Commit**

```bash
git add internal/model/message_gateway.go \
  internal/infra/persistence/migrator/goose \
  internal/testhelper/test_helper.go
git commit -m "feat(message-gateway): add w_message_* models and goose migrations"
```

---

### Task 3: Repository

**Files:**
- Create: `internal/repository/message_gateway.go`
- Test: `internal/repository/message_gateway_test.go`

**Interfaces:**
- Consumes: models from Task 2, `db.DB(ctx)` from `internal/infra/persistence`
- Produces:
  - `CreateMessageChannel`, `UpdateMessageChannel`, `GetMessageChannel`, `ListMessageChannels`, `DeleteMessageChannel`
  - `CreateMessageBinding`, `GetBindingByChannelPlatform(ctx, channelID, platformUserID)`, `ListBindingsByUser`, `DeleteMessageBinding`
  - `UpsertPairingCode` (reuse unexpired row for same channel+platform user), `GetPairingCode`, `DeletePairingCode`, `DeleteExpiredPairingCodes`

- [ ] **Step 1: Test upsert reuses unexpired code**

```go
func TestUpsertPairingCode_ReusesUnexpired(t *testing.T) {
	_, _, cleanup := testhelper.SetupTestEnvironment(t)
	defer cleanup()
	ctx := context.Background()
	first, err := repository.UpsertPairingCode(ctx, 1, "tg-1", "ABCD1234", time.Now().Add(15*time.Minute))
	if err != nil { t.Fatal(err) }
	second, err := repository.UpsertPairingCode(ctx, 1, "tg-1", "ZZZZ9999", time.Now().Add(15*time.Minute))
	if err != nil { t.Fatal(err) }
	if first.Code != second.Code || first.Code != "ABCD1234" {
		t.Fatalf("reuse failed: %+v %+v", first, second)
	}
}
```

- [ ] **Step 2: Run — fail (undefined)**

```bash
go test ./internal/repository -count=1 -run Pairing
```

- [ ] **Step 3: Implement repository using `db.DB(ctx)` only**

`UpsertPairingCode`: `First` where `channel_id` and `platform_user_id` and `expires_at > now`; if found return it; else create with provided code.

`DeleteMessageChannel`: delete pairings and bindings for that channel then the channel (transaction).

- [ ] **Step 4: Tests pass**

- [ ] **Step 5: Commit**

```bash
git add internal/repository/message_gateway.go internal/repository/message_gateway_test.go
git commit -m "feat(message-gateway): add channel, binding, and pairing repositories"
```

---

### Task 4: Domain event

**Files:**
- Create: `internal/listener/message_gateway.go`
- Test: `internal/listener/message_gateway_test.go`
- Modify: `internal/platform/bootstrap/bootstrap.go` — register a log-only handler in `RegisterPushDomainEvents` sibling `RegisterMessageGatewayListeners` (`sync.Once`)

**Interfaces:**
- Produces: `const EventMessageGatewayInbound = "message_gateway.inbound"`
- `type MessageGatewayInbound struct { Msg message_gateway.InboundMessage }`
- `OnMessageGatewayInbound(fn)`
- `EmitMessageGatewayInbound(ctx, msg)` — skip if `msg.BindingUserID == nil`

- [ ] **Step 1: Write emit test**

```go
func TestEmitMessageGatewayInbound_SkipsUnbound(t *testing.T) {
	called := 0
	OnMessageGatewayInbound(func(ctx context.Context, ev MessageGatewayInbound) { called++ })
	EmitMessageGatewayInbound(context.Background(), message_gateway.InboundMessage{Text: "x"})
	if called != 0 {
		t.Fatal("unbound must not emit")
	}
	uid := uint64(9)
	EmitMessageGatewayInbound(context.Background(), message_gateway.InboundMessage{BindingUserID: &uid, Text: "x"})
	if called != 1 {
		t.Fatalf("called=%d", called)
	}
}
```

- [ ] **Step 2: Run — fail compile**
- [ ] **Step 3: Implement listener + bootstrap log-only handler**
- [ ] **Step 4: Test pass**
- [ ] **Step 5: Commit**

```bash
git commit -m "feat(message-gateway): emit message_gateway.inbound domain events"
```

---

### Task 5: Telegram adapter (private chat)

**Files:**
- Create: `pkg/message_gateway/channel/telegram/adapter.go`
- Test: `pkg/message_gateway/channel/telegram/adapter_test.go`

**Interfaces:**
- Consumes: `ChannelConfig`, `Handler`
- Produces: `func New(cfg ChannelConfig, onInbound Handler) (message_gateway.Channel, error)`
- Credentials key `bot_token`; extra key `base_url`
- `Type() == "telegram"`
- `Capabilities`: Text, Image, File, Reply true; Group false

- [ ] **Step 1: Test group updates are dropped (inject a `handleUpdate` method)**

Export `HandlePrivate` for tests:

```go
func TestHandleUpdate_DropsGroups(t *testing.T) {
	var got int
	a := &Adapter{onInbound: func(ctx context.Context, msg message_gateway.InboundMessage) error {
		got++
		return nil
	}}
	a.handleTeleMessage(fakeGroupMessage())
	if got != 0 {
		t.Fatalf("group must be ignored")
	}
}

func TestHandleUpdate_PrivateText(t *testing.T) {
	var got message_gateway.InboundMessage
	a := &Adapter{cfg: message_gateway.ChannelConfig{ID: 7, Type: "telegram"}, onInbound: func(ctx context.Context, msg message_gateway.InboundMessage) error {
		got = msg
		return nil
	}}
	a.handleTeleMessage(fakePrivateText("hi", 42))
	if got.Text != "hi" || got.PlatformUserID != "42" || got.ChannelID != 7 {
		t.Fatalf("%+v", got)
	}
}
```

Build fakes as small structs / telebot.Message with `Chat.Type = telebot.ChatPrivate` vs `ChatGroup`.

- [ ] **Step 2: Run — fail**

- [ ] **Step 3: Implement with `gopkg.in/telebot.v4`**

`Connect`: `telebot.NewBot(Settings{Token, URL: extra base_url, Poller: &telebot.LongPoller{Timeout: 10s}})`, handle `OnText`, `OnPhoto`, `OnDocument` only if private. Download media to `os.MkdirTemp("wg-tg-*")`. `Disconnect`: `bot.Stop()`.

Register factory in `New` via `message_gateway.Register("telegram", New)` from **apps runner**, not `init()` in pkg if that violates bootstrap rule. Spec said no `init()` for **cross-module** integration. Registering a factory in `telegram` package `init` is OK **or** register explicitly in runner. Prefer **explicit register in runner** to keep pkg side-effect free.

- [ ] **Step 4: `go test ./pkg/message_gateway/channel/telegram -count=1`**

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(message-gateway): add Telegram private-chat telebot adapter"
```

Add `gopkg.in/telebot.v4` with `go get`.

---

### Task 6: QQ adapter (C2C only)

**Files:**
- Create: `pkg/message_gateway/channel/qq/adapter.go`
- Test: `pkg/message_gateway/channel/qq/adapter_test.go`

**Interfaces:**
- Same `New(cfg, onInbound)`
- Credentials: `app_id`, `app_secret`
- Extra: `portal_host` default `q.qq.com`
- Drop non-C2C events in a testable `handleC2C` function

- [ ] **Step 1: Write C2C-only tests**

```go
func TestHandleEvent_DropsNonC2C(t *testing.T) {
	var got int
	a := &Adapter{onInbound: func(ctx context.Context, msg message_gateway.InboundMessage) error {
		got++
		return nil
	}}
	a.handleEvent(qqEvent{Kind: "group", UserID: "u1", Text: "hi"})
	if got != 0 {
		t.Fatal("non-C2C must be ignored")
	}
}

func TestHandleEvent_C2CText(t *testing.T) {
	var got message_gateway.InboundMessage
	a := &Adapter{cfg: message_gateway.ChannelConfig{ID: 3}, onInbound: func(ctx context.Context, msg message_gateway.InboundMessage) error {
		got = msg
		return nil
	}}
	a.handleEvent(qqEvent{Kind: "c2c", UserID: "openid-1", Text: "hello", MessageID: "m1"})
	if got.Text != "hello" || got.PlatformUserID != "openid-1" || got.ChannelID != 3 {
		t.Fatalf("%+v", got)
	}
}
```

Define a local `qqEvent` in the adapter file so tests do not need a live gateway. `Connect` uses official `github.com/tencent-connect/botgo` (or the module path from https://bot.q.qq.com/wiki/develop/gosdk/); pin the resolved module in `go.mod` and name it in the commit body. `Connect` enables C2C intent only. `Send` uses C2C REST.

- [ ] **Step 2: `go test ./pkg/message_gateway/channel/qq -count=1` — fail compile**
- [ ] **Step 3: Implement adapter + `New`**
- [ ] **Step 4: Tests pass**
- [ ] **Step 5: Commit**

```bash
git commit -m "feat(message-gateway): add QQ official C2C botgo adapter"
```

---

### Task 7: Worker gateway runner

**Files:**
- Create: `internal/apps/message_gateway/runner/runner.go`
- Create: `internal/apps/message_gateway/runner/inbound.go`
- Test: `internal/apps/message_gateway/runner/inbound_test.go`
- Modify: `internal/cmd/worker.go` start `runner.Start` in a goroutine **before** `worker.StartWorker()`
- Modify: `internal/platform/bootstrap` if needed to register factories

**Interfaces:**
- Consumes: repository, listener, telegram.New, qq.New, pairing.GenerateCode
- Produces: `Start(ctx context.Context) error` — load enabled channels, lock, Connect; poll `updated_at` every 5s and reload changed IDs
- Inbound handler:
  1. Lookup binding by channel+platform user
  2. If none: `UpsertPairingCode` + `ch.Send` formatted code + bind instructions
  3. If bound: set `BindingUserID`, `EmitMessageGatewayInbound`; on emit error send “could not save”; on success send “received”

- [ ] **Step 1: Test inbound unbound mints/reuses code and does not emit**

Use a `fakeChannel` implementing `Channel` that records `Send` calls, and stub repository via a small `inboundService` struct with function fields so the test does not need a live bot.

```go
type inboundDeps struct {
	LookupBinding func(...) (*model.MessageBinding, error)
	UpsertCode    func(...) (*model.MessagePairingCode, error)
	Emit          func(context.Context, message_gateway.InboundMessage)
	Send          func(context.Context, Recipient, OutboundMessage) error
}

func (d inboundDeps) Handle(ctx context.Context, msg message_gateway.InboundMessage) error
```

- [ ] **Step 2–4: Implement Handle + Start**

Credential decrypt: `key := hex.EncodeToString(sum[:])` where `sum := sha256.Sum256([]byte(config.Config.App.SessionSecret))`. If decrypt fails, skip channel and log.

Lock: Redis `SET wg:channel:{id} {node} NX EX 30` refreshed while running; if Redis disabled, start anyway (single-process dev).

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(message-gateway): run adapters on worker and handle pairing inbound"
```

---

### Task 8: Admin HTTP API

**Files:**
- Create: `internal/apps/admin/message_gateway/errs.go`
- Create: `internal/apps/admin/message_gateway/logics.go`
- Create: `internal/apps/admin/message_gateway/handlers.go`
- Create: `internal/apps/admin/message_gateway/routers.go`
- Test: `internal/apps/admin/message_gateway/logics_test.go`
- Modify: `internal/router/v1/admin.go` — `registerAdminMessageGatewayRoutes` under “Messaging”
- Create: `internal/router/v1/message_gateway_admin.go` calling `adminmsg.RegisterRoutes`

**Interfaces:**
- `GET /api/v1/admin/message-gateway/channels`
- `GET /api/v1/admin/message-gateway/channels/definitions`
- `POST /api/v1/admin/message-gateway/channels` body `{name,type,enabled,bot_token|app_id,app_secret,base_url,portal_host}`
- `PATCH /api/v1/admin/message-gateway/channels/:id` empty secrets keep previous
- `DELETE /api/v1/admin/message-gateway/channels/:id`
- `POST /api/v1/admin/message-gateway/channels/:id/test` optional getMe/token probe
- List DTO masks secrets as `********` if non-empty

Definitions JSON:

```go
[]Definition{
  {Type:"telegram", Name:"Telegram", Fields:[]Field{{Key:"bot_token", Type:"password", Required:true}, {Key:"base_url", Type:"text"}}},
  {Type:"qq", Name:"QQ", Fields:[]Field{{Key:"app_id", Required:true}, {Key:"app_secret", Type:"password", Required:true}, {Key:"portal_host", Type:"text"}}},
}
```

- [ ] **Step 1: Test create telegram without token fails; create with token stores ciphertext not plaintext**

- [ ] **Step 2–4: Implement logics + Abort\* handlers + swagger comments**

- [ ] **Step 5:** `make swagger` then commit

```bash
git commit -m "feat(message-gateway): add admin channel CRUD APIs"
```

---

### Task 9: User bind/unbind API

**Files:**
- Create: `internal/apps/message_gateway/errs.go`
- Create: `internal/apps/message_gateway/logics.go`
- Create: `internal/apps/message_gateway/handlers.go`
- Create: `internal/apps/message_gateway/routers.go`
- Test: `internal/apps/message_gateway/logics_test.go`
- Modify: `internal/router/v1/v1.go` call `RegisterMessageGatewayUserRoutes`
- Create: `internal/router/v1/message_gateway.go`

**Interfaces:**
- `GET /api/v1/message-gateway/bindings` — current user, include channel name/type
- `POST /api/v1/message-gateway/bindings` `{ "channel_id": "...", "code": "ABCD-EFGH" }`
- `DELETE /api/v1/message-gateway/bindings/:id`
- Login required. Bind: `NormalizeCode`, `GetPairingCode`, check expiry and `channel_id`, `GetBindingByChannelPlatform` → 409 if other user, insert binding, delete code.
- Unbind: only if `binding.UserID == current`.

- [ ] **Step 1–4: TDD expired code 400; happy path deletes code**

- [ ] **Step 5: `make swagger` + commit**

```bash
git commit -m "feat(message-gateway): add user bind and unbind APIs"
```

---

### Task 10: Frontend service + admin page

**Files:**
- Create: `frontend/lib/services/message-gateway/types.ts`
- Create: `frontend/lib/services/message-gateway/admin.service.ts`
- Create: `frontend/lib/services/message-gateway/index.ts`
- Modify: `frontend/lib/services/index.ts` export
- Create: `frontend/app/(main)/admin/message-gateway/page.tsx`
- Create: `frontend/app/(main)/admin/message-gateway/components/channel-card.tsx`
- Create: `frontend/app/(main)/admin/message-gateway/components/add-channel-dialog.tsx`
- Create: `frontend/app/(main)/admin/message-gateway/channels/telegram/form.tsx`
- Create: `frontend/app/(main)/admin/message-gateway/channels/qq/form.tsx`
- Modify: `frontend/components/layout/sidebar.tsx` admin item after push
- Modify: `frontend/messages/zh-CN.json` + `en.json` namespace `admin.messageGateway`

**Interfaces:**
- `AdminMessageGatewayService.list/create/update/remove/definitions`
- Empty state + add dialog: type select then `<TelegramForm>` or `<QQForm>`
- Card: name, type, enabled switch, delete
- `useTranslations('admin.messageGateway')` — no hardcoded UI strings

- [ ] **Step 1: Add i18n keys to zh-CN only, run `node frontend/scripts/check-i18n-keys.mjs` — fail**
- [ ] **Step 2: Add matching en keys — checker pass**
- [ ] **Step 3: Implement pages (follow `/admin/push` layout: `w-full py-6`, h1 `text-2xl font-semibold tracking-tight`)**
- [ ] **Step 4:** `cd frontend && pnpm tsc --noEmit --jsx preserve`
- [ ] **Step 5: Commit**

```bash
git commit -m "feat(message-gateway): add admin channel cards and per-type forms"
```

---

### Task 11: Profile bind card

**Files:**
- Create: `frontend/lib/services/message-gateway/user.service.ts`
- Create: `frontend/components/common/settings/bot-binding-card.tsx`
- Modify: `frontend/components/common/settings/profile.tsx` render the card
- Modify: i18n `settings.botBinding`

**Interfaces:**
- List bindings (channel name, type, platform_user_id) + unbind
- Bind dialog: select enabled channel + code input
- All copy via next-intl

- [ ] **Step 1–4: keys + component + typecheck**
- [ ] **Step 5: Commit**

```bash
git commit -m "feat(message-gateway): add profile bot pairing card"
```

---

### Task 12: Repo-wide gates

**Files:** swagger `docs/*` if not already committed

- [ ] **Step 1:** `go test ./... -count=1`
- [ ] **Step 2:** `make swagger && make format && node frontend/scripts/check-i18n-keys.mjs && make code-check`
- [ ] **Step 3:** Confirm `RegisterMessageGatewayUserRoutes` in `v1.go` and admin routes exist; no `init()` in apps
- [ ] **Step 4: Commit leftover format/swagger**

```bash
git commit -m "chore(message-gateway): swagger and format"
```

---

## Spec coverage

| Spec | Task |
|---|---|
| pkg types / pairing / registry | 1 |
| tables | 2 |
| repository | 3 |
| domain event | 4 |
| telegram / qq adapters | 5–6 |
| worker runner + pairing inbound | 7 |
| admin API + UI | 8, 10 |
| user bind + profile | 9, 11 |
| gates | 12 |
| Clipper ingest | **not in this plan** |
