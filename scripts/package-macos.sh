#!/usr/bin/env bash
set -euo pipefail

VERSION="v0.6.3-alpha"
PLATFORM="darwin/universal"
SKIP_FRONTEND="false"
SIGN="false"
NOTARIZE="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -Version|--version)
      VERSION="${2:?version is required}"
      shift 2
      ;;
    -Platform|--platform)
      PLATFORM="${2:?platform is required}"
      shift 2
      ;;
    -SkipFrontend|--skip-frontend)
      SKIP_FRONTEND="true"
      shift
      ;;
    -Sign|--sign)
      SIGN="true"
      shift
      ;;
    -Notarize|--notarize)
      NOTARIZE="true"
      shift
      ;;
    *)
      echo "未知参数：$1" >&2
      exit 64
      ;;
  esac
done

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "macOS DMG 只能在 macOS runner 或 macOS 本机打包。" >&2
  exit 1
fi

if [[ -z "${VERSION// }" ]]; then
  echo "Version 不能为空" >&2
  exit 1
fi

VERSION_BODY="${VERSION#v}"
if [[ -z "$VERSION_BODY" ]]; then
  echo "Version 包含非法字符：$VERSION" >&2
  exit 1
fi
case "${VERSION_BODY:0:1}" in
  [A-Za-z0-9]) ;;
  *)
    echo "Version 包含非法字符：$VERSION" >&2
    exit 1
    ;;
esac
case "$VERSION_BODY" in
  *[!A-Za-z0-9.+-]*)
    echo "Version 包含非法字符：$VERSION" >&2
    exit 1
    ;;
esac
if [[ "$VERSION" != "$VERSION_BODY" && "${VERSION:0:1}" != "v" ]]; then
  echo "Version 包含非法字符：$VERSION" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DESKTOP_ROOT="$REPO_ROOT/desktop"
FRONTEND_ROOT="$DESKTOP_ROOT/frontend"
DIST_ROOT="$REPO_ROOT/dist"
MACOS_PACKAGING_ROOT="$REPO_ROOT/packaging/macos"
NORMALIZED_VERSION="${VERSION#v}"
TARGET_ARCH="${PLATFORM#darwin/}"
TARGET_NAME="nextunnel-${VERSION}-darwin-${TARGET_ARCH}"
STAGING_ROOT="$DIST_ROOT/$TARGET_NAME"
DMG_STAGING="$DIST_ROOT/${TARGET_NAME}-dmg"
DMG_PATH="$DIST_ROOT/${TARGET_NAME}.dmg"
APP_SOURCE="$DESKTOP_ROOT/build/bin/NexTunnel.app"
APP_TARGET="$STAGING_ROOT/NexTunnel.app"
SIGNING_STATE="unsigned-alpha"

mkdir -p "$DIST_ROOT"

if [[ "$SKIP_FRONTEND" != "true" ]]; then
  echo "构建桌面端前端"
  (cd "$FRONTEND_ROOT" && npm run build)
fi

echo "打包 macOS 桌面端 $VERSION ($PLATFORM)"
(
  cd "$DESKTOP_ROOT"
  wails build \
    -m \
    -s \
    -trimpath \
    -platform "$PLATFORM" \
    -o "NexTunnel" \
    -ldflags "-s -w -X main.AppVersion=$NORMALIZED_VERSION"
)

if [[ ! -d "$APP_SOURCE" ]]; then
  echo "未找到 Wails macOS 产物：$APP_SOURCE" >&2
  exit 1
fi

rm -rf "$STAGING_ROOT" "$DMG_STAGING" "$DMG_PATH"
mkdir -p "$STAGING_ROOT" "$DMG_STAGING"
cp -R "$APP_SOURCE" "$APP_TARGET"

if [[ "$SIGN" == "true" ]]; then
  if [[ -z "${MACOS_DEVELOPER_ID_APPLICATION:-}" ]]; then
    echo "启用签名需要设置 MACOS_DEVELOPER_ID_APPLICATION。" >&2
    exit 1
  fi
  # 使用 hardened runtime，为后续 notarization 做准备。
  codesign --force --deep --options runtime --timestamp --sign "$MACOS_DEVELOPER_ID_APPLICATION" "$APP_TARGET"
  SIGNING_STATE="signed"
fi

cp -R "$APP_TARGET" "$DMG_STAGING/NexTunnel.app"
ln -s /Applications "$DMG_STAGING/Applications"
cp "$MACOS_PACKAGING_ROOT/README.txt" "$DMG_STAGING/README.txt"

BACKGROUND_SVG="$MACOS_PACKAGING_ROOT/create-dmg-background.svg"
if [[ -f "$BACKGROUND_SVG" ]]; then
  mkdir -p "$DMG_STAGING/.background"
  cp "$BACKGROUND_SVG" "$DMG_STAGING/.background/background.svg"
fi

MANIFEST_PATH="$STAGING_ROOT/MANIFEST.txt"
RELEASE_MANIFEST_PATH="$DIST_ROOT/${TARGET_NAME}.MANIFEST.txt"
cat > "$MANIFEST_PATH" <<EOF
NexTunnel desktop installer
Version: $VERSION
ApplicationVersion: $NORMALIZED_VERSION
Target: $PLATFORM
Installer: dmg
Binary: NexTunnel.app
Wintun: skipped; macOS uses utun
Signing: $SIGNING_STATE
PrunedResources: true
EOF
cp "$MANIFEST_PATH" "$DMG_STAGING/MANIFEST.txt"
cp "$MANIFEST_PATH" "$RELEASE_MANIFEST_PATH"

hdiutil create \
  -volname "NexTunnel ${VERSION}" \
  -srcfolder "$DMG_STAGING" \
  -ov \
  -format UDZO \
  "$DMG_PATH"

if [[ "$NOTARIZE" == "true" ]]; then
  required_vars=(MACOS_NOTARY_APPLE_ID MACOS_NOTARY_TEAM_ID MACOS_NOTARY_PASSWORD)
  for var_name in "${required_vars[@]}"; do
    if [[ -z "${!var_name:-}" ]]; then
      echo "启用 notarization 需要设置 $var_name。" >&2
      exit 1
    fi
  done
  xcrun notarytool submit "$DMG_PATH" \
    --apple-id "$MACOS_NOTARY_APPLE_ID" \
    --team-id "$MACOS_NOTARY_TEAM_ID" \
    --password "$MACOS_NOTARY_PASSWORD" \
    --wait
  xcrun stapler staple "$DMG_PATH"
  SIGNING_STATE="notarized"
  sed -i '' "s/^Signing:.*/Signing: $SIGNING_STATE/" "$MANIFEST_PATH"
  cp "$MANIFEST_PATH" "$DMG_STAGING/MANIFEST.txt"
  cp "$MANIFEST_PATH" "$RELEASE_MANIFEST_PATH"
fi

shasum -a 256 "$DMG_PATH" | awk '{print tolower($1) "  " $2}' > "$DMG_PATH.sha256"

echo "macOS DMG 已生成：$DMG_PATH"
echo "SHA256：$DMG_PATH.sha256"
