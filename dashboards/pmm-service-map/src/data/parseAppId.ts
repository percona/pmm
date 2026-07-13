import { ParsedAppId } from '../types';

/**
 * Parse a recording-rule app/dest ID into structured parts.
 *
 * Formats seen in the wild:
 *   /k8s/<namespace>/<name>         (coroot default for k8s workloads)
 *   clusterId:namespace:Kind:name   (coroot multi-cluster)
 *   <plain-string>                  → namespace "external" (see below)
 *
 * "External" bucket: any destination that does not match /k8s/ns/name or cluster:...:Kind:name.
 * Typical examples: public cloud endpoints (34.x, 52.x), node/LB IPs (172.31.x, 10.x when not mapped),
 * kube-apiserver, DNS names, or ClusterIPs still unresolved after actual_destination + listen_info.
 * To investigate a specific IP: compare destination vs container_net_tcp_listen_info and rr_* labels in Prometheus.
 */
export function parseAppId(raw: string): ParsedAppId {
  if (!raw) {
    return { raw: '', namespace: '', name: '(unknown)', kind: '' };
  }

  // /k8s/<namespace>/<name>
  const k8sMatch = raw.match(/^\/k8s\/([^/]+)\/(.+)$/);
  if (k8sMatch) {
    return { raw, namespace: k8sMatch[1], name: k8sMatch[2], kind: 'k8s' };
  }

  // clusterId:namespace:Kind:name
  const colonParts = raw.split(':');
  if (colonParts.length >= 4) {
    return {
      raw,
      namespace: colonParts[1],
      name: colonParts[3],
      kind: colonParts[2],
    };
  }

  // Plain string — could be an IP, DNS name, or external service
  return { raw, namespace: 'external', name: raw, kind: '' };
}

export function formatNodeLabel(parsed: ParsedAppId, mode: 'name' | 'namespace-name' | 'raw'): string {
  if (mode === 'raw') {
    return parsed.raw;
  }
  if (parsed.displayName) {
    if (mode === 'namespace-name' && parsed.namespace === 'external') {
      return parsed.displayName;
    }
    if (mode === 'namespace-name' && parsed.namespace && parsed.namespace !== 'external') {
      return `${parsed.namespace}/${parsed.displayName}`;
    }
    return parsed.displayName;
  }
  switch (mode) {
    case 'namespace-name':
      if (parsed.namespace && parsed.namespace !== 'external') {
        return `${parsed.namespace}/${parsed.name}`;
      }
      return parsed.name;
    case 'name':
    default:
      return parsed.name;
  }
}
