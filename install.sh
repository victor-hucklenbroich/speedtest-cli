#!/bin/sh
# Installer for speedtest-cli (https://github.com/victor-hucklenbroich/speedtest-cli).
# Downloads the release archive for this machine's OS/architecture, verifies
# its checksum, and installs the `speedtest` binary:
#
#   curl -fsSL https://raw.githubusercontent.com/victor-hucklenbroich/speedtest-cli/main/install.sh | sh
#
set -eu

REPO="victor-hucklenbroich/speedtest-cli"
BINARY="speedtest"
BASE_URL="https://github.com/$REPO"

fail() {
  echo "install.sh: $*" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

# --- Detect platform --------------------------------------------------------
case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *) fail "unsupported OS $(uname -s); on Windows, run install.ps1 instead (see README)" ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *) fail "unsupported architecture $(uname -m)" ;;
esac

# --- Resolve the version ----------------------------------------------------
if [ -n "${VERSION:-}" ]; then
  # Accept both "1.2.0" and "v1.2.0".
  case "$VERSION" in
    v*) version="$VERSION" ;;
    *) version="v$VERSION" ;;
  esac
else
  redirect="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "$BASE_URL/releases/latest")" ||
    fail "could not resolve the latest release (no releases yet?)"
  version="${redirect##*/}"
  case "$version" in
    v*) ;;
    *) fail "could not determine the latest release tag from $redirect" ;;
  esac
fi

# --- Download and verify ----------------------------------------------------
archive="speedtest-cli_${os}_${arch}.tar.gz"
url="$BASE_URL/releases/download/$version/$archive"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $BINARY $version ($os/$arch)..."
curl -fsSL -o "$tmp/$archive" "$url" || fail "download failed: $url"
curl -fsSL -o "$tmp/checksums.txt" "$BASE_URL/releases/download/$version/checksums.txt" ||
  fail "download failed: $BASE_URL/releases/download/$version/checksums.txt"

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$archive")"
else
  actual="$(shasum -a 256 "$tmp/$archive")"
fi
actual="${actual%% *}"
expected="$(awk -v name="$archive" '$2 == name { print $1 }' "$tmp/checksums.txt")"
[ -n "$expected" ] || fail "no entry for $archive in checksums.txt"
[ "$actual" = "$expected" ] || fail "checksum mismatch for $archive (expected $expected, got $actual)"

tar -xzf "$tmp/$archive" -C "$tmp" "$BINARY"

# --- Install ----------------------------------------------------------------
if [ -n "${BIN_DIR:-}" ]; then
  bin_dir="$BIN_DIR"
  mkdir -p "$bin_dir"
elif [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
  bin_dir="/usr/local/bin"
else
  bin_dir="$HOME/.local/bin"
  mkdir -p "$bin_dir"
fi

install -m 0755 "$tmp/$BINARY" "$bin_dir/$BINARY"
echo "Installed $bin_dir/$BINARY"
"$bin_dir/$BINARY" --version

case ":$PATH:" in
  *":$bin_dir:"*) ;;
  *)
    echo ""
    echo "warning: $bin_dir is not on your PATH. Add it with e.g.:"
    echo "  export PATH=\"$bin_dir:\$PATH\""
    ;;
esac
