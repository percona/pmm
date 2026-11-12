import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import CodeBlock from './code-block';

describe('CodeBlock', () => {
  it('renders a semantic <pre><code> structure', () => {
    render(<CodeBlock content="helm install percona" />);
    const code = screen.getByText('helm install percona');
    expect(code.tagName).toBe('CODE');
    expect(code.parentElement?.tagName).toBe('PRE');
  });

  it('does not render a copy button by default', () => {
    render(<CodeBlock content="echo hi" />);
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('renders a copy button when copyable and copies the content', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    render(<CodeBlock content="echo hi" copyable />);
    const button = screen.getByRole('button');
    fireEvent.click(button);

    await waitFor(() => expect(writeText).toHaveBeenCalledWith('echo hi'));
  });

  it('copies the value prop when provided instead of the rendered content', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    render(<CodeBlock copyable value="raw-command" content={<span>pretty command</span>} />);
    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => expect(writeText).toHaveBeenCalledWith('raw-command'));
  });

  it('renders plain text when no language is provided', () => {
    render(<CodeBlock content="SELECT * FROM users;" />);
    expect(screen.getByText('SELECT * FROM users;')).toBeInTheDocument();
  });

  it('syntax-highlights into tokens when a language is provided', async () => {
    render(<CodeBlock language="sql" content="SELECT * FROM users;" />);

    // After the highlighter lazy-loads, content is split into per-token spans.
    const keyword = await screen.findByText('SELECT');
    expect(keyword.tagName).toBe('SPAN');
    expect(keyword.closest('pre')).toBeInTheDocument();
  });

  it('keeps copying the raw source even when highlighted', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });

    render(<CodeBlock language="sql" copyable content="SELECT 1;" />);
    await screen.findByText('SELECT');
    fireEvent.click(screen.getByRole('button'));

    await waitFor(() => expect(writeText).toHaveBeenCalledWith('SELECT 1;'));
  });

  it('follows the color mode: light and dark yield different token colors', async () => {
    const { unmount } = render(
      <ThemeProvider theme={createTheme({ palette: { mode: 'light' } })}>
        <CodeBlock language="sql" content="SELECT 1;" />
      </ThemeProvider>
    );
    const lightColor = (await screen.findByText('SELECT')).style.color;
    unmount();

    render(
      <ThemeProvider theme={createTheme({ palette: { mode: 'dark' } })}>
        <CodeBlock language="sql" content="SELECT 1;" />
      </ThemeProvider>
    );
    const darkColor = (await screen.findByText('SELECT')).style.color;

    expect(lightColor).toBeTruthy();
    expect(darkColor).toBeTruthy();
    expect(lightColor).not.toBe(darkColor);
  });
});
