#!/bin/bash
# Verify CI/CD setup is complete

set -e

echo "=========================================="
echo "CI/CD Setup Verification"
echo "=========================================="
echo ""

ERRORS=0
WARNINGS=0

# Check workflow files
echo "📋 Checking workflow files..."
WORKFLOWS=(
    ".github/workflows/build-multicdn.yml"
    ".github/workflows/release-multicdn.yml"
    ".github/workflows/docker-multicdn.yml"
    ".github/workflows/test-multicdn.yml"
)

for WORKFLOW in "${WORKFLOWS[@]}"; do
    if [[ -f "$WORKFLOW" ]]; then
        echo "  ✅ $WORKFLOW"
    else
        echo "  ❌ Missing: $WORKFLOW"
        ((ERRORS++))
    fi
done
echo ""

# Check Docker files
echo "🐳 Checking Docker files..."
DOCKER_FILES=(
    ".github/docker/Dockerfile.multicdn"
    ".github/docker/docker-compose.yml"
    ".github/docker/config-multicdn-sample.json"
    ".dockerignore"
)

for FILE in "${DOCKER_FILES[@]}"; do
    if [[ -f "$FILE" ]]; then
        echo "  ✅ $FILE"
    else
        echo "  ❌ Missing: $FILE"
        ((ERRORS++))
    fi
done
echo ""

# Check scripts
echo "📜 Checking scripts..."
SCRIPTS=(
    ".github/scripts/quickstart.sh"
    ".github/scripts/quickstart.bat"
    ".github/scripts/build-local.sh"
    ".github/scripts/build-all.sh"
    ".github/scripts/validate-config.sh"
    ".github/scripts/setup.sh"
)

for SCRIPT in "${SCRIPTS[@]}"; do
    if [[ -f "$SCRIPT" ]]; then
        if [[ "$SCRIPT" == *.sh ]] && [[ ! -x "$SCRIPT" ]]; then
            echo "  ⚠️  $SCRIPT (not executable)"
            ((WARNINGS++))
        else
            echo "  ✅ $SCRIPT"
        fi
    else
        echo "  ❌ Missing: $SCRIPT"
        ((ERRORS++))
    fi
done
echo ""

# Check documentation
echo "📚 Checking documentation..."
DOCS=(
    ".github/README.md"
    ".github/CI_CD_GUIDE.md"
    ".github/MULTICDN_CHEATSHEET.md"
    ".github/SETUP_COMPLETE.md"
)

for DOC in "${DOCS[@]}"; do
    if [[ -f "$DOC" ]]; then
        echo "  ✅ $DOC"
    else
        echo "  ❌ Missing: $DOC"
        ((ERRORS++))
    fi
done
echo ""

# Check YAML syntax (if yamllint available)
if command -v yamllint &> /dev/null; then
    echo "🔍 Validating YAML syntax..."
    for WORKFLOW in "${WORKFLOWS[@]}"; do
        if yamllint "$WORKFLOW" &> /dev/null; then
            echo "  ✅ $WORKFLOW syntax valid"
        else
            echo "  ⚠️  $WORKFLOW has warnings"
            ((WARNINGS++))
        fi
    done
    echo ""
else
    echo "ℹ️  yamllint not installed (skipping YAML validation)"
    echo ""
fi

# Check JSON syntax (if jq available)
if command -v jq &> /dev/null; then
    echo "🔍 Validating JSON syntax..."
    JSON_FILES=(
        ".github/docker/config-multicdn-sample.json"
        ".github/build/friendly-filenames.json"
    )
    
    for JSON in "${JSON_FILES[@]}"; do
        if [[ -f "$JSON" ]]; then
            if jq empty "$JSON" 2>/dev/null; then
                echo "  ✅ $JSON syntax valid"
            else
                echo "  ❌ $JSON has errors"
                ((ERRORS++))
            fi
        fi
    done
    echo ""
else
    echo "ℹ️  jq not installed (skipping JSON validation)"
    echo ""
fi

# Check project structure
echo "📁 Checking project structure..."
if [[ -f "go.mod" ]]; then
    echo "  ✅ go.mod found"
else
    echo "  ❌ go.mod not found"
    ((ERRORS++))
fi

if [[ -d "main" ]] || [[ -f "main/main.go" ]]; then
    echo "  ✅ main package found"
else
    echo "  ⚠️  main package not found"
    ((WARNINGS++))
fi
echo ""

# Summary
echo "=========================================="
echo "Verification Summary"
echo "=========================================="
echo ""
echo "Errors: $ERRORS"
echo "Warnings: $WARNINGS"
echo ""

if [[ $ERRORS -eq 0 ]]; then
    echo "✅ CI/CD setup verification passed!"
    echo ""
    echo "All required files are present."
    if [[ $WARNINGS -gt 0 ]]; then
        echo "⚠️  Some warnings detected (see above)"
    fi
    echo ""
    echo "Ready to:"
    echo "  1. Push code to trigger builds"
    echo "  2. Create release tags"
    echo "  3. Build Docker images"
    echo ""
    exit 0
else
    echo "❌ Verification failed!"
    echo ""
    echo "Please fix the errors above before proceeding."
    echo ""
    exit 1
fi
