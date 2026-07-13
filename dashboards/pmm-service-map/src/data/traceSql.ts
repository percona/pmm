import { SelectedNode } from '../types';

function escapeSql(s: string): string {
  return s.replace(/'/g, "''");
}

const MAX_TRACE_NAME_TOKENS = 48;
const MIN_PATH_SEGMENT_LEN = 3;
const MAX_EXACT_IN = 200;

/**
 * Repeatedly strip a trailing -<digits> from the pod segment (StatefulSet / replica ordinals).
 * Verified on live data: metrics use `pmm-ha-pg-db-instance1-2v9k-0` while OTLP ServiceName is
 * `.../pmm-ha-pg-db-instance1-2v9k` (same path without the last ordinal).
 */
export function podSegmentOrdinalVariants(podSeg: string): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  let s = podSeg;
  while (s && !seen.has(s)) {
    seen.add(s);
    out.push(s);
    const m = s.match(/^(.+)-(\d+)$/);
    if (!m) {
      break;
    }
    const next = m[1];
    if (next === s) {
      break;
    }
    s = next;
  }
  return out;
}

/**
 * From Coroot-style app_id paths, derive likely OTLP `service.name` strings (ordinal aliases).
 */
export function expandOtelServiceNameCandidates(ids: string[]): string[] {
  const out = new Set<string>();
  for (const raw of ids) {
    if (!raw?.trim()) {
      continue;
    }
    const id = raw.trim();
    out.add(id);

    const m = id.match(/^\/k8s\/([^/]+)\/(.+)$/);
    if (!m) {
      continue;
    }
    const ns = m[1];
    const rest = m[2];

    if (!rest.includes('/')) {
      for (const podVar of podSegmentOrdinalVariants(rest)) {
        out.add(`/k8s/${ns}/${podVar}`);
      }
      continue;
    }

    const slashIdx = rest.indexOf('/');
    const podSeg = rest.slice(0, slashIdx);
    const afterPod = rest.slice(slashIdx + 1);
    for (const podVar of podSegmentOrdinalVariants(podSeg)) {
      out.add(`/k8s/${ns}/${podVar}`);
      out.add(`/k8s/${ns}/${podVar}/${afterPod}`);
    }
  }
  return [...out];
}

/**
 * Coroot/Prometheus app_id paths vs OTLP: also substring on segments for short service names.
 */
export function expandTraceSearchTokens(ids: string[]): string[] {
  const out = new Set<string>();
  for (const raw of ids) {
    if (!raw?.trim()) {
      continue;
    }
    const t = raw.trim();
    out.add(t);
    const m = t.match(/^\/k8s\/[^/]+\/(.+)$/);
    if (m) {
      const rest = m[1];
      out.add(rest);
      for (const seg of rest.split('/')) {
        if (seg.length >= MIN_PATH_SEGMENT_LEN) {
          out.add(seg);
        }
      }
    }
  }
  return Array.from(out)
    .sort((a, b) => b.length - a.length)
    .slice(0, MAX_TRACE_NAME_TOKENS);
}

/** Container + pod ids from the graph (this node only; edge selection still uses both endpoints). */
export function collectNodeTraceNameIds(node: SelectedNode): string[] {
  const list: string[] = [];
  if (node.traceServiceNames && node.traceServiceNames.length > 0) {
    list.push(...node.traceServiceNames);
  } else {
    list.push(node.id);
  }
  if (!list.includes(node.id)) {
    list.push(node.id);
  }
  return list;
}

/**
 * Exact IN (expanded OTLP names) OR case-insensitive substring fallbacks.
 */
export function buildNodeTraceServiceWhereClause(expandedIds: string[], substringTokens: string[]): string {
  const exact = [...new Set(expandedIds.filter(Boolean))].slice(0, MAX_EXACT_IN);
  const inPart =
    exact.length > 0
      ? `ServiceName IN (${exact.map((id) => `'${escapeSql(id)}'`).join(', ')})`
      : '1 = 0';

  const posParts: string[] = [];
  for (const t of substringTokens) {
    const low = escapeSql(t.toLowerCase());
    posParts.push(`position(lower(coalesce(ServiceName, '')), lower('${low}')) > 0`);
  }

  if (posParts.length === 0) {
    return `(${inPart})`;
  }
  return `(${inPart} OR ${posParts.join(' OR ')})`;
}
