import { describe, expect, it } from 'vitest';
import { normalizeOperatorYaml } from './logParserPresets';

describe('normalizeOperatorYaml', () => {
  it('splits parse_from onto its own line after a single-quoted regex', () => {
    const collapsed = [
      '- type: regex_parser',
      "  regex: '^(?P<message>.*)$' parse_from: body",
      '  parse_to: attributes',
    ].join('\n');

    const normalized = normalizeOperatorYaml(collapsed);

    expect(normalized).toContain("'\n  parse_from: body");
  });

  it('splits parse_from onto its own line after a double-quoted regex', () => {
    const collapsed = [
      '- type: regex_parser',
      '  regex: "^(?P<message>.*)$" parse_from: body',
      '  parse_to: attributes',
    ].join('\n');

    const normalized = normalizeOperatorYaml(collapsed);

    expect(normalized).toContain('"\n  parse_from: body');
  });

  it('converts literal backslash-n sequences when there are no real newlines', () => {
    const collapsed = String.raw`- type: regex_parser\n  regex: 'foo'\n  parse_from: body`;

    const normalized = normalizeOperatorYaml(collapsed);

    expect(normalized).toContain('\n  parse_from:');
  });

  it('quotes unquoted regex values that contain colons', () => {
    const collapsed = [
      '- type: regex_parser',
      '  regex: ^(?P<timestamp>\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}\\.\\d+Z) (?P<message>.*)$',
      '  parse_from: body',
    ].join('\n');

    const normalized = normalizeOperatorYaml(collapsed);

    expect(normalized).toContain("regex: '^(?P<timestamp>");
    expect(normalized).toContain('\n  parse_from: body');
  });

  it('dedents indented preset YAML from the database', () => {
    const indented = [
      '    - type: regex_parser',
      "      regex: '^(?P<timestamp>\\\\d{4})'",
      '      parse_from: body',
    ].join('\n');

    const normalized = normalizeOperatorYaml(indented);

    expect(normalized).toBe(
      ["- type: regex_parser", "  regex: '^(?P<timestamp>\\\\d{4})'", '  parse_from: body'].join('\n')
    );
  });
});
