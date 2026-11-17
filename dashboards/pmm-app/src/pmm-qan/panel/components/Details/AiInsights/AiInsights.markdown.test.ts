import { renderAnalysis } from './AiInsights.markdown';

const toDom = (md: string) => {
  const root = document.createElement('div');

  root.innerHTML = renderAnalysis(md);

  return root;
};

describe('renderAnalysis', () => {
  it('renders tables', () => {
    const root = toDom('| Table | Live tuples |\n|---|---:|\n| orders | 2,285,419 |');

    expect(root.querySelectorAll('th')).toHaveLength(2);
    expect(root.querySelector('td')?.textContent).toBe('orders');
    // Wide tables scroll on their own instead of widening the tab.
    expect(root.querySelector('table')?.parentElement?.className).toBe('ai-table');
  });

  it('numbers list items from where the list starts', () => {
    const root = toDom('1. Run the plan\n\n```sql\nSELECT 1;\n```\n\n2. Add an index\n3. Raise work_mem');

    const numbers = Array.from(root.querySelectorAll('ol > li')).map((li) => li.getAttribute('data-n'));

    expect(numbers).toEqual(['1', '2', '3']);
  });

  it('keeps a numbered item that follows a paragraph line, outside code', () => {
    const root = toDom([
      '2. Add an index:',
      '```sql',
      'CREATE INDEX i ON orders (status);',
      '```',
      '   It helps most if few rows match.',
      '3. Raise work_mem',
      '```text',
      'Sort',
      '1. not a list',
      '```',
    ].join('\n'));

    const numbers = Array.from(root.querySelectorAll('ol > li')).map((li) => li.getAttribute('data-n'));

    expect(numbers).toEqual(['2', '3']);
    expect(root.querySelector('.ai-code-lang + button')).not.toBeNull();
    expect(root.querySelectorAll('pre code')[1]?.textContent).toBe('Sort\n1. not a list');
  });

  it('gives code blocks a language, highlighting and a copy button', () => {
    const root = toDom('```sql\nSELECT status FROM orders;\n```');
    const block = root.querySelector('.ai-code');

    expect(block?.querySelector('.ai-code-lang')?.textContent).toBe('sql');
    expect(block?.querySelector('button[data-copy-code]')).not.toBeNull();
    expect(block?.querySelector('code .hljs-keyword')?.textContent).toBe('SELECT');
    expect(block?.querySelector('code')?.textContent).toBe('SELECT status FROM orders;');
  });

  it('drops scripts and inline styles from the answer', () => {
    const root = toDom('Hello <span style="position:fixed">there</span><script>alert(1)</script>');

    expect(root.querySelector('script')).toBeNull();
    expect(root.querySelector('[style]')).toBeNull();
    expect(root.textContent).toContain('there');
  });
});
