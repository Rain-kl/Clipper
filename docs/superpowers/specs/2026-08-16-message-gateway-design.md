# Message Gateway Design (Wavelet)

**Date:** 2026-08-16  
**Status:** Approved for implementation planning  
**Scope:** Wavelet framework only (sub-project 1 of 3)  
**Product:** Wavelet scaffold — reusable inbound/outbound messaging channels

## 1. Goal

Give Wavelet a Hermes-style **message gateway**: admins configure Telegram and QQ bots in the admin UI; end users bind their private-chat identity with a one-time short code on the profile page. Inbound private messages become **domain events**. Products (Clipper later) subscribe and decide what to persist.

This spec does **not** write Clipper `c_items`. That is sub-project 3 after Wavelet is merged into Clipper.

### In scope (Wavelet)

- `pkg/message_gateway` + `channel/telegram` + `channel/qq`
- Admin UI `/admin/message-gateway` (empty state, add channel, per-type forms, cards)
- User profile card: bind / list / unbind
- Worker-hosted connections (Telegram long poll, QQ official WebSocket)
- Pairing codes, bindings, encrypted credentials
- Domain event on authorized inbound messages
- Text + image + file inbound/outbound at the adapter layer (v1 private chat only)

### Out of scope

- Group / guild / channel / @mention routing
- Hermes sessions, slash commands, STT, cron, circuit-breaker fleet, 20 platforms
- Telegram webhook mode (long poll only)
- Writing business tables (`c_*`) or calling `upload.Ingest` from `pkg/`
- Changing `pkg/push` (outbound notification remains a separate system)
- Multi-tenant hosted bots (`owner_scope=user`) — column exists, v1 only inserts `system`

### Later sub-projects (not this spec)

2. Clipper `git fetch upstream && git merge upstream/main`
3. Clipper listener: inbound event → `upload.Ingest` + `c_items` with `source=telegram|qq`

## 2. Decisions (locked)

| Topic | Choice |
|---|---|
| Ownership | Instance-level shared bots; users pair to them |
| Pairing | 8-char one-time code, 15 minutes, consumed on success |
| Profile | Show bound platform user ID; user can unbind |
| Surface | Private chat / C2C only; groups ignored + logged |
| Runtime | Worker process starts the gateway; API does not poll |
| Events | Bound inbound → `internal/listener` domain event |
| Telegram SDK | `gopkg.in/telebot.v4` (tucnak/telebot current module) |
| QQ SDK | Official QQ Bot Go SDK (`botgo` / [docs](https://bot.q.qq.com/wiki/develop/gosdk/)) |
| Hermes | Adapter shape only (`Connect`/`Disconnect`/`Send`/inbound event/capabilities) |

## 3. Architecture

```
Admin UI ──CRUD──► API ──w_message_channels──► Worker Gateway
                                                      │
User DM ──telebot / botgo──► Channel adapter ─────────┤
                                                      ▼
                                         unbound? mint pairing code, reply
                                         bound?   emit InboundMessage event
                                                      │
User profile ──bind/unbind──► API ──w_message_bindings / pairing_codes
                                                      │
                                         product listener (Clipper later)
```

`pkg/message_gateway` has **no** Gin, GORM, sessions, or `internal/apps` imports. Persistence lives in `internal/repository`. Encryption of credentials uses existing Wavelet secret helpers from apps/repository, not from `pkg`.

### 3.1 Packages

```
pkg/message_gateway/
  types.go          # Message, Attachment, Capability, ChannelType
  channel.go        # Channel interface
  registry.go       # Register / Lookup
  pairing.go        # GenerateCode (alphabet, length) — no storage
  channel/telegram/ # telebot private chat
  channel/qq/       # official botgo C2C
internal/apps/admin/message_gateway/
internal/apps/message_gateway/   # user bind/unbind
internal/listener/               # InboundMessage event type + dispatch
internal/model/                  # channel, binding, pairing row types
internal/repository/             # CRUD
```

Frontend:

```
frontend/lib/services/message-gateway/
frontend/app/(main)/admin/message-gateway/
  page.tsx
  components/channel-card.tsx
  components/add-channel-dialog.tsx
  channels/telegram/form.tsx
  channels/qq/form.tsx
frontend/components/common/settings/   # profile bind card
```

Sidebar admin item: `titleKey: 'messageGateway'`, url `/admin/message-gateway`, placed next to push.

### 3.2 Channel interface

```go
type Channel interface {
    Type() string
    Connect(ctx context.Context) error
    Disconnect(ctx context.Context) error
    Send(ctx context.Context, to Recipient, msg OutboundMessage) error
    Capabilities() Capability
}

type Capability struct {
    Text, Image, File, Reply bool
    Group bool // always false in v1 adapters
}

type InboundMessage struct {
    ChannelID      uint64
    ChannelType    string
    PlatformUserID string
    ChatID         string
    MessageID      string
    Text           string
    Attachments    []Attachment // local temp paths + mime; no upload package
    BindingUserID  *uint64      // nil if unbound
}

type Handler func(ctx context.Context, msg InboundMessage) error
```

Gateway runner (Worker) constructs adapters from DB rows, calls `Connect`, and registers one `Handler` that implements pairing + event emit.

### 3.3 Process model

- `internal/cmd` Worker path: after bootstrap, `messagegateway.Start(ctx)`.
- API process never starts adapters.
- Enable / disable / credential change: Worker watches DB (poll every few seconds or Redis pub/sub already used by the platform). v1 may poll `updated_at` every 5s. Hot-reload only the changed channel.
- Single Worker assumed. If a second Worker starts, it tries a Redis/DB lock keyed by `channel_id` + token fingerprint; failure → skip that channel and log.
- Per-channel reconnect with exponential backoff; one channel crash must not stop others.

## 4. Data model

All tables `w_*`. Dual goose SQL (Postgres + SQLite). No physical FKs.

### 4.1 `w_message_channels`

| Column | Type | Notes |
|---|---|---|
| id | snowflake | PK |
| name | string | admin display name |
| type | string | `telegram` \| `qq` |
| owner_scope | string | v1 always `system` |
| owner_id | bigint null | reserved; null in v1 |
| enabled | bool | |
| credentials | text | encrypted JSON; never returned raw |
| extra | text | optional JSON (base URL, sandbox host) |
| created_at, updated_at | timestamptz | |

### 4.2 `w_message_bindings`

| Column | Type | Notes |
|---|---|---|
| id | snowflake | PK |
| user_id | bigint | Wavelet user |
| channel_id | bigint | |
| platform_user_id | string | Telegram user id / QQ OpenID |
| created_at | timestamptz | |

Unique `(channel_id, platform_user_id)`. A user may bind many channels; a platform identity binds to at most one user per channel.

### 4.3 `w_message_pairing_codes`

| Column | Type | Notes |
|---|---|---|
| code | string | PK, 8 chars |
| channel_id | bigint | |
| platform_user_id | string | |
| expires_at | timestamptz | now + 15m |
| created_at | timestamptz | |

Delete row on successful bind. Expired rows ignored and cleaned by a small periodic delete (may live in the Worker loop).

**Code alphabet:** `ABCDEFGHJKLMNPQRSTUVWXYZ23456789` (no `0O1I`). Format displayed as `XXXX-XXXX`.

## 5. Pairing and profile

1. Unbound private message arrives.
2. Worker upserts a pairing row (if an unexpired code already exists for that pair, reuse it).
3. Bot replies with the code and “Settings → Profile → Bind a bot”.
4. User opens profile card, picks an **enabled** channel, submits the code.
5. API validates, inserts binding, deletes the code.
6. Profile lists bindings: channel name, type, **platform user ID**, unbind button.

Unauthorized/unbound never emit `InboundMessage` to product listeners.

## 6. Admin UI and credentials

Empty state + “Add channel”. Dialog: choose `telegram` | `qq`, then that type’s form.

| Type | Required | Optional |
|---|---|---|
| telegram | name, bot token | API base URL (default `https://api.telegram.org`) |
| qq | name, App ID, App Secret | sandbox / portal host (default `q.qq.com`) |

Cards: name, type, enabled toggle, connected/disconnected (best-effort from Worker), edit, delete. Edit never echoes raw secrets; empty secret field means “keep current”.

Create: validate field shape, optional SDK probe (`getMe` / token), then insert. Probe failure → 400, no row.

## 7. HTTP API

Admin (admin middleware), prefix `/api/v1/admin/message-gateway`:

- `GET /channels` — list (secrets masked)
- `GET /channels/definitions` — form schema per type
- `POST /channels` — create
- `PATCH /channels/:id` — update
- `DELETE /channels/:id` — delete (cascade bindings + pairing rows in logics)
- `POST /channels/:id/test` — optional probe

User (login required), prefix `/api/v1/message-gateway`:

- `GET /bindings` — current user’s bindings
- `POST /bindings` — `{ channel_id, code }`
- `DELETE /bindings/:id` — unbind own row only

Errors via `response.Abort*`. Invalid/expired code → 400. Platform identity already bound → 409.

## 8. Inbound media and reply

Adapters accept private-chat **text, images, files**. Voice/STT is out of scope; a voice message is treated as a file attachment if the SDK delivers bytes, otherwise ignored with a log.

Attachments are written under `t.TempDir()`-equivalent process temp (`os.MkdirTemp`) and passed as filesystem paths on `InboundMessage`. The product listener (Clipper later) must `upload.Ingest` and then delete the temp file. Gateway deletes leftovers older than 1 hour.

If the product handler returns an error, the bot sends a generic “could not save your message” (no internal error text). Success ACK is a single short reply when `Capabilities().Reply` is true. ACK can be disabled later via extra JSON; v1 always ACKs.

Group / non-C2C updates: log and drop. Do not mint pairing codes.

## 9. Domain event

Name: `message_gateway.inbound` (exact string in `internal/listener`).

Payload: `InboundMessage` plus `BindingUserID` set. Register in `internal/platform/bootstrap` (no `init()`). Wavelet ships a no-op or log-only listener. Clipper will register the clip writer in sub-project 3.

## 10. Error handling

- Adapter panics: recover in the gateway runner, mark channel disconnected, backoff reconnect.
- Media download failure: keep text; attachment entry has `Error` string; still emit if bound.
- Encrypt/decrypt failure: treat channel as disabled, log, do not start adapter.
- i18n: all new UI strings in `zh-CN.json` + `en.json` (`admin.messageGateway`, `settings.botBinding`).

## 11. Testing and done criteria

| Gate | Pass |
|---|---|
| Fake channel + pairing | generate, reuse unexpired, expire, consume, conflict 409 |
| Telegram adapter unit | mock telebot updates: text, photo, document; groups dropped |
| QQ adapter unit | mock C2C event; non-C2C dropped |
| API tests | admin CRUD, bind/unbind, bad code |
| `go test ./...` | pass |
| Frontend | `pnpm tsc --noEmit --jsx preserve`, `check-i18n-keys.mjs` |
| Manual | add TG channel → DM gets code → profile bind → second DM emits event in logs |

## 12. Risks

- **Official QQ Bot** requires a published/sandbox app at q.qq.com; local dev may only test Telegram.
- **telebot v4 module path** is `gopkg.in/telebot.v4`; pin a version in `go.mod`.
- **One poller per token:** documented; second Worker skips the channel.
- **Token leak in logs:** never log raw credentials or pairing codes at info level (debug only, redacted).
