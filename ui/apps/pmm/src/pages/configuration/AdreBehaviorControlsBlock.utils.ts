export type AdreBehaviorVariant = 'fast' | 'investigation' | 'format';

const FAST_SHIPPED: Record<string, boolean> = {
  time_skills: false,
  todowrite_instructions: false,
  todowrite_reminder: false,
};

export function shippedPreset(variant: AdreBehaviorVariant): Record<string, boolean> {
  if (variant === 'investigation') return {};
  return { ...FAST_SHIPPED };
}

/** Merge stored settings with PMM shipped preset for empty keys (editing model). */
export function hydrateAdreBehaviorMap(
  raw: Record<string, boolean> | undefined | null,
  variant: AdreBehaviorVariant
): Record<string, boolean> {
  return { ...shippedPreset(variant), ...(raw ?? {}) };
}

export function effectiveValue(
  map: Record<string, boolean>,
  key: string,
  variant: AdreBehaviorVariant
): boolean {
  if (Object.prototype.hasOwnProperty.call(map, key)) return map[key];
  if (variant === 'investigation') return true;
  return shippedPreset(variant)[key] ?? true;
}

export function labelForKey(key: string): string {
  return key
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}
