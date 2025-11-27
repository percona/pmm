# PMM node_exporter Configuration Investigation - Findings and Documentation

## Executive Summary

This document summarizes the investigation into PMM's node_exporter configuration mechanism and provides guidance on customizing node_exporter collectors. A comprehensive documentation PR has been prepared for the pmm-doc repository.

## Investigation Results

### How node_exporter is Started

1. **Configuration Source**: `managed/services/agents/node.go`
   - The `nodeExporterConfig()` function (line 35) generates all node_exporter arguments
   - Arguments are passed to pmm-agent via the SetStateRequest API
   - pmm-agent starts node_exporter as a subprocess using the process supervisor

2. **Default Collectors** (for non-macOS systems):
   
   **Enabled Collectors:**
   - High Resolution (5s): cpu, diskstats, filefd, filesystem, loadavg, meminfo, meminfo_numa, netdev, netstat, processes, stat, time, vmstat, textfile.hr, standard.go, standard.process, buddyinfo
   - Medium Resolution (10s): hwmon, textfile.mr
   - Low Resolution (60s): bonding, entropy, uname, textfile.lr
   
   **Disabled Collectors (line 81-106):**
   - `mdadm` - RAID array statistics
   - `arp`, `bcache`, `conntrack`, `drbd`, `edac`, `infiniband`, `interrupts`, `ipvs`, `ksmd`, `logind`, `mountstats`, `netclass`, `nfs`, `nfsd`, `ntp`, `qdisc`, `runit`, `sockstat`, `supervisord`, `systemd`, `tcpstat`, `timex`, `wifi`, `xfs`, `zfs`

### Customization Mechanism

**Supported: Disabling Collectors**

The system supports **disabling** collectors through:

1. **CLI Flag**: `pmm-admin config --disable-collectors=collector1,collector2`
   - Defined in: `admin/commands/config.go` (line 52, 127)
   - Passed to pmm-agent setup command
   - Stored in agent configuration

2. **API Parameter**: `disable_collectors` array
   - Defined in: `api/inventory/v1/agents.proto` (AddNodeExporterParams)
   - Accepted by AddNodeExporter API endpoint

3. **Implementation**: `managed/utils/collectors/collectors.go`
   - `FilterOutCollectors()` function removes disabled collectors from arguments
   - Called in `node.go` line 124

**Example Usage:**
```bash
pmm-admin config \
    --server-url=https://admin:admin@pmm-server:443 \
    --disable-collectors=netdev,netstat,vmstat \
    192.168.1.10 generic node1
```

### Limitations Identified

**NOT Supported: Enabling Disabled Collectors**

1. **No mechanism to enable disabled collectors**
   - The API only has `disable_collectors`, no `enable_collectors` parameter
   - Cannot enable mdadm collector through configuration
   - Cannot enable any of the 26+ disabled collectors

2. **No mechanism for collector-specific flags**
   - Cannot add flags like `--collector.netdev.address-info`
   - Cannot pass custom arguments to specific collectors
   - All collector configurations are hardcoded in `node.go`

3. **Configuration is immutable after setup**
   - Must unregister and re-register node to change collector settings
   - Cannot modify collector configuration through API after registration

### Code References

**Key Files:**
- `managed/services/agents/node.go` - node_exporter configuration (lines 35-146)
- `managed/utils/collectors/collectors.go` - collector filtering logic
- `admin/commands/config.go` - CLI flag definition (line 52)
- `api/inventory/v1/agents.proto` - API definition

**Relevant Functions:**
- `nodeExporterConfig()` - Generates node_exporter arguments
- `FilterOutCollectors()` - Removes disabled collectors from argument list
- `DisableDefaultEnabledCollectors()` - Helper for other exporters

## Documentation Created

### Location
- Repository: `/tmp/pmm-doc`
- Branch: `document-node-exporter-customization`
- Commit: 2892849c7

### Files Modified

1. **`docs/how-to/extend-metrics.md`** (182 lines added)
   - New section: "Configure node_exporter Collectors"
   - Complete list of default enabled/disabled collectors
   - Documentation on `--disable-collectors` flag
   - Usage examples and common use cases
   - Limitations clearly explained
   - Workaround using textfile collector
   - Example bash script for mdadm metrics collection

2. **`docs/details/commands/pmm-admin.md`** (3 lines added)
   - Added `--disable-collectors` flag to `pmm-admin config` documentation
   - Cross-reference to extend-metrics.md

### Documentation Highlights

**What's Covered:**
- ✅ How to disable collectors during setup
- ✅ List of all default enabled/disabled collectors
- ✅ Common use cases (reduce network overhead, memory pressure)
- ✅ Limitations clearly stated
- ✅ Workaround for disabled collectors using textfile collector
- ✅ Example script for mdadm metrics
- ✅ Guidance on filing feature requests

**What Users Will Learn:**
- Which collectors are available and their default state
- How to disable collectors to reduce resource usage
- Why they cannot enable disabled collectors (technical limitation)
- Alternative approach using textfile collector
- How to create custom metrics scripts

## Specific Use Case Solutions

### Use Case 1: Enable mdadm Collector

**Problem**: mdadm collector is disabled by default, cannot be enabled through configuration.

**Solution**: Use textfile collector workaround (documented in extend-metrics.md)

```bash
#!/bin/bash
# /usr/local/bin/mdadm_metrics.sh
OUTPUT_FILE="/usr/local/percona/pmm2/collectors/textfile-collector/low-resolution/mdadm.prom"

echo "# HELP node_md_state RAID array state (1=active, 0=inactive)" > "$OUTPUT_FILE"
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

Add to cron: `*/5 * * * * /usr/local/bin/mdadm_metrics.sh`

### Use Case 2: Enable netdev.address-info

**Problem**: Cannot add collector-specific flags like `--collector.netdev.address-info`.

**Current Status**: Not supported - requires code changes.

**Recommendation**: 
1. Use textfile collector to export network address information
2. File a feature request in PMM JIRA for native support

**Feature Request Template**:
```
Title: Support collector-specific flags for node_exporter

Description:
Add support for passing collector-specific configuration flags to node_exporter,
specifically to enable features like:
- --collector.netdev.address-info
- --collector.systemd.unit-include=<regex>
- Other collector-specific options

This would require:
1. Adding enable_collectors parameter to API
2. Adding support for collector-specific flags/options
3. Updating managed/services/agents/node.go to accept and pass these flags
```

### Use Case 3: Disable Collectors to Reduce Overhead

**Problem**: System with 100+ network interfaces has high CPU usage from netdev collector.

**Solution**: Disable network collectors during setup (SUPPORTED)

```bash
pmm-admin config \
    --server-url=https://admin:admin@pmm-server:443 \
    --disable-collectors=netdev,netclass,netstat \
    192.168.1.10 generic node1
```

This is fully supported and documented.

## Recommendations

### For Users
1. Use `--disable-collectors` during initial setup to customize which collectors run
2. Use textfile collector for metrics from disabled collectors
3. File feature requests in JIRA if you need to enable disabled collectors
4. Plan collector configuration before initial setup (cannot change after registration)

### For PMM Development Team
Consider these enhancements in future releases:

1. **Add enable_collectors support**
   - API parameter to enable disabled collectors
   - Validates against list of available collectors
   - Security consideration: some collectors may have performance impact

2. **Add collector-specific options**
   - Support for passing collector-specific flags
   - API structure: `collector_options: map<string, string>`
   - Example: `{"netdev.address-info": "true"}`

3. **Make collector configuration mutable**
   - Allow updating collector configuration without re-registration
   - API endpoint: ChangeNodeExporter with collector parameters

4. **Add collector presets**
   - Predefined profiles: "minimal", "standard", "comprehensive"
   - Easy way to enable all collectors for troubleshooting

## Testing Performed

The documentation was validated against:
- Source code in `managed/services/agents/node.go`
- API definitions in `api/inventory/v1/agents.proto`
- CLI implementation in `admin/commands/config.go`
- Test cases in `managed/services/agents/node_test.go`

All documented behavior matches the actual implementation.

## Next Steps

To create the documentation PR:

1. **Navigate to the pmm-doc working directory:**
   ```bash
   cd /tmp/pmm-doc
   ```

2. **Review the changes:**
   ```bash
   git diff origin/main docs/how-to/extend-metrics.md
   git diff origin/main docs/details/commands/pmm-admin.md
   ```

3. **Push to your fork** (if you have one):
   ```bash
   git remote add fork https://github.com/YOUR_USERNAME/pmm-doc.git
   git push fork document-node-exporter-customization
   ```

4. **Create PR on GitHub:**
   - Navigate to https://github.com/percona/pmm-doc
   - Click "New Pull Request"
   - Select branch: `document-node-exporter-customization`
   - Title: "Document node_exporter collector customization"
   - Description: Reference this investigation

5. **PR Description Template:**
   ```markdown
   ## Summary
   This PR adds comprehensive documentation for customizing node_exporter collectors in PMM.

   ## Changes
   - Expanded docs/how-to/extend-metrics.md with node_exporter collector configuration
   - Added --disable-collectors flag documentation to pmm-admin config
   - Documented default enabled/disabled collectors
   - Explained limitations and workarounds
   - Added practical examples and use cases

   ## Addresses
   User request to document how to customize node_exporter startup arguments,
   specifically for enabling mdadm collector and netdev.address-info flag.

   ## Key Points
   - Clearly documents what IS supported (disabling collectors)
   - Clearly documents what is NOT supported (enabling disabled collectors)
   - Provides practical workarounds using textfile collector
   - Includes full list of default collector configurations
   ```

## Conclusion

The investigation is complete and comprehensive documentation has been created. The documentation:

✅ Explains how node_exporter is configured in PMM
✅ Lists all default enabled/disabled collectors  
✅ Documents the supported `--disable-collectors` mechanism
✅ Clearly states limitations
✅ Provides practical workarounds for common use cases
✅ Includes specific examples for mdadm and network metrics
✅ Guides users on filing feature requests

The documentation is ready for review and merging into the pmm-doc repository.
