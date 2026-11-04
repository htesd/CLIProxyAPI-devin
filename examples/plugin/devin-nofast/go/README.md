# devin-nofast

Request normalizer plugin that removes fast/priority selection from requests
bound for the Devin executor (target format `interactions`):

- deletes `service_tier` (Codex `/fast` sends `priority`);
- deletes `thinking_level` when it is `fast` or `priority`;
- trims a `-fast` / `-priority` suffix from `thinking_level` (`high-fast` -> `high`).

Devin's fast (`-priority`) model variants are not covered by the ChatGPT plan
partner billing and some fast variants are rejected upstream with 403, which
would otherwise cool down the credential.

Build (glibc must match the CPA runtime image, e.g. Debian bookworm):

```bash
go build -buildvcs=false -buildmode=c-shared -trimpath -o devin-nofast.so .
```

Enable:

```yaml
plugins:
  enabled: true
  dir: "plugins"
  configs:
    devin-nofast:
      enabled: true
      priority: 1
```

The plugin system initializes at startup; restart CPA after enabling it.
