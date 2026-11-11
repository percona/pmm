import type { NodeRole } from 'types/ha.types';
import type { ManagedNode } from 'types/nodes.types';
import type { MonitoringStatus } from '../Inventory.types';

export interface NodeRow extends ManagedNode {
  agentsStatus: MonitoringStatus;
  haRole?: NodeRole;
}
