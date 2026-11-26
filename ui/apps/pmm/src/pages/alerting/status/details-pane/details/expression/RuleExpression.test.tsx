import { render, screen } from '@testing-library/react';
import RuleExpression from './RuleExpression';

describe('RuleExpression', () => {
  it('shows the fallback until the rule definition is available', () => {
    render(<RuleExpression fallback="up == 0" />);

    expect(screen.getByText('up == 0')).toBeInTheDocument();
  });

  it('shows each step with its ref, kind and condition marker', () => {
    render(
      <RuleExpression
        fallback="unused"
        definition={{
          uid: 'rule',
          condition: 'B',
          data: [
            {
              refId: 'A',
              datasourceUid: 'clickhouse-uid',
              model: { refId: 'A', rawSql: 'SELECT 1' },
            },
            {
              refId: 'B',
              datasourceUid: '__expr__',
              model: {
                refId: 'B',
                type: 'reduce',
                expression: 'A',
                reducer: 'max',
              },
            },
          ],
        }}
      />
    );

    expect(screen.getByTestId('rule-step-A')).toHaveTextContent(
      'A · SQL query'
    );
    expect(screen.getByTestId('rule-step-A')).toHaveTextContent('SELECT 1');
    expect(screen.getByTestId('rule-step-B')).toHaveTextContent(
      'B · Reduce · condition'
    );
    expect(screen.getByTestId('rule-step-B')).toHaveTextContent('max($A)');
    expect(screen.queryByText('unused')).not.toBeInTheDocument();
  });
});
