export type PromMatchOp = '=' | '!=' | '=~' | '!~';

export interface PromMatcher {
  label: string;
  value: string;
  op?: PromMatchOp;
}

/** Escapes a value for a double-quoted PromQL string literal. */
export const escapeLabelValue = (value: string): string =>
  value.replace(/\\/g, '\\\\').replace(/"/g, '\\"').replace(/\n/g, '\\n');

/** The inside of a series selector's braces, such as `service_type="mongodb",job=~".*_hr"`. */
export const matcher = (matchers: PromMatcher[]): string =>
  matchers
    .map(
      ({ label, value, op = '=' }) =>
        `${label}${op}"${escapeLabelValue(value)}"`
    )
    .join(',');
