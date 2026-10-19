import { escapeLabelValue, matcher } from './promql';

describe('escapeLabelValue', () => {
  it('leaves plain values alone', () => {
    expect(escapeLabelValue('orders-shard1')).toBe('orders-shard1');
  });

  it('escapes backslashes before quotes', () => {
    expect(escapeLabelValue('a\\b')).toBe('a\\\\b');
    expect(escapeLabelValue('say "hi"')).toBe('say \\"hi\\"');
    expect(escapeLabelValue('\\"')).toBe('\\\\\\"');
  });

  it('escapes newlines', () => {
    expect(escapeLabelValue('a\nb')).toBe('a\\nb');
  });
});

describe('matcher', () => {
  it('joins matchers with commas and defaults to equality', () => {
    expect(
      matcher([
        { label: 'service_type', value: 'mongodb' },
        { label: 'job', op: '=~', value: '.*_hr' },
      ])
    ).toBe('service_type="mongodb",job=~".*_hr"');
  });

  it('escapes every value', () => {
    expect(matcher([{ label: 'environment', value: 'p"rod' }])).toBe(
      'environment="p\\"rod"'
    );
  });

  it('keeps an empty value, which PromQL reads as label absent', () => {
    expect(matcher([{ label: 'environment', value: '' }])).toBe(
      'environment=""'
    );
  });

  it('is empty for no matchers', () => {
    expect(matcher([])).toBe('');
  });
});
