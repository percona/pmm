import { describe, it, expect } from 'vitest';
import { splitEmailAddresses } from './Advisors.utils';

describe('splitEmailAddresses', () => {
  it('splits on commas, semicolons and whitespace', () => {
    expect(
      splitEmailAddresses(
        ' dba@example.com,oncall@example.com;\nops@example.com '
      )
    ).toEqual(['dba@example.com', 'oncall@example.com', 'ops@example.com']);
  });

  it('returns no address for blank input', () => {
    expect(splitEmailAddresses(' , ')).toEqual([]);
  });
});
