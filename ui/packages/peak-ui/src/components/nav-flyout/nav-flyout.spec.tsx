import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import Collapse from '@mui/material/Collapse';
import IconButton from '@mui/material/IconButton';
import ListItem from '@mui/material/ListItem';
import { describe, expect, it, vi } from 'vitest';
import { getThemeOptions } from '../../design';
import NavItem from '../nav-item';
import NavFlyout from './nav-flyout';

const renderFlyout = () => {
  const anchor = document.createElement('button');
  const onClose = vi.fn();

  document.body.appendChild(anchor);
  render(
    <ThemeProvider theme={createTheme(getThemeOptions('pmm')('light'))}>
      <NavFlyout id="inventory-panel" label="Inventory" anchorEl={anchor} onClose={onClose}>
        <ListItem disablePadding>
          <NavItem text="Services" href="#services" />
          <IconButton aria-label="Add a service">+</IconButton>
        </ListItem>
        <ListItem disablePadding>
          <NavItem text="Nodes" href="#nodes" selected />
        </ListItem>
        <Collapse in={false}>
          <ListItem disablePadding>
            <NavItem text="Agents" href="#agents" />
          </ListItem>
        </Collapse>
      </NavFlyout>
    </ThemeProvider>
  );

  return { onClose };
};

describe('NavFlyout', () => {
  it('is a navigation landmark named by its heading', () => {
    renderFlyout();

    const panel = screen.getByRole('navigation', { name: 'Inventory' });
    expect(panel).toHaveTextContent('Inventory');
    expect(screen.getByText('Inventory')).toHaveClass('MuiTypography-sectionHeading');
  });

  it('moves focus to the current page as it opens', async () => {
    renderFlyout();

    await waitFor(() => expect(screen.getByRole('link', { name: 'Nodes' })).toHaveFocus());
  });

  it('moves between entries with the arrow keys, Home and End, skipping a closed group', () => {
    renderFlyout();
    const services = screen.getByRole('link', { name: 'Services' });
    const nodes = screen.getByRole('link', { name: 'Nodes' });

    services.focus();
    fireEvent.keyDown(services, { key: 'ArrowDown' });
    expect(nodes).toHaveFocus();

    fireEvent.keyDown(nodes, { key: 'ArrowDown' });
    expect(services).toHaveFocus();

    fireEvent.keyDown(services, { key: 'ArrowUp' });
    expect(nodes).toHaveFocus();

    fireEvent.keyDown(nodes, { key: 'Home' });
    expect(services).toHaveFocus();

    fireEvent.keyDown(services, { key: 'End' });
    expect(nodes).toHaveFocus();
  });

  it('moves on from the row of a focused action', () => {
    renderFlyout();
    const action = screen.getByRole('button', { name: 'Add a service' });

    action.focus();
    fireEvent.keyDown(action, { key: 'ArrowDown' });

    expect(screen.getByRole('link', { name: 'Nodes' })).toHaveFocus();
  });

  it('closes on Escape', () => {
    const { onClose } = renderFlyout();

    fireEvent.keyDown(screen.getByRole('link', { name: 'Services' }), { key: 'Escape' });

    expect(onClose).toHaveBeenCalled();
  });
});
