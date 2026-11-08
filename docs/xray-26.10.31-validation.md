# Xray 26.10.31 upgrade validation

Validated on 2026-11-08. The official latest stable release API and Git tag resolve
v26.10.31 to `a4a3dff5b20c2faae567b742c172c7b95e75c083`. The Go module remains
`github.com/xtls/xray-core`, pinned as
`v1.260327.1-0.20261031132019-a4a3dff5b20c`. Go 1.27 remains required; the
existing Go 1.27.1 Docker and CI pins are retained.

The dependency graph now selects `golang.org/x/net v0.61.0`. Its
`http2/transport_wrap.go` includes `dialForRoundTrip` with a per-address in-flight
dial map, restoring coalescing on the Go 1.27 wrapper path. This addresses the
upstream connection-fanout issue; it does not establish the cause of any ISP
throttling or the production server's stale sockets.

The upstream userspace WireGuard TUN API now takes a connection handler and
returns three values instead of exposing a gVisor stack. The existing regression
uses an echo handler restricted to the same TCP endpoint. Encrypted traffic,
four concurrent workers, every post-reload echo, and every retired-handler
assertion are preserved; no retries or relaxed deadlines were added.

## Validation

Linux amd64 tests used a disposable workshop worktree and `golang:1.27.1`.
The production image build used the repository Dockerfile (`golang:1.27.1-alpine`).

| Check | Result |
| --- | --- |
| Linux `go test ./... -timeout=180s` | Passed |
| Linux `go test -race ./... -timeout=180s` | Passed |
| Linux `go vet ./...` | Passed |
| Linux WireGuard regression, `-count=10` | Passed: 100 reloads |
| Production Dockerfile build | Passed |
| macOS arm64 Go 1.27.0 full suite and vet | Passed |
| macOS WireGuard regression, `-count=10` | Passed: 100 reloads |
| macOS full race suite | Failed once: closed pipe after reload 6 |
| macOS WireGuard race regression, `-count=3` | Passed: 30 reloads |

The macOS race failure was `traffic after reload 6: io: read/write on closed
pipe`, without a data-race report. Subsequent passes do not explain that failure.
The previously documented 26.9.9 initialization race was not reproduced by these
checks; this is not proof of its absence. Keep the upgrade in draft pending
review of the intermittent WireGuard failure. No production edge was deployed.

Sources: [release](https://github.com/XTLS/Xray-core/releases/tag/v26.10.31),
[upstream issue](https://github.com/XTLS/Xray-core/issues/6797),
[x/net v0.61.0](https://github.com/golang/net/blob/v0.61.0/http2/transport_wrap.go).
