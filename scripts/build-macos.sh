#!/bin/bash
# macOS GUI 应用构建脚本
# 在 M1 Mac 上运行，生成 ImgBed-macos-arm64.zip

set -e

VERSION=${1:-"v1.0.1"}
APP_NAME="ImgBed"
ARCH="arm64"
OUTPUT="ImgBed-macos-${ARCH}.zip"

echo "=== 构建 macOS GUI 应用 ==="
echo "版本: $VERSION"
echo "架构: $ARCH"

# 1. 构建 Go GUI 二进制
echo ""
echo "[1/4] 编译 Go GUI 二进制..."
cd server
CGO_ENABLED=1 GOOS=darwin GOARCH=${ARCH} go build -tags "gui sqlite_fts5" -ldflags="-s -w" -o imgbed-macos
cd ..

# 2. 创建 .app 目录结构
echo ""
echo "[2/4] 创建 .app 目录结构..."
rm -rf "${APP_NAME}.app"
mkdir -p "${APP_NAME}.app/Contents/MacOS"
mkdir -p "${APP_NAME}.app/Contents/Resources"

# 3. 复制二进制并设置可执行权限
echo ""
echo "[3/4] 配置应用包..."
cp server/imgbed-macos "${APP_NAME}.app/Contents/MacOS/${APP_NAME}"
chmod +x "${APP_NAME}.app/Contents/MacOS/${APP_NAME}"

# 4. 复制图标（如果有）
if [ -f "systray/icon.png" ]; then
    # 转换 PNG 为 ICNS（需要 sips 和 iconutil，macOS 自带）
    # 先转 icns
    sips -s format icns systray/icon.png --out "${APP_NAME}.app/Contents/Resources/${APP_NAME}.icns" 2>/dev/null || true
fi

# 5. 创建 Info.plist
echo ""
echo "[4/4] 创建 Info.plist..."
cat > "${APP_NAME}.app/Contents/Info.plist" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>ImgBed</string>
    <key>CFBundleIdentifier</key>
    <string>com.imgbed.macos</string>
    <key>CFBundleName</key>
    <string>ImgBed</string>
    <key>CFBundleDisplayName</key>
    <string>ImgBed</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>LSMinimumSystemVersion</key>
    <string>11.0</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF

# 6. 打包成 zip
echo ""
echo "打包成 zip..."
rm -f "$OUTPUT"
zip -r "$OUTPUT" "${APP_NAME}.app"

echo ""
echo "=== 构建完成 ==="
echo "输出文件: $OUTPUT"
echo "大小: $(du -h "$OUTPUT" | cut -f1)"
echo ""
echo "上传到 GitHub Release:"
echo "  gh release upload $VERSION $OUTPUT --clobber"
