/** The shape the side-car stores under `DIAGNOSTICS_DELIVERY_INPUTS`. */
export interface DeliveryInputs {
  endpoint?: string | null;
  secrets?: Record<string, string>;
}

export interface StoredDeliveryInputs {
  endpoint: string;
  /** Secret values as the side-car returns them — masked once an override exists. */
  secrets: Record<string, string>;
  /** Whether the side-car holds a per-deployment override (masks are restorable). */
  hasOverride: boolean;
  /** Whether the key came back at all; `false` means the side-car does not expose it. */
  isPresent: boolean;
}

export type ConnectionStatus = 'configured' | 'not-configured' | 'drifted';

export interface ServiceNowFormValues {
  endpoint: string;
  /**
   * Secret values positionally aligned with the declared names.
   *
   * Deliberately not keyed by name: react-hook-form reads a field name as a
   * path, so a declared name carrying a `.` or `[` would register as a nested
   * field and read back as `undefined` — silently submitting an empty string
   * over a stored secret. The names are runtime data from the side-car, so the form
   * never lets them reach a path.
   */
  secrets: string[];
}
