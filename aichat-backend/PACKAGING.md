# AI Chat Backend Packaging Guide

This document describes how to build and package the AI Chat Backend as an RPM for deployment on Red Hat-based Linux distributions.

## Prerequisites

### Build System Requirements

- **Operating System**: RHEL/CentOS/Rocky Linux 8+ or Fedora 35+
- **Go**: Version 1.23 or later
- **RPM Build Tools**: `rpm-build`, `rpmbuild`
- **Development Tools**: `git`, `make`
- **Node.js**: Version 16+ (for MCP servers)

### Installing Prerequisites

```bash
# RHEL/CentOS/Rocky Linux
sudo dnf install -y rpm-build rpm-devel rpmlint make git nodejs npm golang

# Fedora
sudo dnf install -y rpm-build rpm-devel rpmlint make git nodejs npm golang

# Verify Go version
go version  # Should be 1.23+
```

## Building the RPM Package

### Quick Build

```bash
# Clone the repository (if not already done)
cd aichat-backend

# Build RPM package
make rpm
```

### Step-by-Step Build Process

1. **Clean previous builds**:
   ```bash
   make clean
   ```

2. **Create source tarball**:
   ```bash
   make tarball
   # Creates aichat-backend-1.0.0.tar.gz
   ```

3. **Setup RPM build environment**:
   ```bash
   make rpm-setup
   # Creates build/rpm directory structure
   ```

4. **Build the RPM**:
   ```bash
   make rpm
   # Builds both binary and source RPMs
   ```

### Build Outputs

After successful build, you'll find:

- **Binary RPM**: `build/rpm/RPMS/x86_64/aichat-backend-1.0.0-1.el*.x86_64.rpm`
- **Source RPM**: `build/rpm/SRPMS/aichat-backend-1.0.0-1.el*.src.rpm`

## Package Details

### Package Information

- **Name**: `aichat-backend`
- **Version**: `1.0.0`
- **License**: `AGPL-3.0`
- **Architecture**: `x86_64`
- **Dependencies**: `systemd`, `nodejs >= 16.0`, `npm`

### Installed Files

```
/usr/bin/aichat-backend                           # Main binary
/etc/aichat-backend/config.yaml                  # Configuration file
/etc/aichat-backend/mcp-servers.json             # MCP servers config
/usr/lib/systemd/system/aichat-backend.service   # Systemd service
/etc/logrotate.d/aichat-backend                  # Log rotation config
/var/log/aichat-backend/                         # Log directory
/var/lib/aichat-backend/                         # Working directory
```

### System User

The package creates a dedicated system user:
- **Username**: `aichat`
- **Group**: `aichat`
- **Home Directory**: `/var/lib/aichat-backend`
- **Shell**: `/sbin/nologin`

## Installation and Management

### Installing the RPM

```bash
# Install the package
sudo rpm -ivh aichat-backend-1.0.0-1.el*.x86_64.rpm

# Or using dnf/yum
sudo dnf install ./aichat-backend-1.0.0-1.el*.x86_64.rpm
```

### Configuration

1. **Set OpenAI API Key**:
   ```bash
   # Create environment file
   sudo mkdir -p /etc/sysconfig
   sudo tee /etc/sysconfig/aichat-backend << EOF
   OPENAI_API_KEY=your-api-key-here
   EOF
   ```

2. **Configure MCP Servers** (optional):
   ```bash
   # Edit MCP servers configuration
   sudo nano /etc/aichat-backend/mcp-servers.json
   ```

3. **Adjust main configuration** (if needed):
   ```bash
   # Edit main configuration
   sudo nano /etc/aichat-backend/config.yaml
   ```

### Service Management

```bash
# Enable and start the service
sudo systemctl enable aichat-backend
sudo systemctl start aichat-backend

# Check service status
sudo systemctl status aichat-backend

# View logs
sudo journalctl -u aichat-backend -f

# Restart service after configuration changes
sudo systemctl restart aichat-backend
```

### Firewall Configuration

```bash
# Open port 3001 (default port)
sudo firewall-cmd --permanent --add-port=3001/tcp
sudo firewall-cmd --reload
```

## Package Verification

### Check Package Contents

```bash
# List package files
make rpm-check

# Or manually
rpm -qpl aichat-backend-1.0.0-1.el*.x86_64.rpm
```

### Verify Package Dependencies

```bash
# Check dependencies
rpm -qpR aichat-backend-1.0.0-1.el*.x86_64.rpm
```

### Test Installation

```bash
# Install in test environment
make rpm-install

# Test service
curl http://localhost:3001/health

# Check MCP tools
curl http://localhost:3001/api/v1/mcp/tools
```

## Customization

### Building with Custom Version

```bash
# Build with custom version
make rpm VERSION=1.1.0 RELEASE=2
```

### Modifying the Spec File

The spec file `aichat-backend.spec` can be customized for specific requirements:

1. **Change installation paths**:
   ```spec
   %{_bindir}/aichat-backend           # /usr/bin/
   %{_sysconfdir}/aichat-backend/      # /etc/aichat-backend/
   %{_unitdir}/                        # /usr/lib/systemd/system/
   ```

2. **Add additional dependencies**:
   ```spec
   Requires: your-additional-package
   ```

3. **Modify service configuration**:
   Edit the systemd service definition in the spec file.

## Troubleshooting

### Common Build Issues

1. **Go version too old**:
   ```bash
   # Install newer Go version
   wget https://golang.org/dl/go1.23.0.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
   export PATH=/usr/local/go/bin:$PATH
   ```

2. **Missing dependencies**:
   ```bash
   # Install build dependencies
   sudo dnf groupinstall "Development Tools"
   sudo dnf install rpm-build rpm-devel
   ```

3. **Permission issues**:
   ```bash
   # Set up rpmbuild directory in home
   mkdir -p ~/rpmbuild/{BUILD,BUILDROOT,RPMS,SOURCES,SPECS,SRPMS}
   echo "%_topdir $HOME/rpmbuild" > ~/.rpmmacros
   ```

### Runtime Issues

1. **Service fails to start**:
   ```bash
   # Check service logs
   sudo journalctl -u aichat-backend

   # Check configuration
   sudo /usr/bin/aichat-backend -config /etc/aichat-backend/config.yaml -validate
   ```

2. **Permission errors**:
   ```bash
   # Fix file ownership
   sudo chown -R aichat:aichat /var/lib/aichat-backend
   sudo chown -R aichat:aichat /var/log/aichat-backend
   ```

3. **Port binding issues**:
   ```bash
   # Check if port is in use
   sudo netstat -tlnp | grep 3001

   # Check SELinux
   sudo setsebool -P httpd_can_network_connect 1
   ```

## Security Considerations

### File Permissions

The RPM sets secure permissions:
- Configuration files: `0644` (readable by all, writable by root)
- Binary: `0755` (executable)
- Log directory: `0755` owned by `aichat:aichat`
- Working directory: `0755` owned by `aichat:aichat`

### Service Security

The systemd service includes security hardening:
- Runs as non-root user (`aichat`)
- Private temporary directory
- Protected system directories
- Limited capabilities
- No new privileges

### Network Security

- Service binds to all interfaces by default
- Configure firewall rules appropriately
- Consider using reverse proxy (nginx/apache) for SSL termination

## Maintenance

### Log Rotation

Log rotation is configured automatically:
- Daily rotation
- Keep 30 days of logs
- Compress old logs
- Reload service after rotation

### Updates

To update the package:
```bash
# Build new version
make rpm VERSION=1.1.0

# Install update
sudo rpm -Uvh aichat-backend-1.1.0-1.el*.x86_64.rpm

# Service will restart automatically
```

### Backup

Important files to backup:
- `/etc/aichat-backend/` - Configuration
- `/var/lib/aichat-backend/` - Working directory
- `/var/log/aichat-backend/` - Logs (optional)

## Uninstallation

```bash
# Stop and disable service
sudo systemctl stop aichat-backend
sudo systemctl disable aichat-backend

# Remove package
sudo rpm -e aichat-backend

# Note: Configuration files marked as %config(noreplace) will be preserved
# User and group will be removed automatically
```

## Building in CI/CD

### GitHub Actions Example

```yaml
name: Build RPM
on: [push, pull_request]

jobs:
  build-rpm:
    runs-on: ubuntu-latest
    container: rockylinux:8
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Install build dependencies
      run: |
        dnf install -y rpm-build golang git make nodejs npm
        
    - name: Build RPM
      run: |
        cd aichat-backend
        make rpm
        
    - name: Upload RPM artifacts
      uses: actions/upload-artifact@v3
      with:
        name: rpm-packages
        path: aichat-backend/build/rpm/RPMS/
```

This packaging approach provides a production-ready deployment method for the AI Chat Backend service on enterprise Linux distributions. 