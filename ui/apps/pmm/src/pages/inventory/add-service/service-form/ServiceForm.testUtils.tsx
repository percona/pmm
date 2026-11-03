import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { addService } from 'api/services';
import { listNodes } from 'api/nodes';
import type { FC } from 'react';
import { AgentStatus, ManagedAgentType } from 'types/agent.types';
import { ManagedNodeType, type UniversalNode } from 'types/nodes.types';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';

const pmmAgentNode = (
  nodeId: string,
  isPmmServerNode: boolean
): UniversalNode => ({
  nodeId,
  nodeName: nodeId,
  nodeType: ManagedNodeType.generic,
  address: '',
  isPmmServerNode,
  isPmmInternalNode: false,
  agents: [
    {
      agentId: `${nodeId}-agent`,
      agentType: ManagedAgentType.pmmAgent,
      status: AgentStatus.running,
      isConnected: true,
    },
  ],
});

// Requires `vi.mock('api/nodes', () => ({ listNodes: vi.fn() }))` in the test file.
export const mockNodes = () =>
  vi.mocked(listNodes).mockResolvedValue({
    nodes: [pmmAgentNode('pmm-server', true), pmmAgentNode('client', false)],
  });

export const waitForNodePreselect = () =>
  waitFor(() =>
    expect(screen.getByTestId('select-input-node-id')).toHaveValue('pmm-server')
  );

export const renderServiceForm = (Form: FC) =>
  render(
    <TestWrapper routerProps={{ initialEntries: ['/add-instance'] }}>
      {wrapWithQueryProvider(<Form />)}
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
