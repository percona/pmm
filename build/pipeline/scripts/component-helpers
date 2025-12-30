#!/bin/bash
# Shared helper functions for component builds

set -o errexit

# Build a single component
build_component() {
    local component=$1
    local build_cmd=$2
    
    mkdir -p /output /build/source
    
    # Get component info from .gitmodules
    local url
    local ref
    
    url=$(gitmodules /tmp/.gitmodules "${component}" "url" 2>/dev/null || true)
    ref=$(gitmodules /tmp/.gitmodules "${component}" "branch" 2>/dev/null || \
          gitmodules /tmp/.gitmodules "${component}" "tag" 2>/dev/null || true)
    
    # Fallback for components not in .gitmodules
    if [ -z "$url" ] || [ -z "$ref" ]; then
        case "${component}" in
            VictoriaMetrics)
                url="https://github.com/VictoriaMetrics/VictoriaMetrics.git"
                ref="a5e3c6d4492db765800363dfae48a04b4d7888be"
                ;;
            redis_exporter)
                url="https://github.com/oliver006/redis_exporter.git"
                ref="8d5f9dea4a8863ce7cad62352957ae5e75047346"
                ;;
            nomad)
                url="https://github.com/hashicorp/nomad.git"
                ref="9103d938133311b2da905858801f0e111a2df0a1"
                ;;
            percona-toolkit)
                url="https://github.com/percona/percona-toolkit.git"
                ref="HEAD"
                ;;
            *)
                echo "Error: No URL/ref found for ${component}" >&2
                return 1
                ;;
        esac
    fi
    
    local src_dir="/build/source/${component}"
    
    # Clone or update repository
    if [ ! -d "${src_dir}" ]; then
        echo "Cloning ${component}..."
        if grep -qE '^[0-9a-f]{7,40}$' <<< "${ref}"; then
            git clone "${url}" "${src_dir}"
            cd "${src_dir}"
            git checkout "${ref}"
        else
            git clone --depth 1 --branch "${ref}" "${url}" "${src_dir}"
            cd "${src_dir}"
        fi
    else
        echo "Updating ${component}..."
        cd "${src_dir}"
        if [[ "${ref}" =~ ^[0-9a-f]{7,40}$ ]]; then
            git fetch origin "${ref}" || git fetch origin
            git checkout "${ref}"
        else
            git fetch --depth 1 origin "${ref}"
            git reset --hard FETCH_HEAD
        fi
        git clean -fdx
    fi
    
    # Execute build command
    echo "Building ${component}..."
    eval "${build_cmd}"
}
