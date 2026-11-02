import { useQuery } from '@tanstack/react-query';
import { api } from 'api/api';
import { NAV_OS } from 'contexts/navigation/navigation.constants';
import { useManagedServices } from 'hooks/api/useServices';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';
import { useMemo } from 'react';
import type { ManagedService } from 'types/services.types';
import { INVENTORY_STALE_TIME, NODES_QUERY_KEY } from '../search.constants';
import { SERVICE_TECH, TECH_ALIASES, searchItem } from '../search.keywords';
import { Messages } from '../search.messages';
import type { RemoteState, SearchItem } from '../search.types';

// What /v1/management/nodes returns per node, after the client's camelCase conversion.
export interface ManagedNode {
  nodeId: string;
  nodeName: string;
  nodeType: string;
  address: string;
}

const SERVICES_FALLBACK_PATH = `${PMM_NEW_NAV_GRAFANA_PATH}/inventory/services`;
const NODE_DASHBOARD = `${PMM_NEW_NAV_GRAFANA_PATH}/d/node-instance-summary/node-summary`;

export const listManagedNodes = async () =>
  (await api.get<{ nodes: ManagedNode[] }>('/management/nodes')).data;

// Accepts both 'mysql' and 'SERVICE_TYPE_MYSQL_SERVICE'.
export const serviceTypeKey = (serviceType: string) =>
  serviceType.toLowerCase().replace(/^service_type_|_service$/g, '');

export const mapServices = (services: ManagedService[] = []): SearchItem[] =>
  services.map((service) => {
    const type = serviceTypeKey(service.serviceType ?? '');
    const tech = SERVICE_TECH[type];
    const name = tech?.name ?? service.serviceType;

    return searchItem({
      id: `service:${service.serviceId}`,
      kind: 'service',
      group: 'inventory',
      label: service.serviceName,
      description: [
        Messages.crumbs.inventory,
        Messages.crumbs.serviceOn(name, service.nodeName),
      ],
      keywords: [
        service.nodeName,
        service.environment,
        service.cluster,
        service.replicationSet,
        service.address,
        type,
      ].filter(Boolean),
      aliases: tech?.nav && TECH_ALIASES[tech.nav.id],
      navIcon: tech?.nav?.icon,
      to: tech?.dashboard
        ? `${tech.dashboard}?var-service_name=${encodeURIComponent(service.serviceName)}`
        : SERVICES_FALLBACK_PATH,
      remote: true,
      // Services share their technology's dashboard, so each one keeps its own entry.
      dedupeKey: `service:${service.serviceId}`,
    });
  });

export const mapNodes = (nodes: ManagedNode[] = []): SearchItem[] =>
  nodes.map((node) =>
    searchItem({
      id: `node:${node.nodeId}`,
      kind: 'node',
      group: 'inventory',
      label: node.nodeName,
      description: [
        Messages.crumbs.inventory,
        Messages.crumbs.node(node.address),
      ],
      keywords: [node.address].filter(Boolean),
      navIcon: NAV_OS.icon,
      to: `${NODE_DASHBOARD}?var-node_name=${encodeURIComponent(node.nodeName)}`,
      remote: true,
      dedupeKey: `node:${node.nodeId}`,
    })
  );

export const useInventoryItems = (
  enabled: boolean
): { items: SearchItem[]; state: RemoteState } => {
  const services = useManagedServices(
    {},
    { enabled, staleTime: INVENTORY_STALE_TIME }
  );
  const nodes = useQuery({
    queryKey: NODES_QUERY_KEY,
    queryFn: listManagedNodes,
    staleTime: INVENTORY_STALE_TIME,
    enabled,
  });
  const servicesData = services.data;
  const nodesData = nodes.data;
  const isLoading = services.isLoading || nodes.isLoading;
  const isError = services.isError || nodes.isError;
  const { refetch: refetchServices } = services;
  const { refetch: refetchNodes } = nodes;

  const items = useMemo(
    () => [
      ...mapServices(servicesData?.services),
      ...mapNodes(nodesData?.nodes),
    ],
    [servicesData, nodesData]
  );

  const state = useMemo<RemoteState>(
    () => ({
      status: isLoading
        ? 'loading'
        : isError
          ? 'error'
          : servicesData || nodesData
            ? 'ready'
            : 'idle',
      refetch: () => {
        void refetchServices();
        void refetchNodes();
      },
    }),
    [isLoading, isError, servicesData, nodesData, refetchServices, refetchNodes]
  );

  return { items, state };
};
