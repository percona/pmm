import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { SeenBefore } from './SeenBefore';

describe('SeenBefore', () => {
  it('lists past incidents with why they match and how they ended', () => {
    render(
      <MemoryRouter>
        <SeenBefore
          items={[
            {
              id: 'i1',
              title: 'Alert: MySQL replication SQL thread stopped',
              status: 'completed',
              createdAt: '2026-10-16T13:00:00Z',
              summary: '',
              rootCauseSummary: 'The replica is writable.',
              resolutionSummary: 'Set read_only=1.',
              score: 9,
              reasons: ['Same alert', 'Same node'],
            },
            {
              id: 'i2',
              title: 'Chat: failed logins',
              status: 'open',
              createdAt: '2026-10-15T13:00:00Z',
              summary: 'shop_app used a wrong password.',
              rootCauseSummary: '',
              resolutionSummary: '',
              score: 2,
              reasons: ['Similar description'],
            },
          ]}
        />
      </MemoryRouter>
    );

    const section = screen.getByRole('region', { name: 'Seen before' });
    expect(section).toHaveTextContent('Same alert');
    expect(section).toHaveTextContent('Root cause: The replica is writable.');
    expect(section).toHaveTextContent('Resolution: Set read_only=1.');
    expect(section).toHaveTextContent(
      'Summary: shop_app used a wrong password.'
    );
    expect(
      screen.getByRole('link', { name: 'Chat: failed logins' })
    ).toHaveAttribute('href', '/investigations/i2');
  });

  it('shows nothing without past incidents', () => {
    const { container } = render(<SeenBefore items={[]} />);
    expect(container).toBeEmptyDOMElement();
  });
});
