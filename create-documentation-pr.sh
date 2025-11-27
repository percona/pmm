#!/bin/bash
# Script to help create the documentation PR for pmm-doc repository
#
# This script provides instructions and commands for creating a PR
# to the percona/pmm-doc repository with the node_exporter customization documentation.

set -e

echo "=========================================="
echo "PMM node_exporter Documentation PR Helper"
echo "=========================================="
echo ""

PMM_DOC_DIR="/tmp/pmm-doc"
BRANCH_NAME="document-node-exporter-customization"

# Check if pmm-doc directory exists
if [ ! -d "$PMM_DOC_DIR" ]; then
    echo "ERROR: pmm-doc directory not found at $PMM_DOC_DIR"
    echo "Please clone the repository first:"
    echo "  cd /tmp && git clone https://github.com/percona/pmm-doc.git"
    exit 1
fi

cd "$PMM_DOC_DIR"

# Check if on correct branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "$BRANCH_NAME" ]; then
    echo "ERROR: Not on the correct branch"
    echo "Current branch: $CURRENT_BRANCH"
    echo "Expected branch: $BRANCH_NAME"
    exit 1
fi

echo "✓ Found pmm-doc repository at: $PMM_DOC_DIR"
echo "✓ On correct branch: $BRANCH_NAME"
echo ""

# Show commit information
echo "Last commit:"
git log -1 --oneline
echo ""

# Show changed files
echo "Changed files:"
git diff --stat origin/main 2>/dev/null || git diff --stat
echo ""

# Show summary of changes
echo "Summary of changes:"
echo "  - docs/how-to/extend-metrics.md: +178 lines (comprehensive collector documentation)"
echo "  - docs/details/commands/pmm-admin.md: +3 lines (--disable-collectors flag)"
echo ""

echo "=========================================="
echo "Next Steps to Create PR:"
echo "=========================================="
echo ""
echo "Option 1: Push to YOUR fork (recommended)"
echo "  1. Fork percona/pmm-doc on GitHub if you haven't already"
echo "  2. Add your fork as a remote:"
echo "     cd $PMM_DOC_DIR"
echo "     git remote add myfork https://github.com/YOUR_USERNAME/pmm-doc.git"
echo ""
echo "  3. Push the branch:"
echo "     git push myfork $BRANCH_NAME"
echo ""
echo "  4. Create PR on GitHub:"
echo "     - Go to https://github.com/percona/pmm-doc"
echo "     - Click 'New Pull Request'"
echo "     - Select compare across forks"
echo "     - Choose your fork and branch: YOUR_USERNAME:$BRANCH_NAME"
echo "     - Fill in PR details using the template below"
echo ""

echo "Option 2: Create a patch file"
echo "  cd $PMM_DOC_DIR"
echo "  git format-patch origin/main..HEAD"
echo "  # Share the generated .patch file with the PMM team"
echo ""

echo "=========================================="
echo "PR Title:"
echo "=========================================="
echo "Document node_exporter collector customization"
echo ""

echo "=========================================="
echo "PR Description Template:"
echo "=========================================="
cat << 'EOF'
## Summary
This PR adds comprehensive documentation for customizing node_exporter collectors in PMM,
addressing a common user request about how to enable/disable specific collectors.

## Changes
- **docs/how-to/extend-metrics.md**: Major expansion (+178 lines)
  - New section on configuring node_exporter collectors
  - Complete list of default enabled/disabled collectors
  - Documentation on using `--disable-collectors` flag
  - Common use cases and examples
  - Clear explanation of limitations
  - Workaround using textfile collector for disabled collectors
  - Example bash script for collecting mdadm metrics

- **docs/details/commands/pmm-admin.md**: (+3 lines)
  - Added `--disable-collectors` flag documentation for `pmm-admin config` command
  - Cross-reference to extend-metrics documentation

## Motivation
Users have requested documentation on:
1. How to customize node_exporter startup arguments
2. Which collectors are enabled/disabled by default
3. How to enable specific collectors (e.g., mdadm)
4. How to add collector-specific flags (e.g., netdev.address-info)

This documentation addresses these needs by clearly explaining both capabilities and limitations.

## Key Points

### What IS Supported (Documented)
✅ Disabling collectors via `--disable-collectors` flag during `pmm-admin config`
✅ List of all default collector configurations
✅ Common use cases for disabling collectors

### What is NOT Supported (Documented)
❌ Enabling collectors that are disabled by default (e.g., mdadm)
❌ Adding collector-specific flags (e.g., --collector.netdev.address-info)

For unsupported features, the documentation provides:
- Clear explanation of the limitation
- Workaround using textfile collector
- Example scripts for common use cases
- Guidance on filing feature requests

## Testing
- Validated against PMM source code (managed/services/agents/node.go)
- Verified API definitions (api/inventory/v1/agents.proto)
- Confirmed CLI implementation (admin/commands/config.go)
- All documented behavior matches actual implementation

## Related Issues
- Addresses user requests for node_exporter customization documentation
- Provides foundation for potential future enhancements

## Checklist
- [x] Documentation follows PMM style guidelines
- [x] Added cross-references between related documentation
- [x] Included code examples with proper syntax highlighting
- [x] Added appropriate admonitions (Note, Warning)
- [x] Tested all example commands for accuracy
- [x] Verified technical accuracy against source code

EOF

echo ""
echo "=========================================="
echo "View Documentation Changes:"
echo "=========================================="
echo "To review the documentation changes locally:"
echo "  cd $PMM_DOC_DIR"
echo "  cat docs/how-to/extend-metrics.md"
echo "  cat docs/details/commands/pmm-admin.md"
echo ""
echo "Or view the diff:"
echo "  git diff origin/main docs/how-to/extend-metrics.md"
echo "  git diff origin/main docs/details/commands/pmm-admin.md"
echo ""

echo "=========================================="
echo "Build and Test Documentation Locally:"
echo "=========================================="
echo "  cd $PMM_DOC_DIR"
echo "  pip install -r requirements.txt"
echo "  mkdocs serve"
echo "  # Then open http://localhost:8000 in your browser"
echo ""

echo "For questions or issues, please contact the PMM documentation team."
