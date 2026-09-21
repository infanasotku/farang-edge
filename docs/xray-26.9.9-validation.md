# Xray 26.9.9 upgrade validation

Tested on 2026-09-21 in an isolated Linux workshop worktree using Go 1.27.1.
The original double-close panic did not recur in the functional reload tests,
but the race detector found a separate upstream WireGuard initialization race.
This upgrade has not been deployed and does not have a clean concurrency check.

## Release pin

Release `v26.9.9` resolves to commit
`52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120`. Its module still declares
`github.com/xtls/xray-core`, so Go rejects the v26 release tag directly. The
verified commit resolves to `v1.260327.1-0.20260908222543-52a412d9e2f5`.
The dependency requires Go 1.27; Docker and PR CI now use Go 1.27.1.

## Results

| Check | Result |
| --- | --- |
| `go test -v ./... -timeout=180s` | Passed |
| `go test -v ./internal/xray -run TestWireGuardReload -count=10 -timeout=180s` | Passed: 100 reloads, four concurrent traffic workers |
| `go test -race -v ./... -timeout=180s` | Failed: upstream bind initialization race |
| `go vet ./...` | Passed |
| Production Dockerfile build | Passed, image built locally in the workshop |

The regression creates a real userspace WireGuard peer, random ephemeral keys,
and a TCP echo service reachable only through the encrypted tunnel. Every new
generation must successfully exchange data. Every retired outbound must reject
a late dispatch with a WireGuard closed error. Requests interrupted during
replacement are allowed to fail; successful traffic after each reload is required.
Restarts are spaced beyond WireGuard's 20 ms handshake flood-protection window.
No production credentials, endpoints, kernel TUN, or host network changes are used.

PR CI retains ordinary test mode and now runs all packages, including this
regression. The additional race check was not made a required CI command because
the unmodified requested release fails it; the failure remains unresolved here.

## Unresolved upstream race

The race report identifies a read of `bind.listenFunc` in
`proxy/wireguard/bind.go:37` by `Device.RoutineTUNEventReader`, concurrent with its
assignment in `proxy/wireguard/client.go:343` by `Handler.init`.
`device.NewDevice` is called at line 341 and starts goroutines before the bind's
callbacks are assigned. The userspace TUN already has an Up event queued.

This is separate from the old repeated-channel-close panic. The report proves
unsynchronized initialization; a nil-function panic is a possible consequence,
not an observed failure in these functional runs. An upstream correction needs
to synchronize bind initialization with device startup, followed by a successful
race-enabled regression. The release source has not been patched or replaced
with a fork as part of this exact-version upgrade.

Sources: [release](https://github.com/XTLS/Xray-core/releases/tag/v26.9.9),
[handler initialization](https://github.com/XTLS/Xray-core/blob/52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120/proxy/wireguard/client.go#L341),
[bind read](https://github.com/XTLS/Xray-core/blob/52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120/proxy/wireguard/bind.go#L37).
