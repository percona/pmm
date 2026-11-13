import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { OmSectionTabs } from './OmSectionTabs';

const renderAt = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <OmSectionTabs />
    </MemoryRouter>
  );

describe('OmSectionTabs', () => {
  it('links to every section of the app', () => {
    renderAt('/operations');

    const sections = screen.getByRole('navigation', {
      name: 'Operations for MongoDB',
    });
    expect(sections).toHaveTextContent('FleetNodesAutomationsSettings');
    expect(screen.getByRole('tab', { name: 'Automations' })).toHaveAttribute(
      'href',
      '/operations/automations'
    );
  });

  it('marks the current section, also on a page below it', () => {
    renderAt('/operations/nodes/install');

    expect(screen.getByRole('tab', { name: 'Nodes' })).toHaveAttribute(
      'aria-current',
      'page'
    );
    expect(screen.getByRole('tab', { name: 'Fleet' })).not.toHaveAttribute(
      'aria-current'
    );
  });
});
