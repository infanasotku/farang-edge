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
