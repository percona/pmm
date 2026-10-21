import { useEffect, useState } from 'react';
import ELK, { ElkNode, ElkExtendedEdge } from 'elkjs/lib/elk.bundled.js';
import { type Node as RFNode, type Edge as RFEdge } from '@xyflow/react';
import { ServiceMapData, ServiceMapOptions } from '../../types';
import { formatNodeLabel } from '../../data/parseAppId';
import { HEALTH_COLORS, NODE_WIDTH, NODE_HEIGHT, GROUP_PADDING, EDGE_MIN_WIDTH, EDGE_MAX_WIDTH } from '../../constants';

const elk = new ELK();

function edgeWidth(rps: number): number {
  if (rps <= 0) {
    return EDGE_MIN_WIDTH;
  }
  return Math.max(EDGE_MIN_WIDTH, Math.min(EDGE_MAX_WIDTH, Math.log2(rps + 1) * 1.5));
}

export interface LayoutResult {
  rfNodes: RFNode[];
  rfEdges: RFEdge[];
}

export function useGraphLayout(
  data: ServiceMapData | null,
  options: ServiceMapOptions
): { layout: LayoutResult | null; layoutLoading: boolean } {
  const [layout, setLayout] = useState<LayoutResult | null>(null);
  const [layoutLoading, setLayoutLoading] = useState(false);

  useEffect(() => {
    if (!data || data.nodes.length === 0) {
      setLayout(null);
      return;
    }

    let cancelled = false;
    setLayoutLoading(true);

    async function computeLayout() {
      const nsRename = options.namespaceRenameMap ?? {};
      const groupByPod = options.groupByPod ?? true;

      // Group nodes by namespace
      const nsByNs = new Map<string, string[]>();
      for (const n of data!.nodes) {
        const ns = n.parsed.namespace || 'external';
        const list = nsByNs.get(ns) ?? [];
        list.push(n.id);
        nsByNs.set(ns, list);
      }

      const elkChildren: ElkNode[] = [];

      for (const [ns, members] of nsByNs.entries()) {
        const groupId = `ns-${ns}`;
        const children: ElkNode[] = members.map((id) => ({
          id,
          width: NODE_WIDTH,
          height: NODE_HEIGHT,
        }));

        elkChildren.push({
          id: groupId,
          children,
          layoutOptions: {
            'elk.padding': `[top=${GROUP_PADDING},left=${GROUP_PADDING},bottom=${GROUP_PADDING},right=${GROUP_PADDING}]`,
          },
        });
      }

      const elkEdges: ElkExtendedEdge[] = data!.edges.map((e) => ({
        id: e.id,
        sources: [e.source],
        targets: [e.target],
      }));

      const elkGraph: ElkNode = {
        id: 'root',
        children: elkChildren,
        edges: elkEdges,
        layoutOptions: {
          'elk.algorithm': 'layered',
          /** Prefer left-to-right layers; wider spacing reduces vertical stacking within a namespace. */
          'elk.direction': 'RIGHT',
          'elk.spacing.nodeNode': '48',
          'elk.layered.spacing.nodeNodeBetweenLayers': '110',
          'elk.layered.crossingMinimization.strategy': 'LAYER_SWEEP',
          'elk.hierarchyHandling': 'INCLUDE_CHILDREN',
        },
      };

      try {
        const result = await elk.layout(elkGraph);

        if (cancelled) {
          return;
        }

        const rfNodes: RFNode[] = [];
        const rfEdges: RFEdge[] = [];

        for (const group of result.children ?? []) {
          if (group.id.startsWith('ns-')) {
            const ns = group.id.slice(3);
            rfNodes.push({
              id: group.id,
              type: 'namespaceGroup',
              position: { x: group.x ?? 0, y: group.y ?? 0 },
              data: { label: nsRename[ns] || ns.toUpperCase() },
              draggable: false,
              selectable: false,
              focusable: false,
              style: {
                width: group.width,
                height: group.height,
                pointerEvents: 'none' as const,
              },
            });

            for (const child of group.children ?? []) {
              const svcNode = data!.nodes.find((n) => n.id === child.id);
              if (!svcNode) {
                continue;
              }
              const label = formatNodeLabel(svcNode.parsed, options.labelMode);
              rfNodes.push({
                id: child.id,
                type: 'serviceNode',
                position: { x: child.x ?? 0, y: child.y ?? 0 },
                parentId: group.id,
                extent: 'parent' as const,
                data: {
                  label,
                  rps: svcNode.rps,
                  errPct: svcNode.errPct,
                  p95Ms: svcNode.p95Ms,
                  health: svcNode.health,
                  fullId: svcNode.id,
                  namespace: svcNode.parsed.namespace,
                  bytesIn: svcNode.bytesIn,
                  bytesOut: svcNode.bytesOut,
                  groupByPod,
                  podChildContainerCount: svcNode.podChildContainerCount ?? 0,
                },
              });
            }
          }
        }

        for (const e of data!.edges) {
            const srcNode = data!.nodes.find((n) => n.id === e.source);
            const tgtNode = data!.nodes.find((n) => n.id === e.target);
            rfEdges.push({
              id: e.id,
              source: e.source,
              target: e.target,
              type: 'serviceEdge',
              data: {
                rps: e.rps,
                errPct: e.errPct,
                p95Ms: e.p95Ms,
                bytesIn: e.bytesIn,
                bytesOut: e.bytesOut,
                health: e.health,
                sourceLabel: srcNode ? formatNodeLabel(srcNode.parsed, options.labelMode) : e.source,
                targetLabel: tgtNode ? formatNodeLabel(tgtNode.parsed, options.labelMode) : e.target,
              },
            style: {
              strokeWidth: edgeWidth(e.rps),
              stroke: HEALTH_COLORS[e.health],
            },
          });
        }

        setLayout({ rfNodes, rfEdges });
      } catch {
        setLayout(null);
      } finally {
        if (!cancelled) {
          setLayoutLoading(false);
        }
      }
    }

    computeLayout();
    return () => {
      cancelled = true;
    };
  }, [data, options.labelMode, options.namespaceRenameMap, options.groupByPod]);

  return { layout, layoutLoading };
}
