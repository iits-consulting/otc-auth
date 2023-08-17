#!/bin/bash
set -euo pipefail

binary_path=$1
binary_os=$2
binary_version=$3
binary_arch=$4

exec >> post_build_output.txt 2>&1

if [ "$binary_os" == "darwin" ]; then
  codesign --keychain build.keychain --sign "Developer ID Application: Impresssol GmbH (J486APST94)" --options=runtime --timestamp --deep $binary_path
  codesign -vvvv --deep --strict $binary_path
  codesign -dvvv --entitlements :- $binary_path
  echo "[*] Binary signed"
  zip "$binary_path.zip" $binary_path
  xcrun notarytool submit "$binary_path.zip" \
               --apple-id "mweya.ruider@iits-consulting.de" \
               --password $NOTARYTOOL_PASS \
               --team-id "J486APST94" \
               --wait
  echo "[*] Binary submitted"
fi