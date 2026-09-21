farang-edge is the data plane component of the Farang proxy network.

It is responsible for handling and proxying traffic according to configuration provided by farang-control.

## Configuration

The edge reads its configuration from environment variables:

- `ENGINE_ID`: engine UUID in farang-control.
- `CONTROL_BASE_URL`: farang-control base URL.
- `CONTROL_AUTH_TOKEN`: edge API key used for Control requests.
- `REPLACEMENT_PERMIT`: optional one-time permit for deliberately replacing a live engine instance.

To perform a planned replacement, issue a permit in the Control admin panel, set `REPLACEMENT_PERMIT` on the new edge,
and start it before the permit expires. The edge sends the value only in the `X-Replacement-Permit` header of its
registration request. It is never added to heartbeat or spec requests and is not logged. Remove the environment value
after successful registration; the permit is consumed by Control and cannot be reused.

## Published images

Test and production workflows show the published image tag, digest, immutable reference, and pull command in the GitHub
Actions run summary. Each run also provides an `edge-<environment>-image-<run number>` artifact containing the same
values in `image-reference.env`, so the image can be located without searching build logs.

## Development and tests

Use Go 1.27.1. The embedded Xray release is 26.9.9, pinned to release commit
`52a412d9e2f5c2a5142b1b4e2ab3771dacb8b120`. Go records this as
`v1.260327.1-0.20260908222543-52a412d9e2f5` because the upstream `v26.9.9` tag
is incompatible with its module path.

Run `go test ./...`. The WireGuard regression test creates an isolated
userspace peer with random keys and checks encrypted traffic during ten engine
reloads, including late requests on retired handlers. It requires no production
credentials, external endpoint, kernel TUN, or elevated privileges. Run
`go test ./internal/xray -run TestWireGuardReload -count=10` for 100 reloads.

An additional `go test -race ./...` check currently reports an upstream WireGuard
initialization race in this release. See [the workshop validation report](docs/xray-26.9.9-validation.md)
for results and the unresolved limitation.
