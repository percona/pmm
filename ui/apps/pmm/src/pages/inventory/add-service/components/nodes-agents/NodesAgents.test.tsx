import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { FC } from 'react';
import { FormProvider, useForm, useWatch } from 'react-hook-form';
import { useNodes } from 'hooks/api/useNodes';
import { AgentStatus, ManagedAgentType } from 'types/agent.types';
import { ManagedNodeType, type UniversalNode } from 'types/nodes.types';
import { TestWrapper } from 'utils/testWrapper';
import { NodesAgents } from './NodesAgents';

vi.mock('hooks/api/useNodes', () => ({ useNodes: vi.fn() }));

const pmmAgent = (agentId: string) => ({
  agentId,
  agentType: ManagedAgentType.pmmAgent,
  status: AgentStatus.running,
  isConnected: true,
});

const NODE: Omit<UniversalNode, 'nodeId' | 'nodeName' | 'agents'> = {
  nodeType: ManagedNodeType.generic,
  address: '',
  isPmmServerNode: false,
  isPmmInternalNode: false,
};

const mockNodes = (nodes: UniversalNode[]) =>
  vi.mocked(useNodes).mockReturnValue({
    data: { nodes },
    isLoading: false,
  } as ReturnType<typeof useNodes>);

const Values: FC = () => {
  const [nodeId, pmmAgentId, address] = useWatch({
    name: ['nodeId', 'pmmAgentId', 'address'],
  });
  return (
    <output data-testid="values">{`${nodeId}|${pmmAgentId}|${address}`}</output>
  );
};

const Form: FC = () => {
  const methods = useForm({
    defaultValues: { nodeId: '', pmmAgentId: '', address: '' },
  });
  return (
    <FormProvider {...methods}>
      <NodesAgents />
      <Values />
    </FormProvider>
  );
};

const renderPicker = () =>
  render(
    <TestWrapper>
      <Form />
    </TestWrapper>
  );

describe('NodesAgents', () => {
  it('preselects the PMM Server node and its only agent', async () => {
    mockNodes([
      {
        ...NODE,
        nodeId: 'client',
        nodeName: 'client',
        agents: [pmmAgent('client-agent')],
      },
      {
        ...NODE,
        nodeId: 'pmm-server',
        nodeName: 'pmm-server',
        isPmmServerNode: true,
        agents: [pmmAgent('pmm-server')],
      },
    ]);

    renderPicker();

    await waitFor(() =>
      expect(screen.getByTestId('values')).toHaveTextContent(
        'pmm-server|pmm-server|'
      )
    );
    expect(screen.getByTestId('select-node-id-button')).toHaveTextContent(
      'pmm-server'
    );
    expect(screen.getByTestId('select-input-pmm-agent-id')).toBeDisabled();
  });

  it('points the address at localhost on a client node', async () => {
    mockNodes([
      {
        ...NODE,
        nodeId: 'client',
        nodeName: 'client',
        agents: [pmmAgent('client-agent')],
      },
    ]);

    renderPicker();

    await waitFor(() =>
      expect(screen.getByTestId('values')).toHaveTextContent(
        'client|client-agent|localhost'
      )
    );
  });

  it('leaves the agent to the user when the node runs several', async () => {
    mockNodes([
      {
        ...NODE,
        nodeId: 'client',
        nodeName: 'client',
        agents: [pmmAgent('agent-1'), pmmAgent('agent-2')],
      },
    ]);

    renderPicker();

    await waitFor(() =>
      expect(screen.getByTestId('values')).toHaveTextContent('client||')
    );
    expect(screen.getByTestId('select-input-pmm-agent-id')).toBeEnabled();
  });

  it('switches to the agent of a node picked by the user', async () => {
    mockNodes([
      {
        ...NODE,
        nodeId: 'pmm-server',
        nodeName: 'pmm-server',
        isPmmServerNode: true,
        agents: [pmmAgent('pmm-server')],
      },
      {
        ...NODE,
        nodeId: 'client',
        nodeName: 'client',
        agents: [pmmAgent('client-agent')],
      },
    ]);

    renderPicker();

    await waitFor(() =>
      expect(screen.getByTestId('values')).toHaveTextContent(
        'pmm-server|pmm-server|'
      )
    );

    fireEvent.mouseDown(screen.getAllByRole('combobox')[0]);
    fireEvent.click(await screen.findByRole('option', { name: 'client' }));

    await waitFor(() =>
      expect(screen.getByTestId('values')).toHaveTextContent(
        'client|client-agent|localhost'
      )
    );
  });
});
