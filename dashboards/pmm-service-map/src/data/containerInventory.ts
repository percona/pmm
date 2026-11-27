import { podId } from './podAggregate';

/**
 * Coroot `container_info` lists every cgroup/container the agent tracks. Merge those paths into
 * podToContainerAppIds so the grouped-pod sidebar includes sidecars without TCP listeners or L7.
 */
export function mergeContainerIdsIntoPodMap(
  existing: Record<string, string[]> | undefined,
  containerIds: string[]
): Record<string, string[]> {
  const byPod = new Map<string, Set<string>>();

  for (const [pid, list] of Object.entries(existing ?? {})) {
    let s = byPod.get(pid);
    if (!s) {
      s = new Set();
      byPod.set(pid, s);
    }
    for (const id of list) {
      s.add(id);
    }
  }

  for (const cid of containerIds) {
    if (!cid.startsWith('/k8s/')) {
      continue;
    }
    const pid = podId(cid);
    if (pid === cid) {
      continue;
    }
    let s = byPod.get(pid);
    if (!s) {
      s = new Set();
      byPod.set(pid, s);
    }
    s.add(cid);
  }

  const out: Record<string, string[]> = {};
  for (const [pid, set] of byPod) {
    out[pid] = Array.from(set).sort();
  }
  return out;
}
