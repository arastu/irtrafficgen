> [!CAUTION]
> **Use only where you are allowed to.** This program can send real HTTPS (and optional DNS) traffic to hosts and IP ranges on the internet. Run it only on networks and targets you **own**, have **written authorization** for, or fully control in a **lab**. Misuse may violate law or contracts. This is not legal advice. The software is **as-is**; maintainers disclaim liability. Defaults favor **dry-run**; **`--live`** is entirely your responsibility.

# irtrafficgen

**irtrafficgen** is an Iran-focused traffic generator for authorized testing. It samples targets from **embedded** V2Ray-format `geosite.dat` and `geoip.dat` (Iran-related geosite lists and `geoip:ir`), then issues HTTPS operations under rate limits and optional asymmetric (download/upload) profiles.

## Quick start

```bash
make check-assets build
./irtrafficgen inspect
./irtrafficgen run --config config.example.yaml
./irtrafficgen run --config config.example.yaml --live
./irtrafficgen help
```

- Without `--config`, the program uses **built-in defaults** (see [Configuration defaults](#configuration-defaults)).
- **`run`** is the default subcommand (`./irtrafficgen` equals `./irtrafficgen run`).
- **`--live`** forces real network I/O even if `dry_run: true` in the file.
- **`--dry-run true|false`** overrides the `dry_run` YAML value (unless `--live` wins).

## How targets and traffic work

| Source | When |
|--------|------|
| **Geosite lists** | Domain names from embedded lists. If `geosite_lists` is **omitted** (YAML `null`), every list whose code matches `category-*-ir` is used. If set to **`[]`**, no geosite domains—**GeoIP-only** mode. If set to a **named array**, only those lists (names must exist in `geosite.dat`; use `inspect`). |
| **`geoip:ir`** | Random public IPv4/IPv6 from the embedded `geoip:ir` CIDRs (subject to `safety.deny_private_ips`). |

Each worker tick: pick a target, then (if `asymmetric.enabled`) pick an operation **`head`** / **`get`** / **`post`** by weight; otherwise always **`head`**. Live mode uses TLS 1.2+ with HTTP/2 where supported.

## CLI commands

| Command | Purpose |
|---------|---------|
| **`inspect`** | Print geosite list codes, rule counts, and `geoip:ir` CIDR stats. Exit **1** if `geoip:ir` is missing. |
| **`version`** | Print build version from `internal/version`. |
| **`run`** | Worker pool: global QPS limit, optional per-host limit, jitter between ops, metrics at end if `--verbose`. |

Common **`run`** flags: `--config <path>`, `--live`, `--once`, `--dry-run`, `--verbose`. See `./irtrafficgen help run`.

## Configuration file

YAML keys match the struct tags in `internal/config`. A commented template lives in [config.example.yaml](config.example.yaml).

### Top-level

| Key | Type | Description |
|-----|------|-------------|
| **`dry_run`** | bool | If **true**, no DNS or HTTPS I/O; logs planned ops and (for asymmetric) adds **estimated** bytes to session counters only. |
| **`www_root_domain`** | bool | For geosite **root-domain** rules, use `www.<domain>` instead of the apex. |
| **`insecure_tls`** | bool | Skip TLS certificate verification (testing only; unsafe on untrusted networks). |
| **`dns_enabled`** | bool | After sampling a **hostname** target, perform a DNS lookup before HTTPS (metrics only; does not change the chosen host). |
| **`sni_for_ip`** | string | TLS Server Name when connecting to an **IP** target (port 443). Many servers require a sensible SNI. |
| **`per_host_limiter_map_max`** | int | Max distinct hosts in the per-host limiter map; if exceeded, oldest entries may be evicted. Invalid values fall back to **2048** at validate time. |

### `limits`

| Key | Description |
|-----|-------------|
| **`global_qps`** | Token-bucket rate for starting operations (all workers combined). Must be **> 0**. |
| **`max_in_flight`** | Number of concurrent worker goroutines. Each waits on `global_qps` (and jitter) before work. Minimum **1**. |
| **`per_host_qps`** | Optional per-host HTTPS rate cap; **0** disables. |
| **`https_timeout_seconds`** | Deadline for each client operation (TLS + request + reading capped body). Minimum **1**. |
| **`jitter_min_ms`**, **`jitter_max_ms`** | Random sleep after acquiring a global slot, before the HTTPS op. **`jitter_max_ms` ≥ `jitter_min_ms`**. |

### `geosite_lists`

Array of list **codes** (strings) present in embedded `geosite.dat`. Omit for auto-discovery of `category-*-ir`; use `[]` for GeoIP-only sampling (requires positive **`weights.geoip_ir`**).

### `weights`

| Key | Description |
|-----|-------------|
| **`geoip_ir`** | Relative weight for sampling a **`geoip:ir`** IP vs. geosite domain targets. |
| **`geosite`** | Map of **list code → weight** for lists in `geosite_lists`. Omitted or missing keys default to **2.0** per list at validation. |

The sum of `geoip_ir` and all referenced geosite weights must be **positive**.

### `safety`

| Key | Description |
|-----|-------------|
| **`deny_private_ips`** | When sampling GeoIP targets, skip private/reserved addresses so only public IPs are used. |
| **`allowed_domain_suffixes`** | If non-empty, only allow sampled hostnames that match one of these suffixes (e.g. `example.ir`). Empty allows all sampled hosts. |

### `asymmetric`

When **`enabled: false`** (default in code if unset), behavior matches the original tool: **HTTPS `HEAD`** only to `/`.

When **`enabled: true`**, each tick samples **`head`**, **`get`**, or **`post`** by **`operation_weights`**. Requires **`download_max_bytes` ≥ 1** and **`upload_max_bytes` ≥ 1**; operation weights must be non-negative with **positive sum**.

| Key | Description |
|-----|-------------|
| **`operation_weights.head`** / **`.get`** / **`.post`** | Relative probabilities for `HEAD`, capped-body `GET`, and `POST` with a capped zero-filled body. |
| **`download_max_bytes`** | Max bytes read from each **GET** response body (hard cap via `LimitReader`). |
| **`upload_max_bytes`** | **POST** body size (bytes) sent per operation. |
| **`target_rx_tx_ratio`** | If **> 0**, slowly bias weights toward session **`bytes_received / bytes_sent`**. **0** keeps fixed weights only. |
| **`ratio_adjust_interval_seconds`** | Minimum seconds between ratio adjustments; if **0** and `target_rx_tx_ratio > 0`, validation sets **30**. |
| **`max_concurrent_large_downloads`** | Cap concurrent **GET** ops across workers. If **GET** weight **> 0** and this is **< 1**, validation sets **2**. |
| **`get_path`**, **`post_path`** | URL paths (must start with `/`; empty normalizes to **`/`**). |
| **`global_qps_large`** | Extra token bucket applied only to **GET** ops (**0** = no extra limit beyond `limits.global_qps`). |
| **`receive_bytes_per_second`** | Throttle **downstream** body reads (**0** = off). |
| **`send_bytes_per_second`** | Throttle **upstream** **POST** body writes (**0** = off). |
| **`max_redirects`** | Max redirects per request; **0** at validate time becomes **3**. |
| **`transport_max_idle_conns_per_host`** | HTTP transport tuning; **0** → **32** when enabled. |
| **`transport_idle_conn_timeout_seconds`** | Idle keep-alive timeout; **0** → **90** when enabled. |
| **`total_download_cap_bytes`** | Session cap on bytes counted toward large downloads; further **GET**s are downgraded to **HEAD** when exceeded (**0** = unlimited). |
| **`total_upload_cap_bytes`** | Same idea for upload side vs. **POST** (**0** = unlimited). |
| **`head_estimate_rx_bytes`**, **`head_estimate_tx_bytes`** | Bytes credited to metrics for successful **HEAD** when asymmetric is enabled; **0** → **4096** each at validate time. |

**Asymmetric dry-run:** same operation mix as live; **no** TLS/DNS/HTTP; estimated **`get`**/**`post`** bytes use caps from config (see [config.example.yaml](config.example.yaml) comments).

### Configuration defaults

If **`--config`** is empty, the program starts from **`config.Default()`** in `internal/config/config.go` (e.g. `dry_run: true`, `limits.global_qps: 5`, `max_in_flight: 20`, `geoip_ir` weight **2**, `geosite_lists` discovered at validate time from embedded data, `dns_enabled: false`, `insecure_tls: true`, asymmetric **off**). A YAML file **merges over** those defaults for keys you set.

Validation runs after load (and may **fill in** geosite weights, paths, and asymmetric defaults as described above). Run **`inspect`** and **`go test ./... -run TestDefaultValidateEmbedded`** after changing embedded `.dat` files.

## Session metrics

Printed to stderr when **`--verbose`** is set at the end of a run. Includes **`https_attempts`**, **`https_success`**, **`bytes_sent`**, **`bytes_received`**, per-op **`head`/`get`/`post`** attempt and success counts, DNS counters, error classes, and **`rx_tx_ratio`** (bytes received ÷ bytes sent) when meaningful. Counters are **uint64** (theoretical wrap on extremely long runs).

## Build and embedded data

- **Go 1.24+**
- Replace files under **`internal/assets/`** when updating `geosite.dat` / `geoip.dat`, then rebuild.

```bash
make check-assets build
# or: VERSION=1.0.0 make build
```

Large `.dat` files may use [Git LFS](https://git-lfs.com/).

## Verify embedded data

```bash
./irtrafficgen inspect
go test ./... -run TestDefaultValidateEmbedded
```
