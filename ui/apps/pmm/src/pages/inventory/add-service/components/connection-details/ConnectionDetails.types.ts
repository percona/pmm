import type { PropsWithChildren } from 'react';

export interface ConnectionDetailsProps extends PropsWithChildren {
  // Picks the node and pmm-agent that run the exporter.
  withNodes?: boolean;
  isAddressDisabled?: boolean;
}
