import type { QueryClient } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { addService } from 'api/services';
import { listNodes } from 'api/nodes';
import type { FC } from 'react';
import { AgentStatus, AgentType } from 'types/agent.types';
import { NodeType, type ManagedNode } from 'types/nodes.types';
import { ServiceStatus } from 'types/services.types';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';

const pmmAgentNode = (
  nodeId: string,
  isPmmServerNode: boolean
): ManagedNode => ({
  nodeId,
  nodeName: nodeId,
  nodeType: NodeType.generic,
  address: '',
  isPmmServerNode,
  isPmmInternalNode: false,
  status: ServiceStatus.up,
  customLabels: {},
  services: [],
  properties: {},
  agents: [
    {
      agentId: `${nodeId}-agent`,
      agentType: AgentType.pmmAgent,
      status: AgentStatus.running,
      isConnected: true,
      disabled: false,
      properties: {},
    },
  ],
});

// Requires `vi.mock('api/nodes', () => ({ listNodes: vi.fn() }))` in the test file.
export const mockNodes = () =>
  vi
    .mocked(listNodes)
    .mockResolvedValue([
      pmmAgentNode('pmm-server', true),
      pmmAgentNode('client', false),
    ]);

export const waitForNodePreselect = () =>
  waitFor(() =>
    expect(screen.getByTestId('select-input-node-id')).toHaveValue('pmm-server')
  );

// Pass a queryClient to keep the query cache across renders.
export const renderServiceForm = (Form: FC, queryClient?: QueryClient) =>
  render(
    <TestWrapper routerProps={{ initialEntries: ['/add-instance'] }}>
      {wrapWithQueryProvider(<Form />, queryClient)}
    </TestWrapper>
  );

export const type = (testId: string, value: string) =>
  fireEvent.change(screen.getByTestId(testId), { target: { value } });

// Clicks a switch, checkbox or radio by its label.
export const clickLabel = (label: string) =>
  fireEvent.click(screen.getByLabelText(label));

export const submit = () =>
  fireEvent.click(screen.getByTestId('add-service-submit'));

// Requires `vi.mock('api/services', () => ({ addService: vi.fn() }))` in the test file.
export const mockAddService = () => {
  vi.clearAllMocks();
  vi.mocked(addService).mockResolvedValue(undefined);
};

// The service params of the one add-service request, e.g. sentPayload('mysql').
export const sentPayload = async (service: string) => {
  await waitFor(() => expect(addService).toHaveBeenCalledTimes(1));
  return (vi.mocked(addService).mock.calls[0][0] as Record<string, object>)[
    service
  ];
};
