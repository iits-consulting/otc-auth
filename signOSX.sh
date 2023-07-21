#!/bin/bash
set -euo pipefail

binary_path=$1
binary_os=$2

if [ "$binary_os" == "darwin" ]; then
  KEYCHAIN=build.keychain
  security create-keychain -p actions $KEYCHAIN
  security default-keychain -s $KEYCHAIN
  security unlock-keychain -p actions $KEYCHAIN
  curl -o AppleWWDRCAG3.cer https://www.apple.com/certificateauthority/AppleWWDRCAG3.cer
  security import AppleWWDRCAG3.cer -k $KEYCHAIN -T /usr/bin/codesign
  curl -o AppleRootCA.cer https://www.apple.com/appleca/AppleIncRootCertificate.cer
  security import AppleRootCA.cer -k $KEYCHAIN -T /usr/bin/codesign
  echo "${{ secrets.MAC_CERT }}" | base64 --decode > certificate.p12
  security import certificate.p12 -k $KEYCHAIN -P ${{ secrets.MAC_CERT_PASS }} -T /usr/bin/codesign
  codesign --keychain $KEYCHAIN --sign "Mac Developer: Mweya Ruider (ZHBMW6QG35)" $binary_path
  codesign --verify -vvvv $binary_path
fi