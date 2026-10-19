import { screen } from '@testing-library/react';
import { renderHub } from '../../testUtils';
import { testDoor } from '../hub.testFixtures';
import { DoorButton, DoorLink } from './DoorLink';

describe('DoorLink', () => {
  it('opens the door and names it with its hidden context', () => {
    renderHub(
      <DoorLink
        door={testDoor('40s', '/graph/d/replset?var-rs=orders', {
          context: 'for orders-shard1',
        })}
      />
    );

    const link = screen.getByRole('link', { name: '40s for orders-shard1' });

    expect(link).toHaveAttribute('href', '/graph/d/replset?var-rs=orders');
    expect(link).toHaveTextContent(/^40s/);
  });

  it('shows visible children instead of the label', () => {
    renderHub(
      <DoorLink door={testDoor('billing', '/graph/d/summary')}>
        Billing cluster
      </DoorLink>
    );

    expect(
      screen.getByRole('link', { name: 'Billing cluster' })
    ).toBeInTheDocument();
  });
});

describe('DoorButton', () => {
  it('carries the Preview chip of an Operations door in its name', () => {
    renderHub(
      <DoorButton
        door={testDoor('Install MongoDB', '/operations/nodes', {
          preview: 'developer',
        })}
      />
    );

    const button = screen.getByRole('link', {
      name: 'Install MongoDB Developer preview',
    });

    expect(button).toHaveAttribute('href', '/operations/nodes');
    expect(screen.getByTestId('preview-chip')).toBeInTheDocument();
  });

  it('keeps the visible label at the start of the name', () => {
    renderHub(
      <DoorButton
        door={testDoor('Open ReplSet summary', '/graph/d/replset', {
          context: 'for billing',
        })}
      />
    );

    expect(
      screen.getByRole('link', { name: 'Open ReplSet summary for billing' })
    ).toBeInTheDocument();
  });
});
