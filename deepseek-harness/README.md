# DeepSeek Harness Runtime

This directory owns the ClawManager runtime images for DeepSeek Harness.

- `Dockerfile.lite` builds the pooled, headless Lite runtime. The shared
  `clawmanager-agent` starts one isolated `dsh web` process per instance. DSH
  listens only on a private loopback port; the agent-facing gateway validates
  the ClawManager instance token and holds DSH's browser-session cookie inside
  the runtime process boundary.
- `Dockerfile.pro` builds the dedicated Webtop Pro runtime. DeepSeek Harness
  runs on loopback port `3080` and opens automatically in the desktop browser.

Both images pin `@deepseek-ai/dsh` and apply the managed Cordis provider patch
from `config/managed.cordis.patch.yml`. The Lite image also pins the `pnpm` CLI
required by dsh-market so pooled non-root instances never need to modify the
system Node installation at runtime. ClawManager supplies the OpenAI-compatible
base URL, instance credential, and allowed model list at runtime.

Build both images from the repository root:

```bash
docker build -f deepseek-harness/Dockerfile.lite -t deepseek-harness-lite:local .
docker build -f deepseek-harness/Dockerfile.pro -t deepseek-harness:local .
```

## CI images

Pushes to `main` that change DeepSeek Harness or either shared agent dependency
automatically publish multi-architecture `deepseek-harness` and
`deepseek-harness-lite` images through `.github/workflows/docker-ghcr.yml`.
Each run updates `latest` and the current Shanghai-date tag.

## Host resource prerequisites

Both images raise the DSH process soft `nofile` limit to `65536` (bounded by
the container hard limit) before starting `dsh web`. Override it with
`DSH_NOFILE_SOFT` only when the deployment has a deliberate lower limit.

Linux inotify limits are node-level kernel settings and cannot be changed by
an unprivileged runtime container. Apply the following persistent settings on
every Kubernetes node that may schedule DeepSeek Harness or desktop runtime
pods:

```text
fs.inotify.max_user_instances = 1024
fs.inotify.max_user_watches = 524288
```

Store them in `/etc/sysctl.d/99-clawmanager-inotify.conf` and apply them with
the operating system's normal sysctl provisioning. The startup guard prints
the effective nofile and inotify values and warns when the host limits are
below these recommendations.
