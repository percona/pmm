import { SettingClass } from '@sep/api';

/** The SEP settings class that owns the diagnostics delivery keys. */
export const SEP_SETTINGS_CLASS: SettingClass = 'SEPSettings';

/**
 * The single structured, writable key. SEP seals its leaves deliberately:
 * `DIAGNOSTICS_DELIVERY_INPUTS__endpoint` / `__secrets` answer 422
 * (`not_overridable`), so the whole object is always written at once.
 */
export const DELIVERY_INPUTS_KEY = 'DIAGNOSTICS_DELIVERY_INPUTS';

/**
 * The read-only delivery plan baked into the SEP image. Its `value.secrets`
 * declares the secret names this deployment must supply — the form renders one
 * field per declared name instead of hardcoding them, so an image that renames
 * one is picked up on the next load rather than 422-ing on save.
 */
export const DELIVERY_PLAN_KEY = 'DIAGNOSTICS_DELIVERY';
