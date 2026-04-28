#!/bin/bash

# 编译脚本
# 功能：构建前端和后端，支持多平台编译，注入版本信息
# 使用方法: ./scripts/build.sh [版本号] [平台]
# 示例: ./scripts/build.sh v0.5.0 linux
# 平台选项: linux, linux-sqlite, windows, mac, all

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

# 获取版本号（从参数或 git tag）
VERSION=${1:-$(git describe --tags --always 2>/dev/null || echo "v0.4.9")}
PLATFORM=${2:-linux}

# 验证版本号格式
if [[ ! $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
    echo -e "${YELLOW}警告: 版本号格式可能不正确: $VERSION${NC}"
fi

# 获取构建信息
BUILD_TIME=$(date +"%Y-%m-%d %H:%M:%S")
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo -e "${GREEN}开始构建...${NC}"
echo "版本号: $VERSION"
echo "构建时间: $BUILD_TIME"
echo "Git提交: $GIT_COMMIT"
echo "目标平台: $PLATFORM"
echo ""

# 构建前端
echo -e "${GREEN}步骤 1/3: 构建前端...${NC}"
cd "$PROJECT_ROOT/frontend"

if [ ! -f "package.json" ]; then
    echo -e "${RED}错误: 未找到 frontend/package.json${NC}"
    exit 1
fi

# 检查是否已安装依赖
if [ ! -d "node_modules" ]; then
    echo "安装前端依赖..."
    if command -v yarn &> /dev/null; then
        yarn install
    elif command -v npm &> /dev/null; then
        npm install
    else
        echo -e "${RED}错误: 未找到 yarn 或 npm${NC}"
        exit 1
    fi
fi

# 构建前端
if command -v yarn &> /dev/null; then
    yarn build
else
    npm run build
fi

if [ ! -d "dist" ] || [ ! -f "dist/index.html" ]; then
    echo -e "${RED}错误: 前端构建失败，未找到 dist/index.html${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 前端构建完成${NC}"
echo ""

# 复制前端文件到 embed 目录
echo -e "${GREEN}步骤 2/3: 复制前端文件到 embed 目录...${NC}"
cd "$PROJECT_ROOT/backend"
rm -rf cmd/server/frontend-dist
cp -r ../frontend/dist cmd/server/frontend-dist
echo -e "${GREEN}✓ 前端文件复制完成${NC}"
echo ""

# 构建后端
echo -e "${GREEN}步骤 3/3: 构建后端...${NC}"

# 创建输出目录
OUTPUT_DIR="$PROJECT_ROOT/releases/$VERSION"
mkdir -p "$OUTPUT_DIR"

# 构建函数
build_platform() {
    local platform=$1
    local goos=$2
    local goarch=$3
    local cgo_enabled=$4
    local output_name=$5
    local desc=$6

    echo -e "${GREEN}构建 $desc...${NC}"
    
    export CGO_ENABLED=$cgo_enabled
    export GOOS=$goos
    export GOARCH=$goarch
    export GOPROXY=https://goproxy.cn

    # 构建参数
    LDFLAGS="-s -w"
    LDFLAGS="$LDFLAGS -X main.Version=$VERSION"
    LDFLAGS="$LDFLAGS -X 'main.BuildTime=$BUILD_TIME'"
    LDFLAGS="$LDFLAGS -X main.GitCommit=$GIT_COMMIT"

    # 编译
    go build -ldflags="$LDFLAGS" -o "$OUTPUT_DIR/$output_name" cmd/server/main.go

    if [ $? -eq 0 ]; then
        local file_size=$(du -h "$OUTPUT_DIR/$output_name" | cut -f1)
        echo -e "${GREEN}✓ $desc 构建成功: $output_name ($file_size)${NC}"
    else
        echo -e "${RED}✗ $desc 构建失败${NC}"
        return 1
    fi
}

# 根据平台构建
case $PLATFORM in
    linux)
        build_platform linux linux amd64 0 "prjflow-linux-amd64" "Linux (静态编译, 仅支持MySQL)"
        ;;
    linux-sqlite)
        build_platform linux-sqlite linux amd64 1 "prjflow-linux-amd64-sqlite" "Linux (支持SQLite, 需要CGO)"
        ;;
    windows)
        build_platform windows windows amd64 0 "prjflow-windows-amd64.exe" "Windows"
        ;;
    mac)
        build_platform mac darwin amd64 0 "prjflow-darwin-amd64" "macOS (Intel)"
        ;;
    mac-arm)
        build_platform mac-arm darwin arm64 0 "prjflow-darwin-arm64" "macOS (Apple Silicon)"
        ;;
    all)
        build_platform linux linux amd64 0 "prjflow-linux-amd64" "Linux (静态编译)"
        build_platform windows windows amd64 0 "prjflow-windows-amd64.exe" "Windows"
        build_platform mac darwin amd64 0 "prjflow-darwin-amd64" "macOS (Intel)"
        build_platform mac-arm darwin arm64 0 "prjflow-darwin-arm64" "macOS (Apple Silicon)"
        ;;
    *)
        echo -e "${RED}错误: 不支持的平台: $PLATFORM${NC}"
        echo "支持的平台: linux, linux-sqlite, windows, mac, mac-arm, all"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}构建完成！${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}"
echo "输出目录: $OUTPUT_DIR"
echo "版本号: $VERSION"
echo "构建时间: $BUILD_TIME"
echo "Git提交: $GIT_COMMIT"
echo ""
echo "文件列表:"
ls -lh "$OUTPUT_DIR" | tail -n +2 | awk '{print "  " $9 " (" $5 ")"}'
echo ""
echo "提示："
echo "  - 静态编译版本（CGO_ENABLED=0）不依赖系统库，可在任何Linux服务器运行"
echo "  - SQLite版本（linux-sqlite）需要CGO，需要服务器GLIBC版本匹配"
echo "  - 记得同时上传 config.yaml 和数据库文件（如果使用SQLite）"
echo ""

