import { createRef, type ReactNode } from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import Button from '@mui/material/Button';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import { describe, expect, it, vi } from 'vitest';
import { getThemeOptions } from '../../design';
import * as illustrations from '../illustrations';
import EmptyState, { EMPTY_STATE_ILLUSTRATIONS, EMPTY_STATE_SIZES } from './empty-state';

const renderInTheme = (node: ReactNode) =>
  render(
    <ThemeProvider theme={createTheme(getThemeOptions('base')('light'))}>{node}</ThemeProvider>
  );

describe('EmptyState', () => {
  it('renders the title, description and actions in that order', () => {
    const { container } = renderInTheme(
      <EmptyState
        title="No services yet"
        description="Services show up here once a database is monitored."
        actions={<Button>Add service</Button>}
      />
    );

    const title = screen.getByRole('heading', { name: 'No services yet' });
    const description = screen.getByText('Services show up here once a database is monitored.');
    const action = screen.getByRole('button', { name: 'Add service' });
    expect(title.compareDocumentPosition(description)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
    expect(description.compareDocumentPosition(action)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
    expect(container).toHaveTextContent(
      'No services yetServices show up here once a database is monitored.Add service'
    );
  });

  it('renders the title as an h3 by default and never as an h1 unless asked', () => {
    renderInTheme(<EmptyState title="No services yet" />);

    expect(screen.getByRole('heading', { level: 3, name: 'No services yet' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { level: 1 })).not.toBeInTheDocument();
  });

  it.each([
    ['h1', 1],
    ['h2', 2],
    ['h4', 4],
  ] as const)('renders the title as %s when titleComponent asks for it', (component, level) => {
    renderInTheme(<EmptyState title="Page not found" titleComponent={component} />);

    expect(screen.getByRole('heading', { level, name: 'Page not found' }).tagName).toBe(
      component.toUpperCase()
    );
  });

  it('hides the illustration from assistive technology', () => {
    const { container } = renderInTheme(
      <EmptyState title="No matching services" illustration="nothing-found" />
    );

    expect(container.querySelector('svg')).toHaveAttribute('aria-hidden', 'true');
    expect(screen.queryByRole('img')).not.toBeInTheDocument();
  });

  it('offers every Peak illustration by name', () => {
    const exported = Object.keys(illustrations).filter((name) => name.endsWith('Illustration'));

    expect(Object.values(EMPTY_STATE_ILLUSTRATIONS)).toHaveLength(exported.length);
    exported.forEach((name) => {
      expect(Object.values(EMPTY_STATE_ILLUSTRATIONS)).toContain(
        illustrations[name as keyof typeof illustrations]
      );
    });
  });

  it.each([
    ['small', '80px', '32px'],
    ['medium', '120px', '48px'],
  ] as const)('sizes a %s empty state', (size, illustration, padding) => {
    const { container } = renderInTheme(
      <EmptyState size={size} title="No services yet" illustration="radar" data-testid="empty" />
    );

    expect(container.querySelector('svg')).toHaveStyle({ fontSize: illustration });
    expect(screen.getByTestId('empty')).toHaveStyle({
      paddingTop: padding,
      paddingBottom: padding,
    });
    expect(EMPTY_STATE_SIZES[size].illustration).toBe(parseInt(illustration, 10));
  });

  it('renders the description as a block so it can hold more than text', () => {
    renderInTheme(
      <EmptyState
        title="No services yet"
        description={
          <ul data-testid="list">
            <li>Add a service</li>
          </ul>
        }
      />
    );

    const description = screen.getByTestId('list').parentElement;
    expect(description?.tagName).toBe('DIV');
    expect(description?.closest('p')).toBeNull();
  });

  it('renders no wrappers for the slots it is not given', () => {
    renderInTheme(<EmptyState title="No services yet" data-testid="empty" />);

    expect(screen.getByTestId('empty').children).toHaveLength(1);
  });

  it('lets an action be used', () => {
    const onClick = vi.fn();
    renderInTheme(
      <EmptyState
        title="No backups yet"
        actions={<Button onClick={onClick}>Set up a backup</Button>}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: 'Set up a backup' }));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('forwards the ref, passes other props to the root and lets sx win', () => {
    const ref = createRef<HTMLDivElement>();
    renderInTheme(
      <EmptyState
        ref={ref}
        title="No services yet"
        data-testid="empty"
        id="services-empty"
        sx={{ py: 1 }}
      />
    );

    const root = screen.getByTestId('empty');
    expect(ref.current).toBe(root);
    expect(root).toHaveAttribute('id', 'services-empty');
    expect(root).toHaveStyle({ paddingTop: '8px', display: 'flex' });
  });

  it('lets Box spacing props override the defaults', () => {
    renderInTheme(
      <EmptyState title="No services yet" data-testid="empty" py={0} gap={2} textAlign="left" />
    );

    expect(screen.getByTestId('empty')).toHaveStyle({
      paddingTop: '0px',
      paddingBottom: '0px',
      gap: '16px',
      textAlign: 'left',
    });
  });
});
