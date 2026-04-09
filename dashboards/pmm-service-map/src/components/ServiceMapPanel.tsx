import { useCallback, useMemo, useRef, useState, type MouseEvent } from 'react';
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
import { NamespaceFilter } from './NamespaceFilter';
import { parseAppId, formatNodeLabel } from '../data/parseAppId';
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
  `,
  loading: css`
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: #666;
    font-size: 14px;
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
    }),
    [options, namespaceRenameMap]
  );

  const { data, loading, error } = useServiceMapData(resolvedOptions, timeRange);

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

  const filteredData = useMemo(() => {
    if (!dataWithFriendly || nsPick.size === 0) {
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
    return {
      nodes: allNodes,
      edges: filteredEdges,
      namespaces: dataWithFriendly.namespaces,
    };
  }, [dataWithFriendly, nsPick]);

  const { layout, layoutLoading } = useGraphLayout(filteredData, resolvedOptions);

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
      if (!filteredData || node.type === 'namespaceGroup') {
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
      setSelectedNode({
        id: node.id,
        label: formatNodeLabel(svcNode.parsed, resolvedOptions.labelMode),
        node: svcNode,
        outgoingEdges: outgoing,
        outgoingLabels,
      });
      setTraceFilter('all');
    },
    [filteredData, resolvedOptions.labelMode, highlightedNodeId]
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

  if (loading || layoutLoading) {
    return (
      <div className={st.root} style={{ width, height }}>
        <div className={st.loading}>Loading service map...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={st.root} style={{ width, height }}>
        <div className={st.error}>{error}</div>
      </div>
    );
  }

  if (!layout || !filteredData || filteredData.nodes.length === 0) {
    return (
      <div className={st.root} style={{ width, height }}>
        <div className={st.loading}>No service data available. Check that recording rules are active.</div>
      </div>
    );
  }

  const namespaces = dataWithFriendly?.namespaces ?? [];

  return (
    <div className={st.root} style={{ width, height }}>
      <ArrowMarkers />
      {namespaces.length >= 1 && (
        <NamespaceFilter
          namespaces={namespaces}
          selected={nsPick}
          onChange={(next) => {
            setNsPick(next);
            setSelectedEdge(null);
            setSelectedNode(null);
            setHighlightedNodeId(null);
            hasInitialFit.current = false;
          }}
        />
      )}
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
            proOptions={{ hideAttribution: true }}
            defaultEdgeOptions={{ type: 'serviceEdge' }}
          >
            <Background variant={BackgroundVariant.Dots} gap={24} size={1} color="#181830" />
            <Controls
              showInteractive={false}
              style={{ background: '#1a1a2e', borderColor: '#3a3a5a', borderRadius: 8 }}
            />
          </ReactFlow>
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
