import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import Markdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import remarkGfm from 'remark-gfm';
import { TestWrapper } from 'utils/testWrapper';
import {
  getMarkdownComponents,
  remarkCodeMeta,
} from './adre-chat-markdown.helpers';

vi.mock('./AdreChart', () => ({
  default: ({ spec }: { spec: unknown }) => (
    <pre data-testid="chart-spec">{JSON.stringify(spec)}</pre>
  ),
}));

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

  it('numbers list items from where the list starts', () => {
    const list = '4. **Move to SCRAM.**\n5. Log the client address.\n';
    const { container } = render(
      <TestWrapper>
        <Markdown components={getMarkdownComponents(list)}>{list}</Markdown>
      </TestWrapper>
    );
    const items = container.querySelectorAll('ol > li');
    expect([...items].map((li) => li.getAttribute('data-n'))).toEqual([
      '4',
      '5',
    ]);
  });
});

describe('chat markdown charts', () => {
  it('draws a chart block as a chart, with its attributes, like the chat renders it', async () => {
    const answer =
      'CPU rose at 20:40.\n\n```chart ds="Metrics" title="CPU busy" from="2026-10-19T20:00:00Z" to="2026-10-19T21:00:00Z" unit=percentunit\n' +
      'sum by (node_name) (rate(node_cpu_seconds_total{mode!="idle"}[5m]))\n```\n';
    render(
      <TestWrapper>
        <Markdown
          remarkPlugins={[remarkGfm, remarkCodeMeta]}
          rehypePlugins={[rehypeRaw]}
          components={getMarkdownComponents(answer)}
        >
          {answer}
        </Markdown>
      </TestWrapper>
    );

    expect(
      JSON.parse((await screen.findByTestId('chart-spec')).textContent ?? '')
    ).toEqual({
      ds: 'Metrics',
      title: 'CPU busy',
      from: '2026-10-19T20:00:00Z',
      to: '2026-10-19T21:00:00Z',
      unit: 'percentunit',
      query:
        'sum by (node_name) (rate(node_cpu_seconds_total{mode!="idle"}[5m]))',
    });
    expect(screen.queryByRole('button', { name: 'Copy code' })).toBeNull();
  });
});

describe('chat markdown tables', () => {
  it('keeps header words whole', () => {
    const table = '| Service | Status |\n|---|---|\n| ps-1 | RUNNING |\n';
    render(
      <TestWrapper>
        <Markdown
          remarkPlugins={[remarkGfm]}
          components={getMarkdownComponents(table)}
        >
          {table}
        </Markdown>
      </TestWrapper>
    );
    expect(screen.getByRole('columnheader', { name: 'Status' })).toHaveStyle({
      whiteSpace: 'nowrap',
      wordBreak: 'normal',
    });
    expect(screen.getByRole('cell', { name: 'RUNNING' })).toHaveStyle({
      overflowWrap: 'break-word',
    });
  });
});

describe('chat markdown panels', () => {
  it('shows a panel image with its dashboard link as the live panel', () => {
    const blob = `https://pmm.test/v1/grafana/render/blob/${'a'.repeat(64)}.png`;
    const link =
      'https://pmm.test/graph/d/mysql-instance-summary?from=1&to=2&var-service_name=ps-1&viewPanel=8';
    const answer = `### Handlers\n![MySQL Handlers](${blob})\n[Open in Grafana](${link})\n`;
    render(
      <TestWrapper>
        <Markdown
          remarkPlugins={[remarkGfm, remarkCodeMeta]}
          rehypePlugins={[rehypeRaw]}
          components={getMarkdownComponents(answer)}
        >
          {answer}
        </Markdown>
      </TestWrapper>
    );

    expect(screen.getByTitle('MySQL Handlers')).toHaveAttribute(
      'src',
      '/graph/d-solo/mysql-instance-summary?from=1&to=2&var-service_name=ps-1&panelId=8&orgId=1&theme=dark&hideLogo=true'
    );
    expect(screen.queryByRole('img')).toBeNull();
    expect(
      screen.getAllByRole('link', { name: 'Open in Grafana' })
    ).toHaveLength(1);
  });
});
