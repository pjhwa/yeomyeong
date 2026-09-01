<div align="center">

# 여명 · YEOMYEONG

### *나라를 되찾아라. 칼을 뽑지 않아도 된다.*

현대적 텍스트 MMO. 성장은 **실제로 한 행동**에서 온다 —
몹을 갈아 숫자를 올리는 게임이 아니다. 레벨 없음. 클래스 없음.
살아있는 경제. 당신을 **기억하는** LLM NPC.

Go로 구축. [DikuMUD](https://dikumud.com) (1990)에서 영감을 받았고, 코드는 한 줄도 공유하지 않는다.

[![Build](https://img.shields.io/github/actions/workflow/status/pjhwa/yeomyeong/ci.yml?branch=main)](https://github.com/pjhwa/yeomyeong/actions)
[![Go](https://img.shields.io/badge/go-1.24%2B-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![Roadmap](https://img.shields.io/badge/milestone-M2-success)](#로드맵)

**[English README](README.md)** · **[데모 플레이](#)** · **[Discord](#)**

</div>

---

> **⚠️ 상태: M2 플레이 가능 — 첫 10분 훅이 들어가 있다.** 경제(M3)와 1부 서사(M4)는
> 아직 설계 목표다. 현재 범위는 [로드맵](#로드맵)을 본다.

<!-- TODO(M1): replace with asciinema demo GIF — 30s: market → haggle → checkpoint → deliver -->
<div align="center">
  <img src="docs/assets/demo.gif" alt="터미널 데모" width="720">
  <br><em>한벽성에서의 30초 — 장터에서 흥정하고, 검문을 빠져나간다.</em>
</div>

## 왜 또 MUD인가?

텍스트 MUD가 죽은 이유는 분명하다. 글을 입힌 스프레드시트가 되었기 때문이다.
쥐를 잡아 숫자가 올랐다. 이 형식은 더 잘해도 된다.

- **레벨 없음, 클래스 없음.** 40여 종 숙련은 *사용*으로 자란다. 망치를 휘두르면
  대장 숙련이 오른다. 흥정하면 흥정이 는다. 총량 상한이 있어서 한 사람이 전부를
  가질 수 없고, 그래서 플레이어가 서로를 필요로 한다.
- **싸우지 않고도 서사를 끝낼 수 있다.** 모든 주요 장애물은 최소 세 해법을 갖는다.
  무력, 기술, 말. 지하 인쇄소에 잉크를 밀수하는 일이 습격과 똑같이 본편을 진행시킨다.
- **세계가 반응한다.** 시세는 수급·날씨·전쟁 소식에 따라 움직인다. NPC에게는 일과가 있다.
  플레이어가 대담해지면 서버 전체의 *탄압 수위*가 오르고 순찰이 늘어난다.
- **기억하는 NPC.** 3계층 대화(스크립트 → 오픈웨이트 모델 → 프리미엄 API)로
  주요 인물은 지난 대화를 기억한다. 실제 게임 상태를 반영하고, 서사 진행 전에는
  비밀을 말하지 않는다.

## 배경

1920년대 풍 대체역사. 반도 국가 **달내 (Dalnae)** 는 15년 전 **무쇠 제국**에
나라를 빼앗겼다. 제국은 지맥의 혈점에 쇠말뚝을 박아 땅의 기운과 사람들
마음까지 수도로 뽑아 간다.

당신은 변방 마을의 아무개로 시작한다. 사라진 전달자, 틀린 활자가 찍힌
신문, 아는 것이 많아 보이는 은퇴한 선생. 비밀결사 **새벽회** 는 눈치 있는
사람을 찾고 있다.

> 장터 어귀 게시판에 관아 안내문이 겹쳐 붙었고, 맨 앞장 모퉁이만 손가락
> 넓이로 찢어져 있다. 그 아래 한벽일보에는 같은 활자가 한 줄에서 두 번
> 어긋나 있다. 좌판 저울추가 나무판을 치는 소리가 일정하고, 참기름
> 고소한 냄새가 난다.

*작중 국가·단체·인물은 전부 허구다. 이 프로젝트는 항쟁의 정신을 계승하되
실존 인물과 단체를 소재로 소비하지 않는다.*

## 60초 만에 접속

```bash
git clone https://github.com/pjhwa/yeomyeong.git
cd yeomyeong
go run ./cmd/server          # DATABASE_URL 비움 = 메모리 저장
# 또는: docker compose up --build
```

그다음:

```bash
open http://localhost:8080   # 브라우저 터미널
nc localhost 4001            # 한글 입력 (권장)
telnet localhost 4001        # 고전 — macOS telnet 은 한글 IME 가 깨짐
```

Postgres는 선택이다. `docker compose up` 은 로컬 Postgres도 띄우지만, 데모는
`DATABASE_URL` 을 비워 메모리 저장을 쓴다.

**첫 10분 (전투 없음):** 달빛골 마을문에서 북쪽으로 장터까지 걷고 (`n` 몇 번,
그다음 `e`), `보다 신문`. 남쪽으로 서당까지 가서 `대화 청람` 을 두 번 —
선생이 자네를 기억한다. 캡처는
[`docs/assets/first-10-minutes.txt`](docs/assets/first-10-minutes.txt).

## 아키텍처

동시성 모델이 다른 모든 결정의 못이다.
**월드 상태를 소유하는 goroutine은 단 하나다.**

```
[client] ──WebSocket/Telnet──> [connection goroutine, one per player]
                                        │ commands (channel)
                                        ▼
                          ┌──────────────────────────────┐
                          │   GAME LOOP  (exactly one)   │
                          │   100ms tick                 │
                          │   • sole writer of world     │
                          │   • combat rounds, NPC AI,   │
                          │     prices, weather, heat    │
                          └──────────────────────────────┘
                                        │ events (channel)
                                        ▼
                        [per-connection output buffers → clients]
```

월드 상태에 뮤텍스 없음. 데이터 레이스는 구조적으로 성립하지 않는다.
I/O는 시뮬을 막지 않는다. DB 기록과 LLM 호출은 비동기 워커에 위임한다.

| 계층 | 선택 |
|---|---|
| 서버 | Go 1.24+ |
| 전송 | WebSocket (우선) + Telnet (호환) |
| 클라이언트 | TypeScript + React |
| 저장 | PostgreSQL + 인메모리 월드, 주기 스냅샷 |
| 콘텐츠 | YAML, 핫 리로드 — **방 하드코딩 0** |
| 스크립트 | Lua (샌드박스) |
| 대화 | 교체 가능한 `DialogueProvider` — 로컬 오픈웨이트 + 프리미엄 API |

## 로드맵

- [x] **M0** — 서버 골격, WS/텔넷, 계정
- [x] **M1** — YAML 월드 로더, 이동, 시작 마을 (방 40)
- [x] **M2** — 숙련: 성장 곡선, 총량 상한, 장비
- [ ] **M3** — 살아있는 경제: 채집 → 제작 → 지역 교역

마일스톤이 끝나면 발주자는 [`docs/COMMISSIONER.md`](docs/COMMISSIONER.md) 대로
직접 접속해 보고, 그 문서의 템플릿으로 피드백을 준다.
- [ ] **M4** — 1부 서사, 복선 원장, 첫 AI NPC
- [ ] **M5** — 지부(길드), 평판, 밀무역, 주간 게임 내 신문
- [ ] **M6** — 웹 클라이언트, 밸런스 패스, 동접 500

## 기여

여기선 작가도 엔지니어만큼 환영받는다 — 잘 쓰인 방 하나가 기능이다.

[`good first issue`](../../issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) 로
시작하고, [CONTRIBUTING.md](CONTRIBUTING.md) 와 `docs/CONTENT-STYLE.md` 를
읽은 뒤에 방 글을 쓴다.

## 라이선스와 크레딧

MIT — [LICENSE](LICENSE).

**DikuMUD** (University of Copenhagen, 1990)에서 영감을 받았다. 텍스트만으로
세계가 성립한다는 것을 증명한 작품이다. 이 프로젝트는 DikuMUD 코드를 사용·복사·
파생하지 않는다. 아이디어의 클린룸 재해석이다.
