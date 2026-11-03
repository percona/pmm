import MenuItem from '@mui/material/MenuItem';
import { SelectInput } from '@percona/peak-ui';
import { type FC, useCallback, useEffect } from 'react';
import { useFormContext, useFormState, useWatch } from 'react-hook-form';
import type {
  ConnectionFormValues,
  NodeFormValues,
} from '../../service-form/ServiceForm.types';
import { FieldGrid } from '../field-grid';
import { Messages } from './NodesAgents.messages';
import type { NodeOption } from './NodesAgents.types';
import { useNodeOptions } from './NodesAgents.hooks';
import { getPreselectedNode } from './NodesAgents.utils';

type Values = NodeFormValues & Pick<ConnectionFormValues, 'address'>;

export const NodesAgents: FC = () => {
  const { control, getValues, setValue } = useFormContext<Values>();
  const { errors } = useFormState({ control, name: ['nodeId', 'pmmAgentId'] });
  const { nodes, isLoading } = useNodeOptions();
  const nodeId = useWatch({ control, name: 'nodeId' });
  const selectedNode = nodes.find((node) => node.nodeId === nodeId);

  const selectAgent = useCallback(
    (node: NodeOption, agentId: string) => {
      setValue('pmmAgentId', agentId, { shouldValidate: !!agentId });

      if (agentId && !node.isPmmServerNode && !getValues('address')) {
        setValue('address', 'localhost');
      }
    },
    [getValues, setValue]
  );

  // A node running several pmm-agents is ambiguous, so the agent is left for the user to pick.
  const selectNode = useCallback(
    (node: NodeOption) => {
      setValue('nodeId', node.nodeId, { shouldValidate: true });
      selectAgent(node, node.agentIds.length === 1 ? node.agentIds[0] : '');
    },
    [selectAgent, setValue]
  );

  useEffect(() => {
    const preselected = getPreselectedNode(nodes);

    if (!getValues('nodeId') && preselected) {
      selectNode(preselected);
    }
  }, [nodes, getValues, selectNode]);

  return (
    <FieldGrid>
      <SelectInput
        name="nodeId"
        label={Messages.nodes}
        loading={isLoading}
        helperText={errors.nodeId?.message}
        formControlProps={{ fullWidth: true, error: !!errors.nodeId }}
        selectFieldProps={{
          onChange: (event) => {
            const node = nodes.find((n) => n.nodeId === event.target.value);

            if (node) {
              selectNode(node);
            }
          },
        }}
      >
        {nodes.map((node) => (
          <MenuItem key={node.nodeId} value={node.nodeId}>
            {node.nodeName}
          </MenuItem>
        ))}
      </SelectInput>
      <SelectInput
        name="pmmAgentId"
        label={Messages.agents}
        helperText={errors.pmmAgentId?.message}
        formControlProps={{ fullWidth: true, error: !!errors.pmmAgentId }}
        selectFieldProps={{
          disabled: (selectedNode?.agentIds.length ?? 0) <= 1,
          onChange: (event) => {
            if (selectedNode) {
              selectAgent(selectedNode, String(event.target.value));
            }
          },
        }}
      >
        {selectedNode?.agentIds.map((agentId) => (
          <MenuItem key={agentId} value={agentId}>
            {agentId}
          </MenuItem>
        ))}
      </SelectInput>
    </FieldGrid>
  );
};
