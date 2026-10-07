import {
  type ConnectivityCheckRequest,
  SettingClass,
} from '@pmm-extensions/api';

/** The side-car settings class that owns the diagnostics delivery keys. */
export const EXTENSIONS_SETTINGS_CLASS: SettingClass = 'ExtensionsSettings';

/**
 * The single structured, writable key. The side-car seals its leaves deliberately:
 * `DIAGNOSTICS_DELIVERY_INPUTS__endpoint` / `__secrets` answer 422
 * (`not_overridable`), so the whole object is always written at once.
 */
export const DELIVERY_INPUTS_KEY = 'DIAGNOSTICS_DELIVERY_INPUTS';

/**
 * The read-only delivery plan baked into the side-car image. Its `value.secrets`
 * declares the secret names this deployment must supply — the form renders one
 * field per declared name instead of hardcoding them, so an image that renames
 * one is picked up on the next load rather than 422-ing on save.
 */
export const DELIVERY_PLAN_KEY = 'DIAGNOSTICS_DELIVERY';

/**
 * The single connectivity target this tab probes, as the side-car's generated
 * `ServiceEnum` — the endpoint probes whatever it is asked for, and the tab has
 * no reason to report on PMM, Inventory, Tasks or Nomad.
 */
export const DELIVERY_TARGETS: ConnectivityCheckRequest['targets'] = [
  'delivery',
];
