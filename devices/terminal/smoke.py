"""Drive a terminal build through a pseudo-terminal and check that it renders and quits.

Usage: python3 -I smoke.py <binary or .wasm> [--kitty] [program flags]
WASI builds run under wasmtime. Exits non-zero when the program fails, draws nothing,
ignores the quit key, or writes no snapshot.
"""

import fcntl
import os
import pty
import re
import select
import struct
import subprocess
import sys
import tempfile
import termios
import time

program, extra = sys.argv[1], sys.argv[2:]
kitty = "--kitty" in extra
if kitty:
    extra.remove("--kitty")
master, slave = pty.openpty()
fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 80, 250, 0, 0))
attrs = termios.tcgetattr(slave)
attrs[0] &= ~(termios.ICRNL | termios.IXON)
attrs[3] &= ~(termios.ICANON | termios.ECHO | termios.ISIG)
termios.tcsetattr(slave, termios.TCSANOW, attrs)
outdir = tempfile.mkdtemp(prefix="terminal-smoke-")
snapshot = os.path.join(outdir, "snapshot.png")
if program.endswith(".wasm"):
    cmd = ["wasmtime", "run", "--env", "TERM=xterm-256color",
           "--env", "NTCHARTS_KITTY=" + ("supported" if kitty else "unsupported"),
           "--dir", f"{outdir}::/out", program, "-snapshot", "/out/snapshot.png"]
else:
    cmd = [program, "-snapshot", snapshot]
env = {**os.environ, "TERM": "xterm-256color",
       "NTCHARTS_KITTY": "supported" if kitty else "unsupported"}
proc = subprocess.Popen(cmd + ["-stats"] + extra, env=env, stdin=slave, stdout=slave, stderr=slave, close_fds=True)
os.close(slave)
buf = b""


def pump(seconds):
    global buf
    end = time.time() + seconds
    while time.time() < end:
        ready, _, _ = select.select([master], [], [], 0.05)
        if ready:
            try:
                buf += os.read(master, 65536)
            except OSError:
                return


pump(1.5)
idle = buf.count("▄".encode())
try:
    for key in (b"\r", b"\r", b"\x1b[C"):
        os.write(master, key)
        pump(0.3)
    os.write(master, b"q")
except OSError:
    pass
end = time.time() + 6
while proc.poll() is None and time.time() < end:
    pump(0.1)
code = proc.poll()
stats = buf.rsplit(b"frames=", 1)[-1].split(b"\r")[0].decode(errors="replace") if b"frames=" in buf else ""
print(f"{os.path.basename(program)}: exit={code} idle_halfblocks={idle} frames={stats}")
if code != 0:
    print("program did not exit cleanly on q:", buf[-400:].decode(errors="replace"), file=sys.stderr)
    if code is None:
        proc.kill()
    sys.exit(1)
if kitty and not re.search(rb"\x1b_G[^;\x1b]*\ba=[tT](?:[,;])", buf):
    print("no Kitty image transmission", file=sys.stderr)
    sys.exit(1)
if kitty and b"\x1b_Ga=d,d=I" not in buf:
    print("no Kitty image cleanup on quit", file=sys.stderr)
    sys.exit(1)
if not kitty and idle == 0:
    print("no half-block output while idle", file=sys.stderr)
    sys.exit(1)
# The pseudo-terminal never answers the OSC 7501 query, so the program hosts
# its own action bar and turns on mouse reporting for it.
if b"Pop" not in buf or b"More" not in buf or b"\x1b[?1002h" not in buf:
    print("no hosted action bar", file=sys.stderr)
    sys.exit(1)
if b"\x1b]7501;state=idle" in buf:
    print("action report leaked to the terminal", file=sys.stderr)
    sys.exit(1)
if not os.path.exists(snapshot):
    print("no snapshot written", file=sys.stderr)
    sys.exit(1)
