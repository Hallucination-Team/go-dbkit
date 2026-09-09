#!/usr/bin/env bash
# Re-home the official DM Go driver source into third_party/dm.
#
# Background: the official driver distribution's go.mod declares module "dm",
# which cannot be fetched with go get, and replace directives do not
# propagate to downstream applications. The source is therefore copied into
# this repository with import paths rewritten, making go-dbkit self-contained.
#
# Usage: tools/rehome_dm_driver.sh /path/to/official-driver.zip
# Idempotent: re-running removes third_party/dm first and regenerates it.
# Driver upgrade: download the new official zip, re-run this script, then
# go mod tidy && go build ./...
set -euo pipefail

ZIP_PATH="${1:?Usage: tools/rehome_dm_driver.sh <official-driver-zip>}"
MODULE_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$MODULE_ROOT/third_party/dm"
NEW_IMPORT="github.com/Hallucination-Team/go-dbkit/third_party/dm"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

unzip -q -o "$ZIP_PATH" -d "$TMP"

SRC="$TMP/dm"
if [ ! -d "$SRC" ]; then
  SRC="$(dirname "$(find "$TMP" -name go.mod | head -1)")"
fi
[ -d "$SRC" ] || { echo "error: DM driver directory not found in zip" >&2; exit 1; }

rm -rf "$DEST"
mkdir -p "$MODULE_ROOT/third_party"
cp -R "$SRC" "$DEST"

# drop its standalone module definition; merge into the main module
rm -f "$DEST/go.mod" "$DEST/go.sum"

# rewrite subpackage imports: "dm/util" -> "github.com/Hallucination-Team/go-dbkit/third_party/dm/util"
# [a-z0-9]: the i18n subpackage contains digits; [a-z]* alone would leave "dm/i18n" unrewritten.
grep -rl '"dm/' "$DEST" --include='*.go' | while read -r f; do
  sed -i "s#\"dm/\([a-z0-9]*\)\"#\"$NEW_IMPORT/\1\"#g" "$f"
done

# verify: official LICENSE / CHANGELOG / VERSION must be preserved
for f in LICENSE CHANGELOG.md VERSION; do
  [ -f "$DEST/$f" ] || { echo "error: missing $DEST/$f" >&2; exit 1; }
done

# verify: no legacy dm/ imports may remain
# match the import shape only ("dm/<sub>" with a closing quote) so that message
# strings like panic("dm/security: ...") are not false positives.
if grep -rnE '"dm/[a-z0-9]+"' "$DEST" --include='*.go' | grep -v '_test.go' | grep -q .; then
  echo "error: unrewritten dm/ imports remain:" >&2
  grep -rnE '"dm/[a-z0-9]+"' "$DEST" --include='*.go' >&2
  exit 1
fi

echo "re-home complete: $DEST"
