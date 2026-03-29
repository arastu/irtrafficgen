> [!CAUTION]
> **Disclaimer — use only where you are allowed to.** This program can initiate real network traffic toward **live** hosts and IP ranges on the internet. Run it **only** on systems and networks you own, with **explicit written authorization**, or in **isolated lab environments** you control. Using it without permission may break **criminal computer-misuse laws** and **contracts** in your jurisdiction. This is **not legal advice**. The software is provided **as-is**; maintainers **disclaim liability** for misuse or damage. Defaults keep **dry-run** behavior; **`--live`** traffic is entirely **your responsibility**.

# irtrafficgen

Iran-focused traffic generator for **authorized lab testing**. It samples targets from **embedded** V2Ray-format `geosite.dat` and `geoip.dat` (Iran geosite lists and `geoip:ir`).

## Requirements

- Go 1.24+

## Build

Replace data files under `internal/assets/` when you want a new dataset, then rebuild.

```bash
make check-assets build
# or: VERSION=1.0.0 make build
```

Large `.dat` files may warrant [Git LFS](https://git-lfs.com/); this repo documents that here only.

## Commands

```bash
./irtrafficgen inspect
./irtrafficgen version
./irtrafficgen run --once
./irtrafficgen --once                    # same as default subcommand `run`
./irtrafficgen run --config config.example.yaml --live
```

- **`inspect`** — Lists geosite entry codes with rule counts and `geoip:ir` CIDR stats. Exits with status 1 if `geoip:ir` is missing.
- **`version`** — Prints `internal/version`
- **`run`** (default) — Worker pool with global QPS limit, optional per-host limit, jitter, dry-run by default. **`--live`** enables real HTTPS (and optional DNS when `dns_enabled` is true in config).
- **`--dry-run true|false`** — Overrides `dry_run` in config; **`--live`** still forces real I/O.

Use `./irtrafficgen help` for full flags.

By default, **every embedded geosite list** whose code matches `category-*-ir` (case-insensitive) is used. Omit `geosite_lists` in YAML for that behavior. Set `geosite_lists: []` explicitly for **`geoip:ir` only**. Override with a named `geosite_lists` array to use a subset. Run `inspect` to see codes in your `geosite.dat`.

## Configuration

See [config.example.yaml](config.example.yaml). Empty `--config` uses built-in defaults merged from [internal/geo/iran.go](internal/geo/iran.go).

## Verify data after an update

```bash
./irtrafficgen inspect
go test ./... -run TestDefaultValidateEmbedded
```
