import { act, fireEvent, render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import messenger from 'lib/messenger';
import type { ListThresholdsResponse } from 'types/alerting.types';
import { TestWrapper } from 'utils/testWrapper';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import AlertThresholds from './AlertThresholds';
import { Messages } from './AlertThresholds.messages';

const mocks = vi.hoisted(() => ({
  useNodeThresholds: vi.fn(),
  usePrometheusAlertRules: vi.fn(),
  applyThresholds: vi.fn(),
}));

vi.mock('hooks/api/useNodeThresholds', () => ({
  useNodeThresholds: mocks.useNodeThresholds,
  useBatchUpdateNodeThresholds: () => ({ mutateAsync: mocks.applyThresholds }),
  nodeThresholdsQueryKey: (nodeId: string) => [
    'alerting:nodeThresholds',
    nodeId,
  ],
}));

vi.mock('hooks/api/usePrometheusAlertRules', () => ({
  usePrometheusAlertRules: mocks.usePrometheusAlertRules,
  PROMETHEUS_ALERT_RULES_QUERY_KEY: ['alerting:prometheusRules'],
}));

// Held as stable module-level objects on purpose: TanStack Query's structural sharing
// keeps `data` referentially stable across refetches that change nothing, and the fix
// under test relies on that identity to decide when to re-seed the form.
const THRESHOLDS: Record<string, ListThresholdsResponse> = {
  'node-1': {
    thresholds: [
      {
        ruleId: 'rule-1',
        paramName: 'threshold',
        defaultValue: 80,
        effectiveValue: 90,
        isOverridden: true,
        unit: 'PARAM_UNIT_PERCENTAGE',
      },
    ],
  } as ListThresholdsResponse,
  'node-2': {
    thresholds: [
      {
        ruleId: 'rule-1',
        paramName: 'threshold',
        defaultValue: 80,
        effectiveValue: 70,
        isOverridden: true,
        unit: 'PARAM_UNIT_PERCENTAGE',
      },
    ],
  } as ListThresholdsResponse,
};

// A fresh object each call, as a real Grafana response is: the payload carries
// evaluation timestamps that change on nearly every poll.
const rulesResponse = () => ({
  data: {
    groups: [
      { rules: [{ name: 'CPU load', labels: { pmm_rule_id: 'rule-1' } }] },
    ],
  },
});

let rulesData: ReturnType<typeof rulesResponse> | undefined;

// A fresh element tree per call: re-rendering the identical element object lets React
// bail out, and these tests need the hooks re-read after a mock changes.
const buildTree = () => (
  <TestWrapper>
    {wrapWithQueryProvider(wrapWithSnackbarProvider(<AlertThresholds />))}
  </TestWrapper>
);

const renderModal = () => {
  const result = render(buildTree());

  return { ...result, refresh: () => result.rerender(buildTree()) };
};

const openFor = (nodeId: string) =>
  act(() => {
    messenger.onMessageReceived({
      data: {
        type: 'OPEN_ALERT_THRESHOLDS_MODAL',
        payload: { nodeId, nodeName: nodeId },
      },
    } as MessageEvent);
  });

const overrideInput = () =>
  screen.getAllByRole('spinbutton')[0] as HTMLInputElement;

const type = (value: string) =>
  fireEvent.change(overrideInput(), { target: { value } });

beforeEach(() => {
  rulesData = undefined;
  mocks.applyThresholds.mockReset();
  mocks.applyThresholds.mockResolvedValue({});
  mocks.useNodeThresholds.mockImplementation((nodeId: string) => ({
    data: THRESHOLDS[nodeId],
    isLoading: false,
  }));
  mocks.usePrometheusAlertRules.mockImplementation(() => ({ data: rulesData }));
});

describe('AlertThresholds', () => {
  it('seeds each field with the effective value once thresholds arrive', () => {
    renderModal();
    openFor('node-1');

    expect(overrideInput().value).toBe('90');
  });

  // The regression this suite exists for. The rules query is separate and slower, and
  // only the thresholds query gates the table, so the operator can be typing when it
  // lands. Seeding off `rows` meant the arriving titles re-seeded the form.
  it('keeps a typed value when the rule titles arrive late', () => {
    const { refresh } = renderModal();
    openFor('node-1');

    type('42');

    rulesData = rulesResponse();
    act(refresh);

    expect(screen.getByText('CPU load')).toBeInTheDocument();
    expect(overrideInput().value).toBe('42');
  });

  // The rules query key is shared with the alerts page, which polls it every 5s, so a
  // fresh-but-equivalent payload arrives repeatedly whenever that page is mounted.
  it('keeps a typed value when an equivalent rules payload arrives again', () => {
    rulesData = rulesResponse();
    const { refresh } = renderModal();
    openFor('node-1');

    type('42');

    rulesData = rulesResponse();
    act(refresh);

    expect(overrideInput().value).toBe('42');
  });

  it('shows the new node values after reopening for a different node', () => {
    renderModal();
    openFor('node-1');
    type('42');

    fireEvent.click(
      screen.getByRole('button', { name: Messages.actions.cancel })
    );
    openFor('node-2');

    expect(overrideInput().value).toBe('70');
  });

  it('discards abandoned edits when the same node is reopened', () => {
    renderModal();
    openFor('node-1');
    type('42');

    fireEvent.click(
      screen.getByRole('button', { name: Messages.actions.cancel })
    );
    openFor('node-1');

    expect(overrideInput().value).toBe('90');
  });

  it('blocks a second submit while the batch is in flight', async () => {
    mocks.applyThresholds.mockReturnValue(new Promise(() => {}));

    renderModal();
    openFor('node-1');
    type('95');

    const submit = screen.getByRole('button', {
      name: Messages.actions.submit,
    });
    await act(async () => {
      fireEvent.click(submit);
    });

    expect(mocks.applyThresholds).toHaveBeenCalledTimes(1);
    expect(submit).toBeDisabled();
  });
});
