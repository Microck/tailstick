#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
icon_dir="${root_dir}/assets/icon"

icon_png="${icon_dir}/tailstick-logo.png"
icon_ico="${icon_dir}/tailstick-logo.ico"
icon_rc="${icon_dir}/tailstick-icon.rc"

if [[ ! -f "${icon_png}" ]]; then
  echo "missing icon source: ${icon_png}" >&2
  exit 1
fi

if ! command -v convert >/dev/null 2>&1; then
  echo "missing dependency: ImageMagick 'convert'" >&2
  exit 1
fi

if ! command -v x86_64-w64-mingw32-windres >/dev/null 2>&1; then
  echo "missing dependency: x86_64-w64-mingw32-windres" >&2
  exit 1
fi

convert "${icon_png}" -background none -define icon:auto-resize=256,128,96,64,48,32,16 "${icon_ico}"

cat >"${icon_rc}" <<'EOF'
1 ICON "tailstick-logo.ico"
EOF

(
  cd "${icon_dir}"
  x86_64-w64-mingw32-windres "${icon_rc##*/}" -O coff -o "${root_dir}/cmd/tailstick-windows-cli/resource_windows_amd64.syso"
  x86_64-w64-mingw32-windres "${icon_rc##*/}" -O coff -o "${root_dir}/cmd/tailstick-windows-gui/resource_windows_amd64.syso"
)

echo "generated:"
echo "  ${icon_ico}"
echo "  ${root_dir}/cmd/tailstick-windows-cli/resource_windows_amd64.syso"
echo "  ${root_dir}/cmd/tailstick-windows-gui/resource_windows_amd64.syso"
