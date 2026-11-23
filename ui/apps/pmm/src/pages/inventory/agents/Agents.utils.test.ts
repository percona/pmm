import { describe, expect, it } from 'vitest';
import { AgentStatus, AgentType } from 'types/agent.types';
import {
  getAgentStatusColor,
  getAgentStatusText,
  getAgentTypeName,
} from './Agents.utils';

describe('Agents utils', () => {
  it.each([
    [AgentStatus.running, 'Running', 'success'],
    [AgentStatus.starting, 'Starting', 'success'],
    [AgentStatus.done, 'Done', 'default'],
    [AgentStatus.initializationError, 'Initialization error', 'error'],
    [AgentStatus.waiting, 'Waiting', 'error'],
    [AgentStatus.unknown, 'Unknown', 'error'],
  ])('shows %s as %s in %s', (status, text, color) => {
    expect(getAgentStatusText(status)).toBe(text);
    expect(getAgentStatusColor(status)).toBe(color);
  });

  it('names known agent types and humanizes unknown ones', () => {
    expect(getAgentTypeName(AgentType.qanMysqlSlowlogAgent)).toBe(
      'QAN MySQL Slow Query Log agent'
    );
    expect(getAgentTypeName('future_agent-type' as AgentType)).toBe(
      'Future agent type'
    );
  });
});
