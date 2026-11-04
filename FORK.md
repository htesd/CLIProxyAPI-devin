# CLIProxyAPI-devin

Downstream of [router-for-me/CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)
(MIT). Branch `devin-fixes` is based on upstream tag `v8.0.53` and carries only
the changes below.

## Devin GPT reasoning signature round trip

For OpenAI models, Devin streams the thought signature with
`signature_type: "openai"` as a JSON array of native OpenAI reasoning items
(`[{"id":"rs_...","type":"reasoning","encrypted_content":"gAAAA...",...}]`).
Upstream v8.0.53 only accepts bare provider signatures or `sealed.v1.` envelopes,
so this signature was dropped when translating to the Responses API and GPT
models re-reasoned from scratch after every tool call.

This fork wraps the raw signature as `devin.openai.v1.<base64url>` in the
reasoning item's `encrypted_content`, accepts that envelope when the client
echoes it back, and restores the original bytes with `signature_type: "openai"`
before sending them to Devin.

- `internal/signature/devin_openai.go` - wrap/unwrap helpers
- `internal/translator/openai/interactions/responses/` - response and request directions
- `internal/runtime/executor/devin_executor.go` - `parseSignatureBytes` restores the raw bytes

## devin-nofast plugin

`examples/plugin/devin-nofast/go` - request normalizer that strips fast/priority
selection before requests reach Devin. See its README.
