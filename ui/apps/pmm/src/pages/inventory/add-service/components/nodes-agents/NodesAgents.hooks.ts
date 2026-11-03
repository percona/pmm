import { useNodes } from 'hooks/api/useNodes';
import { useMemo } from 'react';
import { toNodeOptions } from './NodesAgents.utils';

export const useNodeOptions = () => {
  const { data, isLoading } = useNodes();
  const nodes = useMemo(() => toNodeOptions(data?.nodes ?? []), [data]);

  return { nodes, isLoading };
};
