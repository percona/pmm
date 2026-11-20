import React from 'react';
import { render, screen } from '@testing-library/react';
import { createTheme, ThemeContext } from '@grafana/data';
import { Highlight } from './Highlight';

describe('Highlight::', () => {
  it('renders the code', () => {
    render(<Highlight language="sql">SELECT 1</Highlight>);

    expect(screen.getByTestId('highlight-code').textContent).toEqual('SELECT 1');
  });

  it('renders correct language', () => {
    render(<Highlight language="sql">SELECT 1</Highlight>);

    expect(screen.getByTestId('highlight-code').className).toEqual('language-sql');
  });

  it('follows the dark theme', () => {
    render(<Highlight language="sql">SELECT &apos;abc&apos;</Highlight>);

    expect(screen.getByTestId('highlight-pre')).toHaveStyle({ backgroundColor: '#181b1f' });
    expect(screen.getByText('SELECT')).toHaveStyle({ color: '#66d9ef' });
    expect(screen.getByText('\'abc\'')).toHaveStyle({ color: '#a6e22e' });
  });

  it('follows the light theme', () => {
    render(
      <ThemeContext.Provider value={createTheme({ colors: { mode: 'light' } })}>
        <Highlight language="sql">SELECT &apos;abc&apos;</Highlight>
      </ThemeContext.Provider>,
    );

    expect(screen.getByTestId('highlight-pre')).toHaveStyle({ backgroundColor: '#ffffff' });
    expect(screen.getByText('SELECT')).toHaveStyle({ color: 'rgb(12, 150, 155)' });
    expect(screen.getByText('\'abc\'')).toHaveStyle({ color: 'rgb(72, 118, 214)' });
  });
});
