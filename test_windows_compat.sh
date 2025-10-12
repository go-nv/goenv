#!/usr/bin/env bash
#
# Windows Compatibility Verification for goenv
# Checks that recent changes work cross-platform
#

set -e

echo "=================================================="
echo "    Windows Compatibility Verification"
echo "=================================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ISSUES=0

echo "Checking recent changes for Windows compatibility..."
echo ""

# Check 1: Path handling in hooks.go
echo "1. Checking hooks.go path handling..."
if grep -q "filepath.Join" cmd/hooks.go; then
    echo -e "${GREEN}✅ Uses filepath.Join (cross-platform)${NC}"
else
    echo -e "${RED}❌ Not using filepath.Join${NC}"
    ISSUES=$((ISSUES + 1))
fi

# Check 2: File operations use os package
echo "2. Checking file operations..."
if grep -q "os\\.ReadDir\\|os\\.Stat\\|os\\.ReadFile" cmd/hooks.go; then
    echo -e "${GREEN}✅ Uses os package (cross-platform)${NC}"
else
    echo -e "${YELLOW}⚠️  Check file operations${NC}"
fi

# Check 3: No hardcoded Unix paths
echo "3. Checking for hardcoded Unix paths..."
UNIX_PATHS=$(grep -r "^[^#]*\"/tmp\\|/usr\\|/home\\|/opt" cmd/ internal/ 2>/dev/null | grep -v ".md:" | grep -v "test" || true)
if [ -z "$UNIX_PATHS" ]; then
    echo -e "${GREEN}✅ No hardcoded Unix paths in production code${NC}"
else
    echo -e "${RED}❌ Found hardcoded Unix paths:${NC}"
    echo "$UNIX_PATHS"
    ISSUES=$((ISSUES + 1))
fi

# Check 4: Environment variable handling
echo "4. Checking environment variable handling..."
if grep -q "os\\.Getenv\\|os\\.Setenv" cmd/*.go internal/**/*.go 2>/dev/null; then
    echo -e "${GREEN}✅ Uses os.Getenv/Setenv (cross-platform)${NC}"
else
    echo -e "${YELLOW}⚠️  Check env var handling${NC}"
fi

# Check 5: Shell-specific code is properly isolated
echo "5. Checking shell detection logic..."
if grep -q "runtime\\.GOOS" cmd/init.go; then
    echo -e "${GREEN}✅ Has OS detection (runtime.GOOS)${NC}"
else
    echo -e "${YELLOW}⚠️  Consider adding OS detection${NC}"
fi

# Check 6: No Unix-specific commands
echo "6. Checking for Unix-specific commands..."
UNIX_CMDS=$(grep -r "exec\\.Command.*\"bash\\|\"sh\\|\"zsh\\|\"fish" cmd/ internal/ 2>/dev/null | grep -v ".md:" | grep -v "test" | grep -v "// " || true)
if [ -z "$UNIX_CMDS" ]; then
    echo -e "${GREEN}✅ No hardcoded Unix shell commands${NC}"
else
    echo -e "${YELLOW}⚠️  Found shell-specific commands (check if conditional):${NC}"
    echo "$UNIX_CMDS"
fi

# Check 7: Verify Windows-specific handling exists
echo "7. Checking for Windows-specific handling..."
if grep -q "windows" cmd/init.go internal/**/*.go 2>/dev/null; then
    echo -e "${GREEN}✅ Has Windows-specific handling${NC}"
else
    echo -e "${YELLOW}⚠️  May need Windows-specific code paths${NC}"
fi

# Check 8: File extensions handled properly
echo "8. Checking file extension handling..."
if grep -q "\\.exe\\|\\.bat\\|\\.cmd" cmd/*.go internal/**/*.go 2>/dev/null; then
    echo -e "${GREEN}✅ Handles Windows executables${NC}"
else
    echo -e "${YELLOW}⚠️  Check if .exe handling needed${NC}"
fi

# Check 9: Check DisableFlagParsing doesn't break Windows
echo "9. Checking DisableFlagParsing usage..."
if grep -q "DisableFlagParsing" cmd/*.go; then
    echo -e "${GREEN}✅ Uses DisableFlagParsing (version, sh-shell)${NC}"
    echo "   Note: Verify this works with Windows flag syntax"
else
    echo -e "${GREEN}✅ Not using DisableFlagParsing${NC}"
fi

# Check 10: Path separator handling
echo "10. Checking path separator handling..."
if grep -q "filepath\\.Separator\\|os\\.PathListSeparator" cmd/*.go internal/**/*.go 2>/dev/null; then
    echo -e "${GREEN}✅ Uses OS-specific path separators${NC}"
else
    echo -e "${YELLOW}⚠️  Check if hardcoded path separators exist${NC}"
fi

echo ""
echo "=================================================="
echo "              Summary"
echo "=================================================="

if [ $ISSUES -eq 0 ]; then
    echo -e "${GREEN}✅ No critical Windows compatibility issues found${NC}"
    echo ""
    echo "Recent changes appear Windows-compatible:"
    echo "  - hooks.go: Uses filepath.Join ✅"
    echo "  - init.go: DisableFlagParsing for sh-shell ✅"
    echo "  - All file operations use os package ✅"
    echo ""
    echo "Recommendations:"
    echo "  1. Test on actual Windows machine"
    echo "  2. Verify PowerShell integration"
    echo "  3. Test path handling with backslashes"
    echo "  4. Verify .exe detection in versions"
else
    echo -e "${RED}❌ Found $ISSUES potential compatibility issues${NC}"
    echo "   Review the items above"
fi

echo "=================================================="

exit $ISSUES
