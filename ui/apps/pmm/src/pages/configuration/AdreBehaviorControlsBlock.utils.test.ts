import {
  effectiveValue,
  hydrateAdreBehaviorMap,
  shippedPreset,
} from './AdreBehaviorControlsBlock.utils';

describe('AdreBehaviorControlsBlock utils', () => {
  it('opts investigation into TodoWrite planning', () => {
    expect(shippedPreset('investigation')).toEqual({
      todowrite_instructions: true,
      todowrite_reminder: true,
      general_instructions: false,
    });
  });

  it('lets stored investigation keys override the preset', () => {
    const map = hydrateAdreBehaviorMap(
      { todowrite_reminder: false },
      'investigation'
    );

    expect(effectiveValue(map, 'todowrite_instructions', 'investigation')).toBe(
      true
    );
    expect(effectiveValue(map, 'todowrite_reminder', 'investigation')).toBe(
      false
    );
    expect(effectiveValue(map, 'intro', 'investigation')).toBe(true);
  });

  it('keeps TodoWrite off for fast mode', () => {
    const map = hydrateAdreBehaviorMap(undefined, 'fast');

    expect(effectiveValue(map, 'todowrite_instructions', 'fast')).toBe(false);
    expect(effectiveValue(map, 'time_skills', 'fast')).toBe(false);
  });

  it('drops Holmes general instructions in every preset', () => {
    for (const variant of ['fast', 'investigation', 'format'] as const) {
      const map = hydrateAdreBehaviorMap(undefined, variant);
      expect(effectiveValue(map, 'general_instructions', variant)).toBe(false);
    }
  });
});
