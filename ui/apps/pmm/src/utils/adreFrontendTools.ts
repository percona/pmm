/**
 * Declarations for Holmes POST /api/chat `frontend_tools` (pause mode).
 * Names use `pmm_ui_` prefix to avoid clashing with Holmes built-in tool names.
 * Holmes does not persist these — the client must send them on every request, including resume.
 *
 * Parameter schemas must satisfy OpenAI chat-completions tool `function.parameters`
 * (strict validation: avoid nested objects with additionalProperties; keep a single
 * required field when possible and document optional behavior in description).
 *
 * @see https://github.com/robusta-dev/holmesgpt/blob/master/docs/reference/http-api.md
 */
export const PMM_ADRE_FRONTEND_TOOLS: Array<Record<string, unknown>> = [
  {
    name: 'pmm_ui_navigate_to_dashboard',
    description:
      'Navigate the user’s PMM browser to a Grafana dashboard. Call when the user asks to open/go to/show a dashboard in PMM. First resolve the dashboard UID with grafana_search_dashboards or equivalent, then pass uid here. Prefer this over only pasting markdown links. Optional time range and template variables are not separate schema fields — pass uid only; the UI opens the dashboard with default time.',
    mode: 'pause',
    parameters: {
      type: 'object',
      properties: {
        uid: { type: 'string', description: 'Grafana dashboard UID' },
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
      'Open ServiceNow or an investigation related to ticketing. User may confirm in the browser. Pass exactly one scenario: set url (full ticket URL), or set ticketId and instanceUrl together, or set investigationId for PMM investigation UI. Pass empty string "" for all unused fields.',
    mode: 'pause',
    parameters: {
      type: 'object',
      properties: {
        url: { type: 'string', description: 'Full ServiceNow or ticket URL (use alone)' },
        ticketId: { type: 'string', description: 'Incident sys_id (use with instanceUrl)' },
        instanceUrl: { type: 'string', description: 'ServiceNow instance base URL (use with ticketId)' },
        investigationId: { type: 'string', description: 'PMM investigation id (fallback when no ServiceNow URL)' },
      },
      required: ['url', 'ticketId', 'instanceUrl', 'investigationId'],
    },
  },
  {
    name: 'pmm_ui_check_alerts',
    description:
      'Fetch current firing ADRE/PMM alerts for the user’s session. The client may truncate large lists for model context.',
    mode: 'pause',
    parameters: { type: 'object', properties: {}, required: [] },
  },
  {
    name: 'pmm_ui_render_graph',
    description:
      'Focus the browser on a specific Grafana dashboard panel (viewPanel). Requires dashboard UID and numeric panel id. Optional time range is not passed via schema — use default Grafana time.',
    mode: 'pause',
    parameters: {
      type: 'object',
      properties: {
        panelId: { type: 'string', description: 'Panel id (numeric string)' },
        dashboardUid: { type: 'string', description: 'Dashboard UID' },
      },
      required: ['panelId', 'dashboardUid'],
    },
  },
];
