#!/bin/bash
# 从 Git 历史中移除敏感信息的脚本

set -e

echo "开始从 Git 历史中移除敏感信息..."

# 备份当前分支
echo "创建备份分支..."
git branch backup-before-filter-$(date +%Y%m%d-%H%M%S)

# 使用 git filter-branch 替换历史中的敏感信息
git filter-branch --force --index-filter \
  'git rm --cached --ignore-unmatch transformer-vnet/docs/文生图/doubao-seedream-4.5.md || true' \
  --prune-empty --tag-name-filter cat -- --all

# 重新添加修复后的文件
git checkout HEAD -- transformer-vnet/docs/文生图/doubao-seedream-4.5.md
git add transformer-vnet/docs/文生图/doubao-seedream-4.5.md
git commit --amend --no-edit || true

echo ""
echo "完成！现在可以强制推送："
echo "  git push origin --force --all"
echo ""
echo "⚠️  警告：这会重写 Git 历史，如果其他人也在使用这个仓库，需要通知他们重新克隆。"
