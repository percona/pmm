# PMM node_exporter Collector Customization - Documentation PR Ready

## Quick Summary

✅ **Investigation Complete**  
✅ **Documentation Created**  
✅ **Ready for PR to pmm-doc Repository**

This investigation provides comprehensive documentation for customizing node_exporter collectors in PMM, addressing the specific user request about enabling mdadm collector and netdev.address-info flag.

## What Was Requested

The original request asked for:
1. Investigation of how PMM starts node_exporter
2. Understanding which collectors are enabled/disabled by default  
3. Documentation on how to customize node_exporter arguments
4. Specific examples for mdadm and netdev.address-info collectors

## What Was Delivered

### 1. Complete Code Investigation

**Key Findings:**
- node_exporter configuration is in `managed/services/agents/node.go`
- `--no-collector.mdadm` is disabled by default (line 91)
- PMM supports **disabling collectors** via `--disable-collectors` flag
- PMM does **NOT** support enabling disabled collectors or adding collector-specific flags
- Configuration happens during `pmm-admin config` setup

**See:** `NODE_EXPORTER_CUSTOMIZATION_FINDINGS.md` for detailed technical findings

### 2. Comprehensive Documentation

**Created:**
- Enhanced `docs/how-to/extend-metrics.md` with 178+ new lines
- Updated `docs/details/commands/pmm-admin.md` with flag documentation

**Documentation Includes:**
- Complete list of 30+ default enabled collectors
- Complete list of 26+ default disabled collectors  
- How to use `--disable-collectors` flag
- Common use cases and examples
- Clear explanation of limitations
- Workaround using textfile collector
- Full example script for mdadm metrics collection
- Guidance on filing feature requests

### 3. Ready-to-Use Deliverables

**Files in This Repository:**

1. **`NODE_EXPORTER_CUSTOMIZATION_FINDINGS.md`**
   - Complete technical investigation results
   - Code references and line numbers
   - API and CLI implementation details
   - Recommendations for future enhancements

2. **`create-documentation-pr.sh`** (executable)
   - Interactive helper script
   - Guides through PR creation process
   - Provides PR templates

3. **`0001-Document-node_exporter-collector-customization.patch`**
   - Git patch file with all documentation changes
   - Can be applied directly or shared with team
   - Ready for review

**Documentation Repository:**
- Location: `/tmp/pmm-doc`
- Branch: `document-node-exporter-customization`
- Commit: 2892849c7
- Ready to push and create PR

## Answers to Specific Questions

### Q: How do I enable the mdadm collector?

**A:** You **cannot** enable the mdadm collector through PMM configuration (it's disabled by default and cannot be enabled). 

**Workaround:** Use the textfile collector to export mdadm metrics. Full example script provided in documentation:

```bash
#!/bin/bash
# /usr/local/bin/mdadm_metrics.sh
OUTPUT_DIR="/usr/local/percona/pmm2/collectors/textfile-collector/low-resolution"
OUTPUT_FILE="${OUTPUT_DIR}/mdadm.prom"

# Ensure output directory exists and is writable
if [ ! -d "$OUTPUT_DIR" ]; then
    echo "Error: Directory $OUTPUT_DIR does not exist" >&2
    exit 1
fi

if [ ! -w "$OUTPUT_DIR" ]; then
    echo "Error: Directory $OUTPUT_DIR is not writable" >&2
    exit 1
fi

echo "# HELP node_md_state RAID array state" > "$OUTPUT_FILE"
echo "# TYPE node_md_state gauge" >> "$OUTPUT_FILE"

for array in /dev/md*; do
    if [ -b "$array" ]; then
        array_name=$(basename "$array")
        state=$(cat /sys/block/${array_name}/md/array_state 2>/dev/null || echo "unknown")
        if [ "$state" = "active" ]; then
            echo "node_md_state{device=\"${array_name}\"} 1" >> "$OUTPUT_FILE"
        else
            echo "node_md_state{device=\"${array_name}\"} 0" >> "$OUTPUT_FILE"
        fi
    fi
done
```

Schedule with cron: `*/5 * * * * /usr/local/bin/mdadm_metrics.sh`

See the documentation for complete details.

### Q: How do I enable netdev.address-info?

**A:** You **cannot** add collector-specific flags like `--collector.netdev.address-info` through PMM configuration.

**Recommendation:**
1. Use textfile collector to export network address information
2. File a feature request in [PMM JIRA](https://perconadev.atlassian.net/projects/PMM/) for native support

See documentation for details on filing feature requests.

### Q: How do I disable collectors?

**A:** You **can** disable collectors during initial setup (SUPPORTED):

```bash
pmm-admin config \
    --server-url=https://admin:admin@your-pmm-server:443 \
    --disable-collectors=netdev,netstat,vmstat \
    192.168.1.10 generic node1
```

This is fully documented in both:
- `docs/how-to/extend-metrics.md`
- `docs/details/commands/pmm-admin.md`

### Q: Which collectors are enabled/disabled by default?

**A:** See the comprehensive list in the documentation. Summary:

**Enabled by default:** cpu, diskstats, filefd, filesystem, loadavg, meminfo, meminfo_numa, netdev, netstat, processes, stat, time, vmstat, and more (30+ collectors)

**Disabled by default:** mdadm, arp, bcache, conntrack, drbd, edac, infiniband, interrupts, ipvs, ksmd, logind, mountstats, netclass, nfs, nfsd, ntp, qdisc, systemd, and more (26+ collectors)

Full list with resolution information (HR/MR/LR) is in the documentation.

## How to Create the Documentation PR

### Option 1: Use the Helper Script (Recommended)

```bash
./create-documentation-pr.sh
```

This interactive script will:
- Verify the documentation repository
- Show the changes made
- Provide step-by-step instructions for creating the PR
- Include PR title and description templates

### Option 2: Manual Steps

1. **Navigate to pmm-doc:**
   ```bash
   cd /tmp/pmm-doc
   ```

2. **Review changes:**
   ```bash
   git log -1
   git diff origin/main --stat
   ```

3. **Push to your fork:**
   ```bash
   git remote add myfork https://github.com/YOUR_USERNAME/pmm-doc.git
   git push myfork document-node-exporter-customization
   ```

4. **Create PR on GitHub:**
   - Go to https://github.com/percona/pmm-doc
   - Click "New Pull Request"
   - Compare across forks: `YOUR_USERNAME:document-node-exporter-customization`
   - Use the PR template from `create-documentation-pr.sh`

### Option 3: Share Patch File

The patch file can be shared directly with the PMM documentation team:

```bash
# Patch file location:
/home/runner/work/pmm/pmm/0001-Document-node_exporter-collector-customization.patch
```

Apply with: `git am 0001-Document-node_exporter-collector-customization.patch`

## Validation and Testing

All documentation has been validated against:

✅ **Source Code**
- `managed/services/agents/node.go` - Configuration logic
- `managed/utils/collectors/collectors.go` - Filtering implementation
- `admin/commands/config.go` - CLI flags
- `api/inventory/v1/agents.proto` - API definitions

✅ **Test Cases**
- `managed/services/agents/node_test.go` - Behavior validation

✅ **Documentation Standards**
- MkDocs markdown format
- Code examples with syntax highlighting
- Appropriate admonitions (Note, Warning)
- Cross-references between pages

## Files Summary

| File | Location | Purpose |
|------|----------|---------|
| Enhanced extend-metrics.md | `/tmp/pmm-doc/docs/how-to/` | Main collector customization documentation (+178 lines) |
| Updated pmm-admin.md | `/tmp/pmm-doc/docs/details/commands/` | CLI flag reference (+3 lines) |
| Findings Report | `NODE_EXPORTER_CUSTOMIZATION_FINDINGS.md` | Complete technical investigation |
| PR Helper Script | `create-documentation-pr.sh` | Interactive PR creation guide |
| Patch File | `0001-Document-node_exporter-collector-customization.patch` | Git patch for review |

## Key Takeaways

### What Users CAN Do (Now Documented)

✅ Disable collectors during setup with `--disable-collectors`
✅ View complete list of enabled/disabled collectors  
✅ Understand which collectors run at which resolutions
✅ Use textfile collector for custom metrics
✅ Implement workarounds for disabled collectors

### What Users CANNOT Do (Limitations Documented)

❌ Enable collectors disabled by default (like mdadm)
❌ Add collector-specific flags (like netdev.address-info)
❌ Change collector configuration after initial setup
❌ Configure collectors through PMM UI

### For PMM Development

The documentation provides foundation for potential future enhancements:
- Add `enable_collectors` API parameter
- Support collector-specific options
- Make configuration mutable after setup
- Add collector presets (minimal/standard/comprehensive)

## Next Steps

1. **Review the documentation** in `/tmp/pmm-doc/docs/how-to/extend-metrics.md`
2. **Run the helper script** for PR creation instructions
3. **Create PR** to percona/pmm-doc repository
4. **Share findings** with PMM team for future enhancements

## Questions or Issues?

- Documentation questions: Review `NODE_EXPORTER_CUSTOMIZATION_FINDINGS.md`
- Technical questions: See source code references in findings document
- PR process: Run `./create-documentation-pr.sh` for guidance
- Feature requests: File in [PMM JIRA](https://perconadev.atlassian.net/projects/PMM/)

---

**Documentation Status:** ✅ Complete and Ready for PR  
**Investigation Status:** ✅ Complete with Technical Details  
**Code Validation:** ✅ Verified Against Source Code  
**Examples Provided:** ✅ mdadm and netdev Use Cases Covered
