/**
 * Declarations for Holmes POST /api/chat `frontend_tools` (pause mode).
 * Names use `pmm_ui_` prefix to avoid clashing with Holmes built-in tool names.
 * Holmes does not persist these — the client must send them on every request, including resume.
 * @see https://github.com/robusta-dev/holmesgpt/blob/master/docs/reference/http-api.md
 */
export const PMM_ADRE_FRONTEND_TOOLS: Array<Record<string, unknown>> = [
  {
    name: 'pmm_ui_navigate_to_dashboard',
    description:
      'Navigate the user’s PMM browser to a Grafana dashboard. Call when the user asks to open/go to/show a dashboard in PMM. First resolve the dashboard UID with grafana_search_dashboards or equivalent, then pass uid here. Prefer this over only pasting markdown links.',
    mode: 'pause',
    parameters: {
      type: 'object',
      properties: {
        uid: { type: 'string', description: 'Grafana dashboard UID' },
        from: { type: 'string', description: 'Optional time from (e.g. now-6h)' },
        to: { type: 'string', description: 'Optional time to (e.g. now)' },
        vars: {
          type: 'object',
          description: 'Optional dashboard template variables (key -> value)',
          additionalProperties: { type: 'string' },
        },
      },
      required: ['uid'],
    },
  },
  {
    name: 'pmm_ui_open_explore',
    description: 'Open Grafana Explore in PMM with a pre-filled query (Explore left pane JSON/state).',
    mode: 'pause',
    parameters: {
      type: 'object',
      properties: {
        query: {
          type: 'string',
          description:
            'Explore left pane value. Pass raw JSON/state; if already URL-encoded (contains %XX), it is passed through without double-encoding.',
        },
      },
      required: ['query'],
    },
  },
  {
    name: 'pmm_ui_open_investigation',
    description: 'Open a PMM Investigations detail page by investigation id.',
    mode: 'pause',
    parameters: {
      type: 'object',
      properties: {
        id: { type: 'string', description: 'Investigation UUID' },
      },
      required: ['id'],
    },
  },
  {
    name: 'pmm_ui_focus_qan_query',
    description:
      'Open QAN AI Insights for a specific service and query id. serviceId should be the bare UUID (any /service_id/ prefix is stripped client-side).',
    mode: 'pause',
    parameters: {
      type: 'object',
      properties: {
        serviceId: { type: 'string', description: 'PMM service UUID' },
        queryId: { type: 'string', description: 'QAN query id' },
      },
      required: ['serviceId', 'queryId'],
    },
  },
  {
    name: 'pmm_ui_open_servicenow_ticket',
    description:
      'Open ServiceNow or an investigation related to ticketing. User may confirm in the browser. Use url, or ticketId+instanceUrl, or investigationId as fallback.',
    mode: 'pause',
    parameters: {
      type: 'object',
      properties: {
        url: { type: 'string', description: 'Full ServiceNow or ticket URL' },
        ticketId: { type: 'string', description: 'Incident sys_id when using instanceUrl' },
        instanceUrl: { type: 'string', description: 'ServiceNow instance base URL' },
        investigationId: { type: 'string', description: 'PMM investigation id when linking to investigation UI' },
      },
    },
  },
  {
    name: 'pmm_ui_check_alerts',
    description:
      'Fetch current firing ADRE/PMM alerts for the user’s session. The client may truncate large lists for model context.',
    mode: 'pause',
    parameters: { type: 'object', properties: {} },
  },
  {
    name: 'pmm_ui_render_graph',
    description:
      'Focus the browser on a specific Grafana dashboard panel (viewPanel). Requires dashboard UID and numeric panel id.',
    mode: 'pause',
    parameters: {
      type: 'object',
      properties: {
        panelId: { type: 'string', description: 'Panel id (numeric string)' },
        dashboardUid: { type: 'string', description: 'Dashboard UID' },
        from: { type: 'string', description: 'Optional time from' },
        to: { type: 'string', description: 'Optional time to' },
      },
      required: ['panelId', 'dashboardUid'],
    },
  },
];
