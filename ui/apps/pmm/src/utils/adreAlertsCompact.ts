const DEFAULT_MAX_ALERTS = 35;
/** Max size of JSON for `{ ok, alerts }` returned from check_alerts frontend tool. */
const DEFAULT_MAX_TOOL_JSON = 26_000;

function cloneSliceAlerts(data: unknown, maxAlerts: number): { next: unknown; sliced: boolean } {
  if (!data || typeof data !== 'object') {
    return { next: data, sliced: false };
  }
  const root = { ...(data as Record<string, unknown>) };
  const inner = root.data;
  if (inner && typeof inner === 'object') {
    const d = { ...(inner as Record<string, unknown>) };
    const arr = d.alerts;
    if (Array.isArray(arr) && arr.length > maxAlerts) {
      d.alerts = arr.slice(0, maxAlerts);
      d.alerts_total = arr.length;
      d.alerts_truncated = true;
      root.data = d;
      return { next: root, sliced: true };
    }
  }
  const top = root.alerts;
  if (Array.isArray(top) && top.length > maxAlerts) {
    root.alerts = top.slice(0, maxAlerts);
    root.alerts_total = top.length;
    root.alerts_truncated = true;
    return { next: root, sliced: true };
  }
  return { next: root, sliced: false };
}

/**
 * Limits alert list length and overall JSON size for `frontend_tool_results` (Holmes context).
 */
export function compactAdreAlertsForToolResult(
  data: unknown,
  maxAlerts = DEFAULT_MAX_ALERTS,
  maxToolJson = DEFAULT_MAX_TOOL_JSON
): { value: unknown; truncated: boolean } {
  const { next, sliced } = cloneSliceAlerts(data, maxAlerts);
  let truncated = sliced;
  const candidate = { ok: true, alerts: next };
  const json = JSON.stringify(candidate);
  if (json.length <= maxToolJson) {
    return { value: next, truncated };
  }

  truncated = true;
  const preview = json.slice(0, 12_000);
  return {
    value: {
      note: `Alerts tool output was ~${json.length} characters; only a preview is included.`,
      preview: `${preview}…`,
    },
    truncated,
  };
}
