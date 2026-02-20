#!/bin/bash

# 通过 SMB 挂载 NAS，本地下载字幕后复制到 qb 目录
# 用途：挂载 NAS 下载目录，本地处理字幕后直接复制

set -e

# NAS SMB 配置
NAS_HOST="${NAS_HOST:-192.168.1.246}"
NAS_SHARE="${NAS_SHARE:-downloads}"
NAS_USER="${NAS_USER:-roger}"
NAS_PASS="${NAS_PASS:-}""
MOUNT_POINT="${MOUNT_POINT:-/mnt/nas-downloads}"

# 帮助信息
usage() {
    echo "用法: download-subtitle-smb.sh [选项]"
    echo ""
    echo "选项："
    echo "  -h, --host         NAS IP (默认: $NAS_HOST)"
    echo "  -s, --share        SMB 共享名 (默认: $NAS_SHARE)"
    echo "  -u, --user         NAS 用户名 (默认: $NAS_USER)"
    echo "  -p, --pass         NAS 密码"
    echo "  -m, --mount        本地挂载点 (默认: $MOUNT_POINT)"
    echo "  -t, --torrent-name 要处理的种子名称关键词"
    echo "  -l, --languages    字幕语言 (默认: zh,en)"
    echo "  --help             显示帮助"
    echo ""
    echo "示例："
    echo "  download-subtitle-smb.sh -t \"Lilo Stitch\" -l zh,en"
    echo "  download-subtitle-smb.sh -t \"Young Sheldon\" -p \"mypassword\""
    exit 1
}

# 解析参数
TORRENT_NAME=""
LANGUAGES="zh,en"

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--host)
            NAS_HOST="$2"
            shift 2
            ;;
        -s|--share)
            NAS_SHARE="$2"
            shift 2
            ;;
        -u|--user)
            NAS_USER="$2"
            shift 2
            ;;
        -p|--pass)
            NAS_PASS="$2"
            shift 2
            ;;
        -m|--mount)
            MOUNT_POINT="$2"
            shift 2
            ;;
        -t|--torrent-name)
            TORRENT_NAME="$2"
            shift 2
            ;;
        -l|--languages)
            LANGUAGES="$2"
            shift 2
            ;;
        --help)
            usage
            ;;
        *)
            echo "未知参数: $1"
            usage
            ;;
    esac
done

if [[ -z "$TORRENT_NAME" ]]; then
    echo "❌ 错误：需要指定种子名称关键词 (-t)"
    usage
fi

echo "=== SMB 字幕下载工具 ==="
echo "NAS: //$NAS_HOST/$NAS_SHARE"
echo "挂载点: $MOUNT_POINT"
echo "关键词: $TORRENT_NAME"
echo "语言: $LANGUAGES"
echo ""

# 检查 subliminal
if ! command -v subliminal >/dev/null 2>&1; then
    echo "❌ subliminal 未安装"
    echo "请先安装: pip3 install --user subliminal"
    exit 1
fi

# 检查 cifs-utils
if ! command -v mount.cifs >/dev/null 2>&1; then
    echo "❌ cifs-utils 未安装"
    echo "请安装: sudo apt-get install cifs-utils"
    exit 1
fi

# 步骤 1: 创建挂载点
echo "📂 步骤 1: 创建挂载点..."
sudo mkdir -p "$MOUNT_POINT"

# 步骤 2: 挂载 SMB 共享
echo "🔗 步骤 2: 挂载 SMB 共享..."

if [[ -n "$NAS_PASS" ]]; then
    # 使用密码挂载
    sudo mount -t cifs "//$NAS_HOST/$NAS_SHARE" "$MOUNT_POINT" \
        -o username="$NAS_USER",password="$NAS_PASS",uid=$(id -u),gid=$(id -g) 2>&1
else
    # 尝试免密挂载（如果 NAS 允许 guest）
    sudo mount -t cifs "//$NAS_HOST/$NAS_SHARE" "$MOUNT_POINT" \
        -o guest,uid=$(id -u),gid=$(id -g) 2>&1 || {
        echo "❌ 挂载失败，可能需要密码"
        echo "请使用 -p 参数提供密码"
        exit 1
    }
fi

if [[ ! -d "$MOUNT_POINT" ]] || [[ -z "$(ls -A "$MOUNT_POINT" 2>/dev/null)" ]]; then
    echo "❌ 挂载失败或目录为空"
    exit 1
fi

echo "✅ 挂载成功"
echo ""

# 清理函数（退出时卸载）
cleanup() {
    echo ""
    echo "🧹 清理：卸载 SMB 共享..."
    sudo umount "$MOUNT_POINT" 2>/dev/null || true
    echo "✅ 已卸载"
}
trap cleanup EXIT

# 步骤 3: 查找视频文件
echo "🔍 步骤 3: 查找视频文件..."

VIDEO_FILES=$(find "$MOUNT_POINT" -type f \( -name '*.mp4' -o -name '*.mkv' -o -name '*.avi' \) 2>/dev/null | grep -i "$TORRENT_NAME" || true)

if [[ -z "$VIDEO_FILES" ]]; then
    echo "❌ 未找到匹配的视频文件"
    exit 1
fi

echo "✅ 找到以下视频文件："
echo "$VIDEO_FILES"
echo ""

# 步骤 4: 为每个视频下载字幕
echo "📥 步骤 4: 下载字幕..."

TEMP_DIR=$(mktemp -d)
cd "$TEMP_DIR"

for video_path in $VIDEO_FILES; do
    video_name=$(basename "$video_path")
    video_dir=$(dirname "$video_path")
    
    echo ""
    echo "处理: $video_name"
    
    # 检查是否已有字幕
    existing_subs=$(find "$video_dir" -name "${video_name%.*}.*.srt" -o -name "${video_name%.*}.*.ass" 2>/dev/null | head -1)
    if [[ -n "$existing_subs" ]]; then
        echo "  ⏭️  已有字幕，跳过"
        continue
    fi
    
    # 创建本地占位文件
    touch "$video_name"
    
    # 下载字幕
    echo "  下载字幕..."
    subliminal download -l zho -l eng "$video_name" 2>&1 || echo "  ⚠️  字幕下载可能失败"
    
    # 查找下载的字幕
    for sub in *.srt *.ass 2>/dev/null; do
        if [[ -f "$sub" ]]; then
            # 检查字幕是否匹配当前视频
            if [[ "$sub" == ${video_name%.*}* ]]; then
                echo "  ✅ 找到字幕: $sub"
                echo "  📤 复制到: $video_dir/"
                cp "$sub" "$video_dir/" && echo "  ✅ 复制成功" || echo "  ❌ 复制失败"
                rm "$sub"
            fi
        fi
    done
    
    rm "$video_name"
done

cd >/dev/null
rm -rf "$TEMP_DIR"

echo ""
echo "✅ 全部完成！"
echo "字幕已复制到 NAS 的 qb 下载目录"

# 清理会自动执行（trap EXIT）
