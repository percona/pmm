import { useCallback, useEffect, useMemo, useRef, useState, type MouseEvent } from 'react';
import { PanelProps } from '@grafana/data';
import { css } from '@emotion/css';
import {
  ReactFlow,
  Controls,
  Background,
  BackgroundVariant,
  type NodeTypes,
  type EdgeTypes,
  type Edge,
  type Node,
  type ReactFlowInstance,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';

import { ServiceMapData, ServiceMapOptions, SelectedEdge, SelectedNode, TraceFilter, DEFAULT_OPTIONS } from '../types';
import { MapFiltersBar } from './MapFiltersBar';
import { parseAppId, formatNodeLabel } from '../data/parseAppId';
import { aggregateByPod, getChildContainerNodesForPod, listContainerAppIdsForPod, podId } from '../data/podAggregate';
import { filterPodToContainerAppIdsByNamespaces, filterServiceMapByPodSubstring } from '../data/filterByPodName';
import { getFriendlyExternalLabel } from '../data/friendlyExternalLabels';
import { useServiceMapData } from '../data/useServiceMapData';
import { useGraphLayout } from './graph/useGraphLayout';
import { useTraceData } from '../data/useTraceData';
import { useEdgeTrends } from '../data/useEdgeTrends';
import { ServiceNode } from './graph/ServiceNode';
import { ServiceEdge } from './graph/ServiceEdge';
import { NamespaceGroup } from './graph/NamespaceGroup';
import { EdgeDetailSidebar } from './detail/EdgeDetailSidebar';
import { NodeDetailSidebar } from './detail/NodeDetailSidebar';
import { TraceTable } from './traces/TraceTable';
import { HEALTH_COLORS } from '../constants';

const nodeTypes: NodeTypes = {
  serviceNode: ServiceNode as any,
  namespaceGroup: NamespaceGroup as any,
};

const edgeTypes: EdgeTypes = {
  serviceEdge: ServiceEdge as any,
};

const st = {
  root: css`
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    background: #0b0b18;
    color: #e0e0e0;
    overflow: hidden;
    font-family: Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  `,
  topArea: css`
    flex: 1;
    display: flex;
    min-height: 0;
  `,
  graphArea: css`
    flex: 1;
    min-width: 0;
    position: relative;
  `,
  bottomArea: css`
    height: 30%;
    min-height: 100px;
    max-height: 45%;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
  `,
  loading: css`
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #666;
    font-size: 14px;
  `,
  layoutOverlay: css`
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(11, 11, 24, 0.42);
    z-index: 4;
    pointer-events: none;
    font-size: 12px;
    color: #9090b0;
  `,
  error: css`
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #f2495c;
    font-size: 14px;
    padding: 20px;
    text-align: center;
  `,
};

function parseNamespaceRenameMap(raw: string | Record<string, string>): Record<string, string> {
  if (typeof raw === 'object') {
    return raw;
  }
  if (!raw || typeof raw !== 'string') {
    return {};
  }
  try {
    return JSON.parse(raw);
  } catch {
    return {};
  }
}

function ArrowMarkers() {
  return (
    <svg style={{ position: 'absolute', width: 0, height: 0 }}>
      <defs>
        {Object.entries(HEALTH_COLORS).map(([key, color]) => (
          <marker
            key={key}
            id={`arrow-${key}`}
            viewBox="0 0 10 10"
            refX="10"
            refY="5"
            markerWidth={8}
            markerHeight={8}
            orient="auto-start-reverse"
          >
            <path d="M 0 0 L 10 5 L 0 10 z" fill={color} opacity={0.7} />
          </marker>
        ))}
      </defs>
    </svg>
  );
}

export function ServiceMapPanel({ options, width, height, timeRange }: PanelProps<ServiceMapOptions>) {
  const namespaceRenameMap = useMemo(
    () => parseNamespaceRenameMap(options.namespaceRenameMap),
    [options.namespaceRenameMap]
  );

  const resolvedOptions = useMemo(
    () => ({
      ...options,
      namespaceRenameMap,
      tracesDashboardUid: options.tracesDashboardUid ?? DEFAULT_OPTIONS.tracesDashboardUid,
      tracesViewPanel: options.tracesViewPanel ?? DEFAULT_OPTIONS.tracesViewPanel,
      kubernetesApiClusterIPs: options.kubernetesApiClusterIPs ?? DEFAULT_OPTIONS.kubernetesApiClusterIPs,
      kubernetesApiserverEndpointIPs: options.kubernetesApiserverEndpointIPs ?? DEFAULT_OPTIONS.kubernetesApiserverEndpointIPs,
      destinationLabelOverrides: options.destinationLabelOverrides ?? DEFAULT_OPTIONS.destinationLabelOverrides,
      groupByPod: options.groupByPod ?? DEFAULT_OPTIONS.groupByPod,
      hideWeakEdges: options.hideWeakEdges ?? DEFAULT_OPTIONS.hideWeakEdges,
      weakEdgeMaxRps: options.weakEdgeMaxRps ?? DEFAULT_OPTIONS.weakEdgeMaxRps,
    }),
    [options, namespaceRenameMap]
  );

  /** On-panel View toggles; initialized from panel options, then session-local until refresh. */
  const [groupByPod, setGroupByPod] = useState<boolean>(
    () => resolvedOptions.groupByPod ?? DEFAULT_OPTIONS.groupByPod ?? true
  );
  const [hideWeakEdges, setHideWeakEdges] = useState<boolean>(
    () => resolvedOptions.hideWeakEdges ?? DEFAULT_OPTIONS.hideWeakEdges ?? true
  );

  const mapViewOptions = useMemo(
    () => ({
      ...resolvedOptions,
      groupByPod,
      hideWeakEdges,
    }),
    [resolvedOptions, groupByPod, hideWeakEdges]
  );

  const { data, loading, error } = useServiceMapData(mapViewOptions, timeRange);

  const dataWithFriendly = useMemo((): ServiceMapData | null => {
    if (!data) {
      return null;
    }
    return {
      ...data,
      nodes: data.nodes.map((n) => {
        const friendly = getFriendlyExternalLabel(n.id, resolvedOptions);
        return {
          ...n,
          parsed: {
            ...n.parsed,
            ...(friendly ? { displayName: friendly } : {}),
          },
        };
      }),
    };
  }, [data, resolvedOptions]);

  /** Empty Set = all namespaces */
  const [nsPick, setNsPick] = useState<Set<string>>(() => new Set());

  /** Namespace-filtered container-level graph (before pod aggregation). */
  const containerDataFiltered = useMemo(() => {
    if (!dataWithFriendly) {
      return null;
    }
    if (nsPick.size === 0) {
      return dataWithFriendly;
    }
    const filteredNodes = dataWithFriendly.nodes.filter((n) => nsPick.has(n.parsed.namespace));
    const nodeIds = new Set(filteredNodes.map((n) => n.id));
    const filteredEdges = dataWithFriendly.edges.filter((e) => nodeIds.has(e.source) || nodeIds.has(e.target));
    const connectedIds = new Set<string>();
    for (const e of filteredEdges) {
      connectedIds.add(e.source);
      connectedIds.add(e.target);
    }
    const allNodes = dataWithFriendly.nodes.filter((n) => connectedIds.has(n.id));
    const podToContainerAppIds = filterPodToContainerAppIdsByNamespaces(
      dataWithFriendly.podToContainerAppIds,
      nsPick
    );
    return {
      nodes: allNodes,
      edges: filteredEdges,
      namespaces: dataWithFriendly.namespaces,
      podToContainerAppIds,
    };
  }, [dataWithFriendly, nsPick]);

  /** Immediate value for the pod search input (debounced for graph updates to avoid focus loss). */
  const [podFilterInput, setPodFilterInput] = useState('');
  const [podNameFilterApplied, setPodNameFilterApplied] = useState('');
  useEffect(() => {
    const t = window.setTimeout(() => setPodNameFilterApplied(podFilterInput), 320);
    return () => window.clearTimeout(t);
  }, [podFilterInput]);

  /** Substring filter: matching nodes only; edges require both endpoints to match. */
  const dataAfterPodFilter = useMemo(() => {
    if (!containerDataFiltered) {
      return null;
    }
    return filterServiceMapByPodSubstring(containerDataFiltered, podNameFilterApplied);
  }, [containerDataFiltered, podNameFilterApplied]);

  /** Graph passed to ELK / React Flow */
  const filteredData = useMemo(() => {
    if (!dataAfterPodFilter || !containerDataFiltered) {
      return null;
    }
    if (!groupByPod) {
      return dataAfterPodFilter;
    }
    return aggregateByPod(dataAfterPodFilter, mapViewOptions, containerDataFiltered);
  }, [dataAfterPodFilter, groupByPod, mapViewOptions, containerDataFiltered]);

  const { layout, layoutLoading } = useGraphLayout(filteredData, mapViewOptions);

  const [selectedEdge, setSelectedEdge] = useState<SelectedEdge | null>(null);
  const [selectedNode, setSelectedNode] = useState<SelectedNode | null>(null);
  const [traceFilter, setTraceFilter] = useState<TraceFilter>('all');
  const [highlightedNodeId, setHighlightedNodeId] = useState<string | null>(null);

  const { traces, loading: tracesLoading, error: tracesError } = useTraceData(
    selectedEdge,
    selectedNode,
    traceFilter,
    resolvedOptions.clickhouseDatasource,
    timeRange
  );

  const { rpsSeries, latSeries } = useEdgeTrends(
    selectedEdge,
    resolvedOptions.promDatasource,
    timeRange
  );

  const rfRef = useRef<ReactFlowInstance | null>(null);
  const hasInitialFit = useRef(false);
  const handleInit = useCallback((instance: ReactFlowInstance) => {
    rfRef.current = instance;
    if (!hasInitialFit.current) {
      instance.fitView({ padding: 0.15 });
      hasInitialFit.current = true;
    }
  }, []);

  const handleEdgeClick = useCallback(
    (_event: MouseEvent, edge: Edge) => {
      if (!filteredData) {
        return;
      }
      const svcEdge = filteredData.edges.find((e) => e.id === edge.id);
      if (!svcEdge) {
        return;
      }
      const srcParsed = filteredData.nodes.find((n) => n.id === svcEdge.source)?.parsed ?? parseAppId(svcEdge.source);
      const tgtParsed = filteredData.nodes.find((n) => n.id === svcEdge.target)?.parsed ?? parseAppId(svcEdge.target);
      setSelectedEdge({
        source: svcEdge.source,
        target: svcEdge.target,
        sourceLabel: formatNodeLabel(srcParsed, resolvedOptions.labelMode),
        targetLabel: formatNodeLabel(tgtParsed, resolvedOptions.labelMode),
        edge: svcEdge,
        sourceAppId: svcEdge.source,
        targetAppId: svcEdge.target,
      });
      setSelectedNode(null);
      setTraceFilter('all');
      setHighlightedNodeId(null);
    },
    [filteredData, resolvedOptions.labelMode]
  );

  const handleNodeClick = useCallback(
    (_event: MouseEvent, node: Node) => {
      if (!filteredData || !containerDataFiltered || node.type === 'namespaceGroup') {
        return;
      }
      const isToggleOff = highlightedNodeId === node.id;
      if (isToggleOff) {
        setHighlightedNodeId(null);
        setSelectedNode(null);
        setSelectedEdge(null);
        return;
      }

      setHighlightedNodeId(node.id);
      setSelectedEdge(null);

      const svcNode = filteredData.nodes.find((n) => n.id === node.id);
      if (!svcNode) {
        return;
      }
      const outgoing = filteredData.edges.filter((e) => e.source === node.id);
      const outgoingLabels = outgoing.map((e) => {
        const parsed = filteredData.nodes.find((n) => n.id === e.target)?.parsed ?? parseAppId(e.target);
        return formatNodeLabel(parsed, resolvedOptions.labelMode);
      });
      const label = formatNodeLabel(svcNode.parsed, resolvedOptions.labelMode);

      let internalSamePodEdgesHidden: number | undefined;
      const traceServiceNames = groupByPod
        ? listContainerAppIdsForPod(svcNode.id, containerDataFiltered)
        : undefined;
      const childContainers = groupByPod
        ? getChildContainerNodesForPod(svcNode.id, containerDataFiltered)
        : undefined;
      if (groupByPod) {
        internalSamePodEdgesHidden = containerDataFiltered.edges.filter(
          (e) => podId(e.source) === podId(e.target) && podId(e.source) === svcNode.id
        ).length;
      }

      setSelectedNode({
        id: node.id,
        label,
        node: svcNode,
        outgoingEdges: outgoing,
        outgoingLabels,
        traceServiceNames: traceServiceNames && traceServiceNames.length > 0 ? traceServiceNames : undefined,
        childContainers,
        internalSamePodEdgesHidden,
      });
      setTraceFilter('all');
    },
    [
      filteredData,
      containerDataFiltered,
      resolvedOptions.labelMode,
      groupByPod,
      highlightedNodeId,
    ]
  );

  const handlePaneClick = useCallback(() => {
    setHighlightedNodeId(null);
    setSelectedNode(null);
    setSelectedEdge(null);
  }, []);

  const handleCloseSidebar = useCallback(() => {
    setSelectedEdge(null);
    setSelectedNode(null);
  }, []);

  const clearSelectionAndRefit = useCallback(() => {
    setSelectedEdge(null);
    setSelectedNode(null);
    setHighlightedNodeId(null);
    hasInitialFit.current = false;
  }, []);

  const styledEdges = useMemo(() => {
    if (!layout) {
      return [];
    }
    if (!highlightedNodeId) {
      return layout.rfEdges;
    }
    return layout.rfEdges.map((e) => {
      const connected = e.source === highlightedNodeId || e.target === highlightedNodeId;
      return {
        ...e,
        style: {
          ...e.style,
          opacity: connected ? 1 : 0.12,
        },
      };
    });
  }, [layout, highlightedNodeId]);

  const styledNodes = useMemo(() => {
    if (!layout) {
      return [];
    }
    if (!highlightedNodeId) {
      return layout.rfNodes;
    }
    const connectedIds = new Set<string>();
    connectedIds.add(highlightedNodeId);
    for (const e of layout.rfEdges) {
      if (e.source === highlightedNodeId) {
        connectedIds.add(e.target);
      }
      if (e.target === highlightedNodeId) {
        connectedIds.add(e.source);
      }
    }
    return layout.rfNodes.map((n) => {
      if (n.type === 'namespaceGroup') {
        return n;
      }
      const connected = connectedIds.has(n.id);
      return {
        ...n,
        style: {
          ...n.style,
          opacity: connected ? 1 : 0.25,
        },
      };
    });
  }, [layout, highlightedNodeId]);

  const namespaces = dataWithFriendly?.namespaces ?? [];

  const filterBars = (
    <MapFiltersBar
      groupByPod={groupByPod}
      hideWeakEdges={hideWeakEdges}
      onGroupByPodChange={(next) => {
        setGroupByPod(next);
        clearSelectionAndRefit();
      }}
      onHideWeakEdgesChange={(next) => {
        setHideWeakEdges(next);
        clearSelectionAndRefit();
      }}
      namespaces={namespaces}
      nsSelected={nsPick}
      onNsChange={(next) => {
        setNsPick(next);
        clearSelectionAndRefit();
      }}
      podNameFilter={podFilterInput}
      onPodNameFilterChange={setPodFilterInput}
    />
  );

  if (loading) {
    return (
      <div className={st.root} style={{ width, height }}>
        {filterBars}
        <div className={st.loading}>Loading service map...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={st.root} style={{ width, height }}>
        {filterBars}
        <div className={st.error}>{error}</div>
      </div>
    );
  }

  if (!filteredData || filteredData.nodes.length === 0) {
    const unfilteredCount = dataWithFriendly?.nodes.length ?? 0;
    const filterExcludesAll = nsPick.size > 0 && unfilteredCount > 0;
    const podFilterActive = podNameFilterApplied.trim().length > 0;
    const podFilterExcludesAll = podFilterActive && unfilteredCount > 0 && !filterExcludesAll;
    const emptyMsg = filterExcludesAll
      ? 'No services match the selected namespace filter. Click "All" to show every namespace.'
      : podFilterExcludesAll
        ? 'No workloads match this pod name filter. Clear the Pod field or try another substring.'
        : 'No service data to display. If metrics exist in Explore, try Namespaces → All and confirm the panel Prometheus datasource matches VM.';
    return (
      <div className={st.root} style={{ width, height }}>
        {filterBars}
        <div className={st.loading}>{emptyMsg}</div>
      </div>
    );
  }

  if (!layout) {
    if (layoutLoading) {
      return (
        <div className={st.root} style={{ width, height }}>
          {filterBars}
          <div className={st.loading}>Computing layout…</div>
        </div>
      );
    }
    return (
      <div className={st.root} style={{ width, height }}>
        {filterBars}
        <div className={st.error}>Graph layout could not be computed. Try refreshing the dashboard.</div>
      </div>
    );
  }

  return (
    <div className={st.root} style={{ width, height }}>
      <ArrowMarkers />
      {filterBars}
      <div className={st.topArea}>
        <div className={st.graphArea}>
          <ReactFlow
            nodes={styledNodes}
            edges={styledEdges}
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            onEdgeClick={handleEdgeClick}
            onNodeClick={handleNodeClick}
            onPaneClick={handlePaneClick}
            onInit={handleInit}
            minZoom={0.1}
            maxZoom={3}
            nodesFocusable={false}
            autoPanOnNodeFocus={false}
            proOptions={{ hideAttribution: true }}
            defaultEdgeOptions={{ type: 'serviceEdge' }}
          >
            <Background variant={BackgroundVariant.Dots} gap={24} size={1} color="#181830" />
            <Controls
              showInteractive={false}
              style={{ background: '#1a1a2e', borderColor: '#3a3a5a', borderRadius: 8 }}
            />
          </ReactFlow>
          {layoutLoading && (
            <div className={st.layoutOverlay}>Updating layout…</div>
          )}
        </div>
        {selectedEdge && (
          <EdgeDetailSidebar
            edge={selectedEdge}
            options={resolvedOptions}
            onClose={handleCloseSidebar}
            rpsSeries={rpsSeries}
            latSeries={latSeries}
          />
        )}
        {selectedNode && !selectedEdge && (
          <NodeDetailSidebar
            node={selectedNode}
            onClose={handleCloseSidebar}
          />
        )}
      </div>
      <div className={st.bottomArea}>
        <TraceTable
          traces={traces}
          loading={tracesLoading}
          error={tracesError}
          selectedEdge={selectedEdge}
          selectedNodeLabel={selectedNode?.label}
          filter={traceFilter}
          onFilterChange={setTraceFilter}
          timeRange={timeRange}
          tracesDashboardUid={resolvedOptions.tracesDashboardUid}
          tracesViewPanel={resolvedOptions.tracesViewPanel}
        />
      </div>
    </div>
  );
}
