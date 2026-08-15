# WIRE-PROTOCOL

ENGINE ↔ CLIENT. Agreed before implementation (PLAN.md §9.4).  
M0 scope only. Additive changes bump `v` or add a type; do not reuse a type
with a new shape.

Related: [EVENT-BUS.md](EVENT-BUS.md).

## Transports

| Transport | Address env | Default | Framing |
|---|---|---|---|
| Telnet (compat) | `YEOMYEONG_TELNET_ADDR` | `:4000` | CRLF or LF lines. IAC bytes (`0xFF`) are dropped, not negotiated. |
| WebSocket (primary) | `YEOMYEONG_WS_ADDR` | `:8080` | JSON text frames on `GET /ws` |

Both transports enqueue the **same** command types on the game loop.
There is no second path that mutates the roster.

## Versioning

Every WebSocket frame carries `"v": 1`. Unknown `v` → `sys` error, connection
stays open. Unknown `type` → `sys` error, connection stays open.

## WebSocket — client → server

All client frames:

```json
{"v":1,"type":"<type>","id":"<client-corr>","payload":{}}
```

`id` is echoed on the matching reply when there is one. Max payload size 4 KiB.

| type | payload | When |
|---|---|---|
| `auth.create` | `{ "username", "password" }` | New account, then enter world |
| `auth.login` | `{ "username", "password" }` | Existing account, then enter world |
| `cmd.say` | `{ "text" }` | After `auth.ok` |
| `cmd.quit` | `{}` | Any time after connect |

Username rules (server-enforced): 2–16 runes; Hangul syllables (`가`–`힣`),
ASCII letters, digits, `_`; compared case-insensitively (Unicode simple fold).
Password: 8–72 bytes. Empty `text` on `cmd.say` is rejected.

## WebSocket — server → client

```json
{"v":1,"type":"<type>","id":"<echoed or empty>","payload":{}}
```

| type | payload | Meaning |
|---|---|---|
| `auth.ok` | `{ "username", "session" }` | Entered the world. `session` is an opaque token (M0: unused for reconnect). |
| `auth.err` | `{ "code", "message" }` | Create/login failed. Still at the auth gate. |
| `text` | `{ "channel", "from", "text" }` | Player-visible line (`channel` is `say` or `sys`) |
| `sys` | `{ "code", "message" }` | Protocol/rate-limit/parse error |

`auth.err` / `sys` codes:

| code | When |
|---|---|
| `name_taken` | create, username exists |
| `bad_credentials` | login failed (do not distinguish unknown user vs bad password) |
| `bad_username` | username fails the rune rules |
| `bad_password` | password too short/long |
| `not_authenticated` | `cmd.say` before `auth.ok` |
| `bad_frame` | JSON/schema/`v` |
| `rate_limited` | more than 20 commands in any rolling 1s |
| `internal` | unexpected |

## Telnet — line protocol

Server speaks Korean. Bytes are UTF-8.

```
여명 · YEOMYEONG
계정 이름:
암호:
```

If the name is unknown:

```
그 이름은 장부에 없습니다. 새로 만드시겠습니까? (y/n)
```

`y` → prompt for the password again as the new password, create, enter world.  
`n` → restart at `계정 이름:`.

If the name exists, the typed password is verified. Failure:

```
이름이나 암호가 맞지 않습니다.
계정 이름:
```

On success:

```
<username> 님이 자리에 앉았습니다.
>
```

After the prompt:

| input | command |
|---|---|
| `say <text>` / `말 <text>` | `Say` |
| `quit` / `종료` | `Quit` (close after a farewell line) |
| empty line | ignore |
| anything else | `sys` equivalent: `모르는 말입니다. say / quit` |

Broadcast format (every logged-in connection, including the speaker):

```
[말] <username>: <text>
```

System lines have no `[말]` prefix.

Telnet does not emit JSON. The connection adapter translates lines into the
same `Command` values as WebSocket.

## Rate limit

20 commands per connection per rolling second (PLAN.md §4.5), counted
**before** enqueue. Excess is dropped and the client is told `rate_limited`.
The game loop never sees the dropped command.

## What this document is not

- Not a room protocol. M0 has no rooms.
- Not a skill/combat/inventory schema.
- Not an excuse to put rooms, NPCs, or items in Go source.
