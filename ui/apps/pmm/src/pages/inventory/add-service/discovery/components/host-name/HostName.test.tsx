import { render } from '@testing-library/react';
import { HostName } from './HostName';

describe('HostName', () => {
  it('keeps the host intact and lets it break only after a dot', () => {
    const host =
      'pmm-qa-rds-mysql-8-4.abcdefghijkl.eu-west-1.rds.amazonaws.com';
    const { container } = render(<HostName host={host} />);

    expect(container).toHaveTextContent(host);
    expect(container.querySelectorAll('wbr')).toHaveLength(5);
  });
});
