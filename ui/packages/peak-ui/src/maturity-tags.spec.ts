import { describe, expect, it } from 'vitest';
import { MATURITY_TAGS, badgeStyle, type MaturityStatus } from '../.storybook/maturity-tags';

describe('MATURITY_TAGS', () => {
  it('has unique, lowercase-kebab ids', () => {
    const ids = MATURITY_TAGS.map((t) => t.id);
    expect(new Set(ids).size).toBe(ids.length);
    ids.forEach((id) => expect(id).toMatch(/^[a-z]+(-[a-z]+)*$/));
  });

  it('gives every tag a label, description, and full color set', () => {
    MATURITY_TAGS.forEach((tag) => {
      expect(tag.label.trim()).not.toBe('');
      expect(tag.description.trim()).not.toBe('');
      expect(tag.colors.bg.trim()).not.toBe('');
      expect(tag.colors.fg.trim()).not.toBe('');
      expect(tag.colors.border.trim()).not.toBe('');
    });
  });
});

describe('badgeStyle', () => {
  const { colors } = MATURITY_TAGS[0];

  it('applies the tag colors', () => {
    const style = badgeStyle(colors, 'sidebar');
    expect(style.backgroundColor).toBe(colors.bg);
    expect(style.color).toBe(colors.fg);
    expect(style.display).toBe('inline-block');
  });

  it('sizes badges per context', () => {
    expect(badgeStyle(colors, 'sidebar').fontSize).toBe(8);
    expect(badgeStyle(colors, 'toolbar').fontSize).toBe(12);
    expect(badgeStyle(colors, 'mdx').fontSize).toBe(12);
  });
});

// Governance: every documented story must declare exactly one maturity status on
// its meta, so the sidebar/badges/index.json never silently miss a component.
describe('story maturity coverage', () => {
  const storyFiles = import.meta.glob('./**/*.stories.{ts,tsx}', {
    query: '?raw',
    import: 'default',
    eager: true,
  }) as Record<string, string>;

  const statuses = MATURITY_TAGS.map((t) => t.id);

  const statusesInTagArrays = (source: string): MaturityStatus[] => {
    const tagArrays = [...source.matchAll(/tags:\s*\[([^\]]*)\]/g)].map((m) => m[1]).join(',');
    return statuses.filter((id) => new RegExp(`['"]${id}['"]`).test(tagArrays));
  };

  it('discovers the story files', () => {
    expect(Object.keys(storyFiles).length).toBeGreaterThan(15);
  });

  it.each(Object.entries(storyFiles))('%s declares exactly one status', (_file, source) => {
    expect(statusesInTagArrays(source)).toHaveLength(1);
  });
});
