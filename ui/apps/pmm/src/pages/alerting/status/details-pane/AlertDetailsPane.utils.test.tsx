import { renderHook, waitFor } from '@testing-library/react';
import { AlertRow } from '../AlertsPage.types';
import { useAlertDetailsPane } from './AlertDetailsPane.utils';
import { wrapWithQueryProvider } from 'utils/testUtils';

const { getRulerRule } = vi.hoisted(() => ({
  getRulerRule: vi.fn(),
}));

vi.mock('api/alerting', async (importOriginal) => ({
  ...(await importOriginal<typeof import('api/alerting')>()),
  getRulerRule,
}));

const createAlert = (uid?: string): AlertRow => ({
  type: 'alert',
  id: 'alert-1',
  alertName: 'Alert one',
  ruleName: 'Rule one',
  state: 'Alerting',
  nodeId: 'node-1',
  serviceName: 'service-1',
  summary: 'Alert summary',
  labels: {},
  annotations: {},
  expression: 'up == 0',
  age: '1m',
  silenced: false,
  silencedBy: [],
  silencedAge: '-',
  rawAlert: { labels: {}, annotations: {} },
  rule: {
    uid,
    name: 'Rule one',
    alerts: [],
  },
});

const renderDetailsHook = (alert: AlertRow) =>
  renderHook(() => useAlertDetailsPane(alert), {
    wrapper: ({ children }) => wrapWithQueryProvider(children),
  });

describe('useAlertDetailsPane', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('returns alert-derived details without a rule UID', () => {
    const { result } = renderDetailsHook(createAlert());

    expect(result.current?.summary.alertName).toBe('Alert one');
    expect(result.current?.ruleConfiguration.ruleUid).toBeUndefined();
    expect(getRulerRule).not.toHaveBeenCalled();
  });

  it('keeps alert-derived details when the ruler request fails', async () => {
    getRulerRule.mockRejectedValue(new Error('Ruler unavailable'));

    const { result } = renderDetailsHook(createAlert('rule-1'));

    await waitFor(() => expect(getRulerRule).toHaveBeenCalledWith('rule-1'));
    expect(result.current?.summary.alertName).toBe('Alert one');
    expect(result.current?.ruleConfiguration.ruleUid).toBe('rule-1');
  });
});
