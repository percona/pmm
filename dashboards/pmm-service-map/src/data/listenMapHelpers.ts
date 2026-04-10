/**
 * Coroot listen_info series may include both app_id (sometimes shortened) and container_id.
 * Prefer container_id so IP→workload resolution matches full /k8s/ns/pod/container paths.
 */
export function pickListenWorkloadId(labels: Record<string, string | undefined> | undefined): string {
  if (!labels) {
    return '';
  }
  return labels['container_id'] ?? labels['app_id'] ?? '';
}
