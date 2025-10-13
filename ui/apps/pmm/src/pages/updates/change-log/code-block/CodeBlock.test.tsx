import { render, screen } from '@testing-library/react';
import { CodeBlock } from './CodeBlock';

describe('CodeBlock', () => {
  it('shows inline if code is single line', () => {
    const code = `This is a single line`;
    render(<CodeBlock>{code}</CodeBlock>);

    expect(screen.getByRole('paragraph')).toHaveStyle({
      display: 'inline-block',
    });
  });

  it('shows correctly for multiline code', () => {
    const code = `This is line 1\nThis is line 2`;
    render(<CodeBlock>{code}</CodeBlock>);

    expect(screen.getByRole('paragraph')).not.toHaveStyle({
      display: 'inline-block',
    });
  });
});
