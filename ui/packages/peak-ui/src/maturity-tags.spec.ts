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
  const storyMetas = import.meta.glob('./**/*.stories.{ts,tsx}', {
    import: 'default',
    eager: true,
  }) as Record<string, { tags?: readonly string[] }>;

  const statuses = MATURITY_TAGS.map((t) => t.id);

  const statusesInTags = (tags: unknown): MaturityStatus[] => {
    if (!Array.isArray(tags)) return [];
    return statuses.filter((id) => tags.includes(id));
  };

  it('discovers the story files', () => {
    expect(Object.keys(storyMetas).length).toBeGreaterThan(0);
  });

  it.each(Object.entries(storyMetas))('%s declares exactly one status', (_file, meta) => {
    expect(statusesInTags(meta.tags)).toHaveLength(1);
  });
});
