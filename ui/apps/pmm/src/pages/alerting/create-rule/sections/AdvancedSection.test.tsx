import { fireEvent, render, screen } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import {
  Severity,
  Template,
  TemplateCategory,
  TemplateSource,
} from 'types/alert-templates.types';
import { AdvancedSection } from './AdvancedSection';

const MULTILINE_QUERY = [
  'sum by(service_name) (pg_stat_activity_count{datname!~"template.*"})',
  '/ on(service_name) pg_settings_max_connections',
  '* 100',
].join('\n');

const baseTemplate: Template = {
  name: 'pmm_node_high_cpu_load',
  summary: 'Node high CPU load',
  expr: '',
  params: [],
  for: '300s',
  severity: Severity.WARNING,
  labels: {},
  annotations: {},
  source: TemplateSource.BUILT_IN,
  yaml: '',
  category: TemplateCategory.NODE,
};

const multiExprTemplate: Template = {
  ...baseTemplate,
  queries: [{ refId: 'A', expr: MULTILINE_QUERY }],
  expressions: [
    { refId: 'C', type: 'math', expression: '$A > [[ .threshold ]]' },
  ],
  condition: 'C',
};

const singleExprTemplate: Template = {
  ...baseTemplate,
  name: 'pmm_mysql_down',
  summary: 'MySQL down',
  expr: 'mysql_up == 0',
};

const renderExpanded = (template: Template | null) => {
  const result = render(<AdvancedSection template={template} />, {
    wrapper: TestWrapper,
  });
  const section = screen.queryByTestId('alert-rule-advanced-section');
  if (section) {
    fireEvent.click(screen.getByRole('button', { name: /advanced details/i }));
  }
  return result;
};

describe('AdvancedSection', () => {
  it('renders nothing without a template', () => {
    const { container } = render(<AdvancedSection template={null} />, {
      wrapper: TestWrapper,
    });
    expect(container).toBeEmptyDOMElement();
  });

  it('is collapsed until toggled', () => {
    render(<AdvancedSection template={multiExprTemplate} />, {
      wrapper: TestWrapper,
    });
    expect(
      screen.getByRole('button', { name: /advanced details/i })
    ).toHaveAttribute('aria-expanded', 'false');
  });

  it('shows queries, expressions and the condition for a multi-expr template', () => {
    renderExpanded(multiExprTemplate);

    expect(screen.getByTestId('template-queries')).toBeInTheDocument();
    expect(screen.getByTestId('template-query-A')).toHaveTextContent(
      'pg_settings_max_connections'
    );

    expect(screen.getByTestId('template-expressions')).toBeInTheDocument();
    expect(screen.getByTestId('template-expression-C')).toHaveTextContent(
      '$A > [[ .threshold ]]'
    );

    expect(screen.getByTestId('template-condition')).toHaveTextContent('C');

    // The single-expression fallback must not appear alongside the queries.
    expect(screen.queryByTestId('template-expression')).not.toBeInTheDocument();
  });

  it('preserves newlines in a multi-line query', () => {
    renderExpanded(multiExprTemplate);
    expect(screen.getByTestId('template-query-A').textContent).toBe(
      MULTILINE_QUERY
    );
  });

  it('shows only the flat expression for a single-expr template', () => {
    renderExpanded(singleExprTemplate);

    expect(screen.getByTestId('template-expression')).toHaveTextContent(
      'mysql_up == 0'
    );
    expect(screen.queryByTestId('template-queries')).not.toBeInTheDocument();
    expect(
      screen.queryByTestId('template-expressions')
    ).not.toBeInTheDocument();
    expect(screen.queryByTestId('template-condition')).not.toBeInTheDocument();
  });

  it('shows the summary annotation as the rule alert', () => {
    renderExpanded({
      ...multiExprTemplate,
      annotations: { summary: 'Node high CPU load ({{ $labels.node_name }})' },
    });
    expect(screen.getByTestId('template-alert')).toHaveTextContent(
      'Node high CPU load ({{ $labels.node_name }})'
    );
  });

  it('omits the rule alert when there is no summary annotation', () => {
    renderExpanded(multiExprTemplate);
    expect(screen.queryByTestId('template-alert')).not.toBeInTheDocument();
  });
});
