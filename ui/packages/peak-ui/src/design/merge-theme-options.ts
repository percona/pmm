import type { ThemeOptions } from '@mui/material';
import { deepmerge } from '@mui/utils';

type StyleOverrideFn = (props: unknown) => Record<string, unknown>;
type StyleOverrideValue = Record<string, unknown> | StyleOverrideFn;
type StyleOverrides = Record<string, StyleOverrideValue> | StyleOverrideFn;

type ComponentConfig = Record<string, unknown> & {
  styleOverrides?: StyleOverrides;
  variants?: unknown[];
};

const isComponentConfig = (value: unknown): value is ComponentConfig =>
  typeof value === 'object' && value !== null;

const resolveStyleOverrides = (
  value: StyleOverrides | undefined,
  props?: unknown
): Record<string, StyleOverrideValue> | undefined => {
  if (!value) {
    return undefined;
  }

  if (typeof value === 'function') {
    return value(props) as Record<string, StyleOverrideValue>;
  }

  return value;
};

const mergeStyleOverrideSlot = (
  base: StyleOverrideValue | undefined,
  override: StyleOverrideValue | undefined
): StyleOverrideValue | undefined => {
  if (override === undefined) {
    return base;
  }

  if (base === undefined) {
    return override;
  }

  const baseIsFn = typeof base === 'function';
  const overrideIsFn = typeof override === 'function';

  if (!baseIsFn && !overrideIsFn) {
    return deepmerge(base, override);
  }

  return (props: unknown) =>
    deepmerge(
      baseIsFn ? (base as StyleOverrideFn)(props) : base,
      overrideIsFn ? (override as StyleOverrideFn)(props) : override
    );
};

const mergeStyleOverrides = (
  base: StyleOverrides | undefined,
  override: StyleOverrides | undefined
): StyleOverrides | undefined => {
  if (!override) {
    return base;
  }

  if (!base) {
    return override;
  }

  const baseIsFn = typeof base === 'function';
  const overrideIsFn = typeof override === 'function';

  if (baseIsFn || overrideIsFn) {
    return (props: unknown) => {
      const baseResolved = resolveStyleOverrides(base, props) ?? {};
      const overrideResolved = resolveStyleOverrides(override, props) ?? {};
      const keys = new Set([...Object.keys(baseResolved), ...Object.keys(overrideResolved)]);
      const merged: Record<string, StyleOverrideValue> = {};

      for (const key of keys) {
        const slot = mergeStyleOverrideSlot(baseResolved[key], overrideResolved[key]);

        if (slot !== undefined) {
          merged[key] = slot;
        }
      }

      return merged;
    };
  }

  const keys = new Set([
    ...Object.keys(base as Record<string, StyleOverrideValue>),
    ...Object.keys(override as Record<string, StyleOverrideValue>),
  ]);
  const merged: Record<string, StyleOverrideValue> = {};

  for (const key of keys) {
    const slot = mergeStyleOverrideSlot(
      (base as Record<string, StyleOverrideValue>)[key],
      (override as Record<string, StyleOverrideValue>)[key]
    );

    if (slot !== undefined) {
      merged[key] = slot;
    }
  }

  return merged;
};

export const mergeThemeOptions = (base: ThemeOptions, override: ThemeOptions): ThemeOptions => {
  const merged = deepmerge<ThemeOptions>(base, override);
  const baseComponents = base.components ?? {};
  const overrideComponents = override.components ?? {};
  const componentKeys = new Set([
    ...Object.keys(baseComponents),
    ...Object.keys(overrideComponents),
  ]);

  if (componentKeys.size === 0) {
    return merged;
  }

  const components = { ...(merged.components ?? {}) } as Record<string, ComponentConfig>;

  for (const key of componentKeys) {
    const baseComponent = baseComponents[key as keyof typeof baseComponents];
    const overrideComponent = overrideComponents[key as keyof typeof overrideComponents];
    const baseConfig = isComponentConfig(baseComponent) ? baseComponent : undefined;
    const overrideConfig = isComponentConfig(overrideComponent) ? overrideComponent : undefined;

    const hasStyleOverrides = Boolean(baseConfig?.styleOverrides || overrideConfig?.styleOverrides);
    const hasVariants =
      Array.isArray(baseConfig?.variants) || Array.isArray(overrideConfig?.variants);

    if (!hasStyleOverrides && !hasVariants) {
      continue;
    }

    const mergedConfig: ComponentConfig = deepmerge(baseConfig ?? {}, overrideConfig ?? {});

    if (hasStyleOverrides) {
      mergedConfig.styleOverrides = mergeStyleOverrides(
        baseConfig?.styleOverrides,
        overrideConfig?.styleOverrides
      );
    }

    // deepmerge replaces arrays; concatenate base-first so child variants cascade last (win on conflict)
    if (hasVariants) {
      mergedConfig.variants = [
        ...(Array.isArray(baseConfig?.variants) ? baseConfig.variants : []),
        ...(Array.isArray(overrideConfig?.variants) ? overrideConfig.variants : []),
      ];
    }

    components[key] = mergedConfig;
  }

  return { ...merged, components: components as ThemeOptions['components'] };
};
