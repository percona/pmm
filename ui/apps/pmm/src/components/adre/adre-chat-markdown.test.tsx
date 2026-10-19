import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import Markdown from 'react-markdown';
import { TestWrapper } from 'utils/testWrapper';
import { getMarkdownComponents } from './adre-chat-markdown.helpers';

const content =
  'Run this:\n\n```sql\nSELECT pg_reload_conf();\nSELECT 1;\n```\n';

describe('chat markdown code blocks', () => {
  it('names the language and copies the code', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    render(
      <TestWrapper>
        <Markdown components={getMarkdownComponents(content)}>
          {content}
        </Markdown>
      </TestWrapper>
    );
    expect(screen.getByText('sql')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Copy code' }));
    expect(writeText).toHaveBeenCalledWith(
      'SELECT pg_reload_conf();\nSELECT 1;\n'
    );
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Copied' })).toBeInTheDocument()
    );
  });
});
