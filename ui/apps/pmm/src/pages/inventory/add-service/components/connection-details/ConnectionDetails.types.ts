import type { PropsWithChildren } from 'react';

export interface ConnectionDetailsProps extends PropsWithChildren {
  // Picks the node and pmm-agent that run the exporter.
  withNodes?: boolean;
  isAddressDisabled?: boolean;
  // Username and password are required unless TLS is on (see requireCredentialsWithoutTls).
  requiresCredentials?: boolean;
  // The exporter's own connection timeout, used when the field is left empty.
  defaultTimeout?: string;
}
