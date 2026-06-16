import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import Code from './code';
import CodeBlock from './code-block';

describe('Code', () => {
  it('renders a semantic <code> element with its content', () => {
    render(<Code>npm install</Code>);
    const el = screen.getByText('npm install');
    expect(el.tagName).toBe('CODE');
  });

  it('forwards extra props such as data-testid', () => {
    render(<Code data-testid="inline-code">x</Code>);
    expect(screen.getByTestId('inline-code')).toBeInTheDocument();
  });
});

describe('CodeBlock', () => {
  it('renders a semantic <pre><code> structure', () => {
    render(<CodeBlock>helm install percona</CodeBlock>);
    const code = screen.getByText('helm install percona');
    expect(code.tagName).toBe('CODE');
    expect(code.parentElement?.tagName).toBe('PRE');
  });

  it('does not render a copy button by default', () => {
    render(<CodeBlock>echo hi</CodeBlock>);
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('renders a copy button when copyable and copies the content', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    render(<CodeBlock copyable>echo hi</CodeBlock>);
    const button = screen.getByRole('button');
    fireEvent.click(button);

    await waitFor(() => expect(writeText).toHaveBeenCalledWith('echo hi'));
  });

  it('copies the value prop when provided instead of children', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    render(
      <CodeBlock copyable value="raw-command">
        <span>pretty command</span>
      </CodeBlock>
    );
    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => expect(writeText).toHaveBeenCalledWith('raw-command'));
  });

  it('renders plain text when no language is provided', () => {
    render(<CodeBlock>SELECT * FROM users;</CodeBlock>);
    expect(screen.getByText('SELECT * FROM users;')).toBeInTheDocument();
  });

  it('syntax-highlights into tokens when a language is provided', async () => {
    render(<CodeBlock language="sql">SELECT * FROM users;</CodeBlock>);

    // After the highlighter lazy-loads, content is split into per-token spans.
    const keyword = await screen.findByText('SELECT');
    expect(keyword.tagName).toBe('SPAN');
    expect(keyword.closest('pre')).toBeInTheDocument();
  });

  it('keeps copying the raw source even when highlighted', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    render(
      <CodeBlock language="sql" copyable>
        SELECT 1;
      </CodeBlock>
    );
    await screen.findByText('SELECT');
    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => expect(writeText).toHaveBeenCalledWith('SELECT 1;'));
  });
});
