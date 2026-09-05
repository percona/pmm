import {
  ApiError,
  SettingClassGroup,
  SettingResponse,
  settingErrorMessage,
} from '@sep/api';
import { Messages } from '../../Settings.messages';
import {
  DELIVERY_INPUTS_KEY,
  DELIVERY_PLAN_KEY,
  SEP_SETTINGS_CLASS,
} from './ServiceNowConnection.constants';
import {
  ConnectionStatus,
  DeliveryInputs,
  ServiceNowFormValues,
  StoredDeliveryInputs,
} from './ServiceNowConnection.types';

/** Locate one setting inside the `SEPSettings` group of a LIST response. */
export const findSepSetting = (
  groups: SettingClassGroup[] | undefined,
  key: string
): SettingResponse | undefined =>
  groups
    ?.find((group) => group.setting_class === SEP_SETTINGS_CLASS)
    ?.settings.find((setting) => setting.key === key);

const asInputs = (value: unknown): DeliveryInputs => {
  if (!value || typeof value !== 'object') {
    return {};
  }
  const { endpoint, secrets } = value as DeliveryInputs;
  return {
    endpoint: typeof endpoint === 'string' ? endpoint : null,
    secrets:
      secrets && typeof secrets === 'object' && !Array.isArray(secrets)
        ? Object.fromEntries(
            Object.entries(secrets).map(([name, secret]) => [
              name,
              typeof secret === 'string' ? secret : '',
            ])
          )
        : {},
  };
};

/**
 * The secret names this deployment must supply, read from the baked plan.
 *
 * The plan is the declaration SEP validates a write against, so it always wins.
 * The stored inputs are a fallback for a SEP build that does not list the plan
 * at all: their names are only stale if that build also renamed one, and the
 * cost of guessing wrong there is a 422 the form shows verbatim — better than a
 * form with no fields, which no operator could recover from.
 */
export const declaredSecretNames = (
  groups: SettingClassGroup[] | undefined
): string[] => {
  const planNames = Object.keys(
    asInputs(findSepSetting(groups, DELIVERY_PLAN_KEY)?.value).secrets ?? {}
  );
  if (planNames.length > 0) {
    return planNames;
  }
  return Object.keys(
    asInputs(findSepSetting(groups, DELIVERY_INPUTS_KEY)?.value).secrets ?? {}
  );
};

/**
 * The stored per-deployment inputs. Secrets come back masked once something is
 * stored, so a value here says only that one exists — it is never displayed and
 * never inspected for content.
 */
export const storedDeliveryInputs = (
  groups: SettingClassGroup[] | undefined
): StoredDeliveryInputs => {
  const setting = findSepSetting(groups, DELIVERY_INPUTS_KEY);
  const { endpoint, secrets } = asInputs(setting?.value);
  return {
    endpoint: endpoint ?? '',
    secrets: secrets ?? {},
    hasOverride: setting?.has_override ?? false,
    isPresent: setting !== undefined,
  };
};

/**
 * Seed the form: the stored endpoint, and one empty field per declared secret
 * name in declaration order — the form addresses secrets by position, not by
 * name.
 *
 * Secrets are never seeded from what SEP holds. Every route to the form is a
 * route that replaces them: nothing is stored yet, the stored values no longer
 * satisfy the plan, or the operator asked to renew them. SEP masks a stored
 * secret anyway, so seeding could only ever put a mask in front of someone
 * about to overwrite it.
 */
export const toFormValues = (
  declaredNames: string[],
  stored: StoredDeliveryInputs
): ServiceNowFormValues => ({
  endpoint: stored.endpoint,
  secrets: declaredNames.map(() => ''),
});

/**
 * Build the PATCH value: one whole object carrying exactly the declared secret
 * names.
 *
 * `endpoint` is dropped when blank so SEP keeps the receiver its image bakes
 * in — that is also how a previously entered endpoint is reverted. The secrets
 * are whatever the operator typed: the form requires every declared one, so
 * there is no mask to restore and no empty value to send.
 */
export const buildDeliveryInputsPatch = (
  values: ServiceNowFormValues,
  declaredNames: string[]
): DeliveryInputs => {
  const endpoint = values.endpoint.trim();
  const secrets = Object.fromEntries(
    declaredNames.map((name, index) => [name, values.secrets[index] ?? ''])
  );
  return endpoint ? { endpoint, secrets } : { secrets };
};

/**
 * What the stored inputs say about delivery, without asking SEP a second time.
 *
 * An empty secret is a valid save that leaves delivery unavailable, so it reads
 * as "not configured" rather than as a failure. A declared name with no stored
 * counterpart means the image renamed one after the values were supplied — the
 * value SEP still holds no longer satisfies the plan.
 *
 * A plan that declares no secrets is judged on the override alone: there is no
 * credential left for the deployment to supply, so a stored override is as
 * configured as this form can make it, and the endpoint the operator saved
 * would otherwise never stop reading as missing.
 */
export const connectionStatus = (
  declaredNames: string[],
  stored: StoredDeliveryInputs
): ConnectionStatus => {
  if (!stored.hasOverride) {
    return 'not-configured';
  }
  if (declaredNames.length === 0) {
    return 'configured';
  }
  if (declaredNames.some((name) => stored.secrets[name] === undefined)) {
    return 'drifted';
  }
  return declaredNames.every((name) => stored.secrets[name] !== '')
    ? 'configured'
    : 'not-configured';
};

/**
 * Turn a failed SEP call into something the operator can act on.
 *
 * SEP's 422 message is the most specific thing available — it names the
 * offending secret keys — so it wins over the generic mapping. Everything else
 * distinguishes "you may not do this" from "SEP did not answer", because the
 * two need different responses. A raw HTTP message is never shown: anything
 * unrecognised falls back to `fallback`.
 */
export const sepErrorMessage = (
  error: ApiError | null | undefined,
  fallback: string = Messages.serviceNow.errors.generic
): string => {
  if (!error) {
    return '';
  }
  const validation = settingErrorMessage(error, DELIVERY_INPUTS_KEY);
  if (validation) {
    return validation;
  }
  const { errors } = Messages.serviceNow;
  if (error.status === 403) {
    return errors.forbidden;
  }
  if (error.status === 401) {
    return errors.unauthenticated;
  }
  if (error.kind === 'network' || error.kind === 'timeout') {
    return errors.unreachable;
  }
  return fallback;
};

const ACRONYMS = new Set([
  'api',
  'id',
  'sn',
  'url',
  'uri',
  'jwt',
  'ssl',
  'tls',
]);

const humanizeSecretName = (name: string): string =>
  name
    .split(/[_\-\s]+/)
    .filter(Boolean)
    .map((word, index) => {
      if (ACRONYMS.has(word.toLowerCase())) {
        return word.toUpperCase();
      }
      return index === 0
        ? word.charAt(0).toUpperCase() + word.slice(1).toLowerCase()
        : word.toLowerCase();
    })
    .join(' ');

/**
 * Render a SEP secret name as a field label.
 *
 * A name the delivery plan is known to declare gets the copy Support wrote for
 * it; anything else is a SEP build the UI has no copy for, so its raw name is
 * humanized — `instance_url` reads "Instance URL" — and stays recognisable
 * against SEP's own documentation and error messages.
 */
export const secretLabel = (name: string): string =>
  Messages.serviceNow.secretCopy[name]?.label ?? humanizeSecretName(name);

/**
 * Helper text under a credential field: what the credential is for, or — for a
 * name the UI has no copy for — the raw key SEP will receive it as.
 */
export const secretHelperText = (name: string): string => {
  const { secretCopy, secretHelper } = Messages.serviceNow;
  return secretCopy[name]?.helper ?? secretHelper(name);
};
