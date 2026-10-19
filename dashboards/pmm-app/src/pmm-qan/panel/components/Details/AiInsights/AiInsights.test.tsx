import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { AiInsights } from './AiInsights';
import { fetchQanInsightsCache } from './AiInsights.service';

jest.mock('shared/components/helpers/notification-manager');
jest.mock('shared/core', () => ({ SERVICE_ID_PREFIX: '/service_id/' }));
jest.mock('./AiInsights.service', () => ({
  fetchQanInsights: jest.fn(),
  fetchQanInsightsCache: jest.fn(),
  createServiceNowFromQanInsights: jest.fn(),
}));

const analysis = [
  '## Summary',
  '',
  'Each call reads all of `orders`.',
  '',
  '| Table | Seq scans |',
  '|---|---:|',
  '| orders | 83,799 |',
  '',
  '1. Check the plan:',
  '',
  '```sql',
  'EXPLAIN SELECT 1;',
  '```',
].join('\n');

describe('AiInsights', () => {
  it('renders the answer like the Assistant and copies its code', async () => {
    (fetchQanInsightsCache as jest.Mock).mockResolvedValue({ analysis, created_at: '2026-10-19T11:58:00Z' });
    const writeText = jest.fn().mockResolvedValue(undefined);

    Object.assign(navigator, { clipboard: { writeText } });

    render(
      <AiInsights
        queryId="q1"
        from="now-12h"
        to="now"
        examplesLoading={false}
        examples={[{ service_id: '/service_id/s1', example: 'SELECT 1' } as never]}
      />,
    );

    expect(await screen.findByRole('heading', { name: 'Summary' })).toBeInTheDocument();
    expect(screen.getByRole('table')).toHaveTextContent('83,799');
    expect(screen.queryByText(/\|---/)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Copy code' }));

    expect(writeText).toHaveBeenCalledWith('EXPLAIN SELECT 1;');
    expect(await screen.findByText('Copied')).toBeInTheDocument();
  });
});
