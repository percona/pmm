import { ServiceMapOptions } from '../types';

function splitCsv(s: string): string[] {
  if (!s || typeof s !== 'string') {
    return [];
  }
  return s.split(',').map((x) => x.trim()).filter(Boolean);
}

function parseOverrides(raw: string | Record<string, string> | undefined): Record<string, string> {
  if (!raw) {
    return {};
  }
  if (typeof raw === 'object') {
    return raw;
  }
  try {
    return JSON.parse(raw) as Record<string, string>;
  } catch {
    return {};
  }
}

const IPV4_PORT = /^((?:\d{1,3}\.){3}\d{1,3}):(\d+)$/;

/**
 * Human-readable label for raw IP:port / DNS destinations (namespace "external").
 * Does not change graph node ids — only display.
 */
export function getFriendlyExternalLabel(dest: string, opts: ServiceMapOptions): string | undefined {
  if (!dest || dest.startsWith('/')) {
    return undefined;
  }

  const overrides = parseOverrides(opts.destinationLabelOverrides);
  if (overrides[dest]) {
    return overrides[dest];
  }

  const m = dest.match(IPV4_PORT);
  if (!m) {
    return undefined;
  }
  const ip = m[1];
  const port = m[2];

  const clusterIps = new Set(splitCsv(opts.kubernetesApiClusterIPs));
  if ((port === '443' || port === '6443') && clusterIps.has(ip)) {
    return 'Kubernetes API';
  }

  const apiEnis = new Set(splitCsv(opts.kubernetesApiserverEndpointIPs));
  if ((port === '443' || port === '6443') && apiEnis.has(ip)) {
    return 'Kubernetes API (control plane)';
  }

  if (port === '9100') {
    return `Node exporter (${ip})`;
  }

  return undefined;
}
