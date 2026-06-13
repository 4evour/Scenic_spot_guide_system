#!/usr/bin/env bash
# Git 历史凭据清理脚本
# 使用 git-filter-repo 移除已泄露的 API Key 和 JWT Secret
# 前提: pip install git-filter-repo

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
EXPRESSIONS_FILE="$PROJECT_ROOT/.git-filter-expressions.txt"

cat > "$EXPRESSIONS_FILE" << 'EOF'
sk-496797b87b54458dabe8c5cb25bf1a3d==>REDACTED
a3f8b2c1d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1==>REDACTED
EOF

echo "=== Git History Credential Cleanup ==="
echo "Expressions file: $EXPRESSIONS_FILE"
echo ""
echo "This will rewrite git history. All team members will need to re-clone."
read -p "Continue? (y/N) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 0
fi

cd "$PROJECT_ROOT"
git filter-repo --replace-text "$EXPRESSIONS_FILE" --force

echo ""
echo "=== Verification ==="
echo "Checking for remaining secrets..."
if git log -p | grep -q "sk-496797"; then
    echo "WARNING: sk- pattern still found in history!"
else
    echo "OK: No sk- patterns found."
fi
if git log -p | grep -q "a3f8b2c1d4e5f6a7b8c9d0e1"; then
    echo "WARNING: JWT secret pattern still found in history!"
else
    echo "OK: No JWT secret patterns found."
fi

echo ""
echo "Done. Force-push to remote: git push --force --all && git push --force --tags"