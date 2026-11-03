import { GetHANodeResponse, NodeRole } from 'types/ha.types';
import { getHAHealth } from './ha.utils';

const expectedNodes = 3;

describe('ha.utils', () => {
  it('should return "healthy" if all alive', () => {
    const nodes: GetHANodeResponse[] = [
      { nodeName: 'pmm-ha-1', role: NodeRole.follower, status: 'alive' },
      { nodeName: 'pmm-ha-0', role: NodeRole.leader, status: 'alive' },
      { nodeName: 'pmm-ha-2', role: NodeRole.follower, status: 'alive' },
    ];

    const health = getHAHealth(nodes, expectedNodes);

    expect(health).toBe('healthy');
  });

  it('should return "down" if all dead', () => {
    const nodes: GetHANodeResponse[] = [
      { nodeName: 'pmm-ha-1', role: NodeRole.follower, status: 'dead' },
      { nodeName: 'pmm-ha-0', role: NodeRole.leader, status: 'dead' },
      { nodeName: 'pmm-ha-2', role: NodeRole.follower, status: 'dead' },
    ];

    const health = getHAHealth(nodes, expectedNodes);

    expect(health).toBe('unreachable');
  });

  it('should return "down" if all suspect', () => {
    const nodes: GetHANodeResponse[] = [
      { nodeName: 'pmm-ha-1', role: NodeRole.follower, status: 'suspect' },
      { nodeName: 'pmm-ha-0', role: NodeRole.leader, status: 'suspect' },
      { nodeName: 'pmm-ha-2', role: NodeRole.follower, status: 'suspect' },
    ];

    const health = getHAHealth(nodes, expectedNodes);

    expect(health).toBe('unreachable');
  });

  it('should return "down" if all left', () => {
    const nodes: GetHANodeResponse[] = [
      { nodeName: 'pmm-ha-1', role: NodeRole.follower, status: 'left' },
      { nodeName: 'pmm-ha-0', role: NodeRole.leader, status: 'left' },
      { nodeName: 'pmm-ha-2', role: NodeRole.follower, status: 'left' },
    ];

    const health = getHAHealth(nodes, expectedNodes);

    expect(health).toBe('unreachable');
  });

  it('should return "down" if all unknown', () => {
    const nodes: GetHANodeResponse[] = [
      { nodeName: 'pmm-ha-1', role: NodeRole.follower, status: 'unknown' },
      { nodeName: 'pmm-ha-0', role: NodeRole.leader, status: 'unknown' },
      { nodeName: 'pmm-ha-2', role: NodeRole.follower, status: 'unknown' },
    ];

    const health = getHAHealth(nodes, expectedNodes);

    expect(health).toBe('unreachable');
  });

  it('should return "degraded" if not alive <= 1/3', () => {
    const nodes: GetHANodeResponse[] = [
      { nodeName: 'pmm-ha-1', role: NodeRole.follower, status: 'alive' },
      { nodeName: 'pmm-ha-0', role: NodeRole.leader, status: 'alive' },
      { nodeName: 'pmm-ha-2', role: NodeRole.follower, status: 'dead' },
    ];

    const health = getHAHealth(nodes, expectedNodes);

    expect(health).toBe('degraded');
  });

  it('should return "critical" if not alive <= 2/3', () => {
    const nodes: GetHANodeResponse[] = [
      { nodeName: 'pmm-ha-1', role: NodeRole.follower, status: 'alive' },
      { nodeName: 'pmm-ha-0', role: NodeRole.leader, status: 'dead' },
      { nodeName: 'pmm-ha-2', role: NodeRole.follower, status: 'dead' },
    ];

    const health = getHAHealth(nodes, expectedNodes);

    expect(health).toBe('critical');
  });
});
