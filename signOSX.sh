#!/bin/bash
set -euo pipefail

binary_path=$1
binary_os=$2

exec >> post_build_output.txt 2>&1

if [ "$binary_os" == "darwin" ]; then
  codesign --keychain build.keychain --sign "Mac Developer: Mweya Ruider (ZHBMW6QG35)" $binary_path
  codesign --verify -vvvv $binary_path
fi