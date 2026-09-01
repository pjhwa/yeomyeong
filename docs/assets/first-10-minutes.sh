#!/usr/bin/env bash
# Reproducible first-10-minutes hook: spawn → market newspaper → school 청람 twice
# → warehouse pack examine.
# Assumes the server is listening, or starts `go run ./cmd/server`.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
HOST="${YM_HOST:-127.0.0.1}"
PORT="${YM_PORT:-4001}"
cd "$ROOT"

started=0
if ! (echo >/dev/tcp/"$HOST"/"$PORT") >/dev/null 2>&1; then
  go run ./cmd/server >/tmp/yeomyeong-first10.log 2>&1 &
  started=$!
  trap 'kill "$started" >/dev/null 2>&1 || true' EXIT
  for _ in $(seq 1 50); do
    if (echo >/dev/tcp/"$HOST"/"$PORT") >/dev/null 2>&1; then
      break
    fi
    sleep 0.1
  done
fi

python3 - "$HOST" "$PORT" "${YM_USER:-시연$RANDOM}" <<'PY'
import socket, sys, time
host, port, user = sys.argv[1], int(sys.argv[2]), sys.argv[3]
IAC, WILL, WONT, DO, DONT, SB, SE = 255, 251, 252, 253, 254, 250, 240

def filter_iac(buf: bytes) -> bytes:
    out = bytearray(); i = 0; n = len(buf)
    while i < n:
        b = buf[i]
        if b != IAC:
            out.append(b); i += 1; continue
        if i + 1 >= n: break
        cmd = buf[i + 1]
        if cmd == IAC:
            i += 2; continue
        if cmd in (WILL, WONT, DO, DONT):
            i += 3; continue
        if cmd == SB:
            j = i + 2
            while j + 1 < n and not (buf[j] == IAC and buf[j + 1] == SE):
                j += 1
            i = j + 2; continue
        i += 2
    return bytes(out)

class C:
    def __init__(self):
        self.s = socket.create_connection((host, port), timeout=8)
        self.s.settimeout(0.2)
        self.raw = b""
        self.text = ""
    def recv(self):
        try:
            data = self.s.recv(4096)
            if not data: return False
            self.raw += filter_iac(data)
            # Re-decode the whole buffer so Hangul split across recvs stays intact.
            self.text = self.raw.decode("utf-8", errors="replace")
            return True
        except socket.timeout:
            return True
        except OSError:
            return False
    def wait(self, needle, timeout=8.0):
        deadline = time.time() + timeout
        while time.time() < deadline:
            if needle in self.text: return
            if not self.recv(): break
        raise SystemExit(f"timeout waiting {needle!r}\n---\n{self.text}")
    def send(self, line):
        self.s.sendall((line + "\n").encode("utf-8"))

c = C()
c.wait("이름:")
c.send(user)
c.wait("비밀번호:")
c.send("password1")
deadline = time.time() + 8
while time.time() < deadline:
    c.recv()
    if "새로 만들까요" in c.text:
        c.send("y")
        c.wait("비밀번호:")
        c.send("password1")
        break
    if "들어왔어요" in c.text:
        break
else:
    raise SystemExit("auth failed\n---\n" + c.text)
c.wait("들어왔어요.")
for step in ["n", "n", "n", "n", "n", "e", "e"]:
    c.send(step)
c.wait("달빛골 장터")
c.send("보다 신문")
c.wait("한벽일보")
for step in ["s", "s", "s", "e"]:
    c.send(step)
c.wait("서당")
c.send("대화 청람")
c.wait("처음 보는 얼굴이군")
c.send("대화 청람")
c.wait("또 왔군")
c.wait("만석상회 창고에 짐이 남아")
# Rolling 1s rate limit is 20 cmds; pause so the warehouse walk is not dropped.
time.sleep(1.2)
for step in ["w", "n", "n", "n", "e", "e", "e"]:
    c.send(step)
    time.sleep(0.05)
c.wait("살펴볼 것: 보부상 봇짐")
c.send("보다 짐")
c.wait("거짓 바닥")
c.wait("수레 축")
print(c.text.replace("\r\n", "\n").replace("\r", "\n"))
c.send("quit")
time.sleep(0.2)
c.s.close()
PY
