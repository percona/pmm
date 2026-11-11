import { render, screen } from '@testing-library/react';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import type { ReactElement } from 'react';
import { getThemeOptions } from '../../design';
import NavItem from './nav-item';

const renderInTheme = (ui: ReactElement) =>
  render(<ThemeProvider theme={createTheme(getThemeOptions('pmm')('light'))}>{ui}</ThemeProvider>);

it('marks the selected item as the current page', () => {
  renderInTheme(<NavItem text="Services" selected />);
  expect(screen.getByRole('button', { name: 'Services' })).toHaveAttribute('aria-current', 'page');
});

it('does not mark an unselected item as current', () => {
  renderInTheme(<NavItem text="Nodes" />);
  expect(screen.getByRole('button', { name: 'Nodes' })).not.toHaveAttribute('aria-current');
});

it('does not mark a group holding the selected item as current', () => {
  renderInTheme(<NavItem text="Inventory" hasSelectedChild />);
  expect(screen.getByRole('button', { name: 'Inventory' })).not.toHaveAttribute('aria-current');
});

it('lets the consumer override aria-current', () => {
  renderInTheme(<NavItem text="Inventory" selected aria-current="true" />);
  expect(screen.getByRole('button', { name: 'Inventory' })).toHaveAttribute('aria-current', 'true');
});
