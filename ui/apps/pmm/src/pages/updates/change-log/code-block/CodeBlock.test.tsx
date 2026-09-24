import { render, screen } from '@testing-library/react';
import { CodeBlock } from './CodeBlock';

describe('CodeBlock', () => {
  it('renders single line code as inline <code>', () => {
    const code = `This is a single line`;
    render(<CodeBlock>{code}</CodeBlock>);

    const element = screen.getByText(code);
    expect(element.tagName).toBe('CODE');
  });

  it('renders multiline code as a <pre> block', () => {
    const code = `This is line 1\nThis is line 2`;
    const { container } = render(<CodeBlock>{code}</CodeBlock>);

    expect(container.querySelector('pre')).toBeInTheDocument();
  });

  it('renders single line code with a language as a <pre> block', () => {
    const code = `SELECT 1`;
    const { container } = render(
      <CodeBlock className="language-sql">{code}</CodeBlock>
    );

    expect(container.querySelector('pre')).toBeInTheDocument();
  });
});
