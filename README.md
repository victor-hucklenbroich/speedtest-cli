# speedtest-cli

A small, zero-dependency Go CLI that measures latency, download and upload bandwidth. 
Uses personal [Cloudflare Worker](worker/) as a bulk download / upload server. A different endpoint can be configured using a the `--url` flag or setting the `SPEEDTEST_URL` env variable.

## Build

```
go build .
```

This produces a single static `speedtest` binary. Cross-compile with the
usual `GOOS`/`GOARCH` environment variables.

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

- Worker code lives in [`worker/`](worker/) and deploys automatically via
  GitHub Actions on pushes to `main` that touch `worker/**`.
- Test the CLI against a local worker:
  `cd worker && npx wrangler dev --port 8787`, then
  `./speedtest --url http://localhost:8787`.
- The usage block above is generated from the CLI's `--help` text: run
  `go generate ./...` after changing flags.
