import type { PropsWithChildren } from 'react';

export interface ConnectionDetailsProps extends PropsWithChildren {
  // Picks the node and pmm-agent that run the exporter.
  withNodes?: boolean;
  isAddressDisabled?: boolean;
  // Username is required, and password too unless TLS is on (see requireCredentials).
  requiresCredentials?: boolean;
  // The exporter's own connection timeout, used when the field is left empty.
  defaultTimeout?: string;
}
