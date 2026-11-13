import React from 'react';
import { render, screen } from '@testing-library/react';
import { Highlight } from './Highlight';

jest.mock('highlight.js', () => ({
  highlightElement: jest.fn(),
}));

describe('Highlight::', () => {
  it('renders children correctly', () => {
    render(
      <Highlight language="sql">
        <span>Test children</span>
      </Highlight>,
    );

    expect(screen.getByTestId('highlight-code').textContent).toEqual('Test children');
  });

  it('renders correct language', () => {
    render(
      <Highlight language="sql">
        <span>Test children</span>
      </Highlight>,
    );

    expect(screen.getByTestId('highlight-code').className).toEqual('language-sql');
  });

  it('follows the page theme by default', () => {
    render(<Highlight language="sql">SELECT 1</Highlight>);

    expect(screen.getByTestId('highlight-pre')).toHaveStyle({ backgroundColor: '#181b1f' });
  });

  it('renders in the given theme mode', () => {
    render(
      <Highlight language="sql" mode="light">
        SELECT 1
      </Highlight>,
    );

    expect(screen.getByTestId('highlight-pre')).toHaveStyle({ backgroundColor: '#ffffff' });
  });

  it('uses syntax colours that suit the theme mode', () => {
    const { rerender } = render(
      <Highlight language="sql" mode="light">
        <span className="hljs-string">abc</span>
      </Highlight>,
    );

    expect(screen.getByText('abc')).toHaveStyle({ color: '#032f62' });

    rerender(
      <Highlight language="sql" mode="dark">
        <span className="hljs-string">abc</span>
      </Highlight>,
    );

    expect(screen.getByText('abc')).toHaveStyle({ color: '#a5c261' });
  });
});
