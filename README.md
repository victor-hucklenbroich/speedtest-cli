# speedtest-cli

<img src="./demo/demo.gif" alt="drawing" width="650"/>

A small, zero-dependency Go CLI that measures latency, download and upload bandwidth. 
Uses personal [Cloudflare Worker](worker/) as a bulk download / upload server. A different endpoint can be configured using a the `--url` flag or setting the `SPEEDTEST_URL` env variable.

## Install

**Homebrew (macOS)**

```
brew install --cask victor-hucklenbroich/tap/speedtest-cli
```

**Shell installer (macOS, Linux)**

```
curl -fsSL https://raw.githubusercontent.com/victor-hucklenbroich/speedtest-cli/main/install.sh | sh
```

The script picks the right binary for your OS/architecture, verifies its
checksum, and installs `speedtest` to `/usr/local/bin` (or `~/.local/bin`
when that isn't writable). Pin a version or choose the directory with
environment variables:

```
curl -fsSL https://raw.githubusercontent.com/victor-hucklenbroich/speedtest-cli/main/install.sh | VERSION=v1.2.0 BIN_DIR=~/bin sh
```

**PowerShell installer (Windows)**

```
irm https://raw.githubusercontent.com/victor-hucklenbroich/speedtest-cli/main/install.ps1 | iex
```

Same behavior as the shell installer: picks amd64/arm64, verifies the
checksum, installs `speedtest.exe` to `%LOCALAPPDATA%\Programs\speedtest-cli`,
and adds that folder to your user PATH. `$env:VERSION` and `$env:BIN_DIR`
pin a release or change the directory.

Prebuilt archives for all platforms are also on the
[releases page](https://github.com/victor-hucklenbroich/speedtest-cli/releases).


## Usage

<!-- BEGIN USAGE -->

```
Usage: speedtest [flags]

Flags:
  -down
    	measure download
  -ping
    	measure ping
  -plain
    	plain line output instead of the animated TUI (automatic when stdout is not a terminal)
  -size string
    	transfer size, e.g. 25MB or 500KB (bare number = MB, max 1GB); append + to escalate from there
  -up
    	measure upload
  -url string
    	speedtest server base URL
  -version
    	print version and exit

Phase flags combine: --ping --down runs ping and download

--size 25MB runs a single 25 MB transfer
--size 25MB+ walks the normal escalation ladder but starts it at 25 MB.

The server URL is resolved in this order:
  1. --url flag
  2. SPEEDTEST_URL environment variable
  3. built-in default: https://speedtest-worker.speedtest-cli.workers.dev
```

<!-- END USAGE -->

## Development

- Use `go build` to produce a single static `speedtest` binary in the repository root directory, which can be used with `./speedtest`.
- Worker code lives in [`worker/`](worker/) and deploys automatically via
  GitHub Actions on pushes to `main` that touch `worker/**`.
- Test the CLI against a local worker:
  `cd worker && npx wrangler dev --port 8787`, then
  `./speedtest --url http://localhost:8787`.
- The usage block above is generated from the CLI's `--help` text: run
  `go generate ./...` after changing flags.
- Releasing: pushing a tag like `v1.2.3` triggers the
  [release workflow](.github/workflows/release.yml), which cross-compiles the
  binaries with [GoReleaser](https://goreleaser.com) (see `.goreleaser.yaml`),
  publishes them as a GitHub Release, and bumps the cask in
  [victor-hucklenbroich/homebrew-tap](https://github.com/victor-hucklenbroich/homebrew-tap).
  The version is injected at build time; the `-indev` value in
  `internal/cli` only shows up in local `go build` binaries.
