#!/usr/bin/env bash
#
# Makes the updatable packs of a CDS release: the packs WSO2 Updates is onboarded
# with and the wso2.com download is made from.
#
# The release packs stay the open-source build. A updatable pack is the same pack
# with what the update service needs added, none of which is built from this
# source:
#
#   bin/update_tool_setup.sh, .ps1   from the Updates team's update_tool_setup.zip
#   LICENSE.txt                      the latest WSO2 EULA, in place of Apache 2.0
#
# and with every text file checked to be UTF-8, without which an update can only
# replace a text file rather than apply a diff to it.
#
#   prepare-updatable-packs.sh <release-dir> <update_tool_setup.zip> <eula.txt> <out-dir>
#
# <release-dir> holds the release's cds-*.zip, with their .sha256 when there
# are any. The release packs are not changed; the updatable packs are written to
# <out-dir> under the same names, with fresh checksums.

set -euo pipefail

RELEASE_DIR="${1:?usage: prepare-updatable-packs.sh <release-dir> <update_tool_setup.zip> <eula.txt> <out-dir>}"
SETUP_ZIP="${2:?usage: prepare-updatable-packs.sh <release-dir> <update_tool_setup.zip> <eula.txt> <out-dir>}"
EULA="${3:?usage: prepare-updatable-packs.sh <release-dir> <update_tool_setup.zip> <eula.txt> <out-dir>}"
OUT_DIR="${4:?usage: prepare-updatable-packs.sh <release-dir> <update_tool_setup.zip> <eula.txt> <out-dir>}"

die() { echo "prepare-updatable-packs: $*" >&2; exit 1; }

[ -d "$RELEASE_DIR" ] || die "$RELEASE_DIR not found"
[ -f "$SETUP_ZIP" ]   || die "$SETUP_ZIP not found"
[ -s "$EULA" ]        || die "$EULA not found or empty"
grep -qi "license agreement" "$EULA" || die "$EULA does not look like the WSO2 EULA"
mkdir -p "$OUT_DIR"
OUT_DIR="$(cd "$OUT_DIR" && pwd)"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

unzip -q "$SETUP_ZIP" -d "$WORK/setup"
SETUP_SH="$(find "$WORK/setup" -name 'update_tool_setup.sh' -type f | head -1)"
SETUP_PS1="$(find "$WORK/setup" -name 'update_tool_setup.ps1' -type f | head -1)"
[ -n "$SETUP_SH" ]  || die "$SETUP_ZIP has no update_tool_setup.sh"
[ -n "$SETUP_PS1" ] || die "$SETUP_ZIP has no update_tool_setup.ps1"

sha256() {
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$1"; else shasum -a 256 "$1"; fi
}

prepared=0
for zip in "$RELEASE_DIR"/cds-*.zip; do
  [ -e "$zip" ] || continue
  pack="$(basename "$zip" .zip)"

  # What is taken from the release has to be what the release published.
  if [ -f "$zip.sha256" ]; then
    [ "$(sha256 "$zip" | cut -d' ' -f1)" = "$(cut -d' ' -f1 "$zip.sha256")" ] \
      || die "$pack.zip does not match its published checksum"
  fi

  rm -rf "$WORK/pack"
  mkdir -p "$WORK/pack"
  unzip -q "$zip" -d "$WORK/pack"
  home="$WORK/pack/$pack"
  [ -d "$home" ] || die "$pack.zip does not hold $pack/"

  # The update service identifies the pack by this line, so it has to be the
  # name the pack is published under.
  [ -f "$home/updates/product.txt" ] || die "$pack carries no updates/product.txt"
  [ "$(tr -d '[:space:]' < "$home/updates/product.txt")" = "$pack" ] \
    || die "$pack: updates/product.txt says $(cat "$home/updates/product.txt")"

  # bin/ and updates/ stay siblings: the setup script reads ../updates/product.txt.
  mkdir -p "$home/bin"
  install -m 755 "$SETUP_SH" "$home/bin/update_tool_setup.sh"
  install -m 644 "$SETUP_PS1" "$home/bin/update_tool_setup.ps1"

  cp "$EULA" "$home/LICENSE.txt"
  chmod 644 "$home/LICENSE.txt"

  not_utf8="$(find "$home" -type f -exec file --mime-encoding {} + \
    | grep -v -E ': +(utf-8|us-ascii|binary)$' || true)"
  [ -z "$not_utf8" ] || die "$pack has text files that are not UTF-8:
$not_utf8"

  rm -f "$OUT_DIR/$pack.zip"
  (cd "$WORK/pack" && zip -rq "$OUT_DIR/$pack.zip" "$pack")
  (cd "$OUT_DIR" && sha256 "$pack.zip" > "$pack.zip.sha256")

  echo "  -> $pack ($(cat "$home/updates/product.txt"))"
  prepared=$((prepared + 1))
done

[ "$prepared" -gt 0 ] || die "no cds-*.zip in $RELEASE_DIR"
echo "Prepared $prepared updatable pack(s) in $OUT_DIR"
