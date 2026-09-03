import { ApiError, REDACTED_SECRET, SettingClassGroup } from '@sep/api';
import { Messages } from '../../Settings.messages';
import {
  DELIVERY_INPUTS_KEY,
  DELIVERY_PLAN_KEY,
} from './ServiceNowConnection.constants';
import {
  buildDeliveryInputsPatch,
  connectionStatus,
  declaredSecretNames,
  secretHelperText,
  secretLabel,
  sepErrorMessage,
  storedDeliveryInputs,
  toFormValues,
} from './ServiceNowConnection.utils';

const setting = (key: string, value: unknown, hasOverride = false) =>
  ({
    key,
    value,
    has_override: hasOverride,
    default_value: null,
    description: null,
    is_advanced: false,
    is_applicable: true,
    is_complex: true,
    is_secret: false,
    reload: 'none',
    setting_class: 'SEPSettings',
    type: 'object',
  }) as unknown as SettingClassGroup['settings'][number];

const groups = (
  settings: SettingClassGroup['settings']
): SettingClassGroup[] => [
  {
    setting_class: 'TasksSettings',
    is_app_owned: true,
    settings: [
      setting('DIAGNOSTICS_DELIVERY_INPUTS', { secrets: { nope: '' } }),
    ],
  },
  { setting_class: 'SEPSettings', is_app_owned: false, settings },
];

const plan = (names: string[]) =>
  setting(DELIVERY_PLAN_KEY, {
    endpoint: 'https://baked.service-now.com/',
    secrets: Object.fromEntries(names.map((name) => [name, REDACTED_SECRET])),
  });

const stored = (secrets: Record<string, string>, endpoint = '') =>
  setting(DELIVERY_INPUTS_KEY, { endpoint, secrets }, true);

describe('declaredSecretNames', () => {
  it('reads the names off the baked plan', () => {
    expect(
      declaredSecretNames(
        groups([plan(['sn_api_key', 'client_token']), stored({ stale: 'x' })])
      )
    ).toEqual(['sn_api_key', 'client_token']);
  });

  it('falls back to the stored inputs when the plan is not listed', () => {
    expect(
      declaredSecretNames(groups([stored({ sn_api_key: REDACTED_SECRET })]))
    ).toEqual(['sn_api_key']);
  });

  it('ignores settings of another class', () => {
    expect(declaredSecretNames(groups([]))).toEqual([]);
  });

  it('returns nothing for an absent response', () => {
    expect(declaredSecretNames(undefined)).toEqual([]);
  });
});

describe('storedDeliveryInputs', () => {
  it('reports the endpoint, the masked secrets and the override flag', () => {
    expect(
      storedDeliveryInputs(
        groups([
          stored(
            { sn_api_key: REDACTED_SECRET },
            'https://acme.service-now.com/'
          ),
        ])
      )
    ).toEqual({
      endpoint: 'https://acme.service-now.com/',
      secrets: { sn_api_key: REDACTED_SECRET },
      hasOverride: true,
      isPresent: true,
    });
  });

  it('treats a missing key as unconfigured rather than failing', () => {
    expect(storedDeliveryInputs(groups([]))).toEqual({
      endpoint: '',
      secrets: {},
      hasOverride: false,
      isPresent: false,
    });
  });
});

describe('toFormValues', () => {
  it('seeds one field per declared name from the stored values', () => {
    expect(
      toFormValues(['sn_api_key', 'client_token'], {
        endpoint: 'https://acme.service-now.com/',
        secrets: { sn_api_key: REDACTED_SECRET },
        hasOverride: true,
        isPresent: true,
      })
    ).toEqual({
      endpoint: 'https://acme.service-now.com/',
      secrets: [REDACTED_SECRET, ''],
    });
  });

  it('leaves every field empty when nothing is stored', () => {
    expect(
      toFormValues(['sn_api_key'], {
        endpoint: '',
        secrets: { sn_api_key: REDACTED_SECRET },
        hasOverride: false,
        isPresent: true,
      })
    ).toEqual({ endpoint: '', secrets: [''] });
  });
});

describe('buildDeliveryInputsPatch', () => {
  const unconfigured = {
    endpoint: '',
    secrets: {},
    hasOverride: false,
    isPresent: true,
  };
  const configured = {
    endpoint: 'https://acme.service-now.com/',
    secrets: { sn_api_key: REDACTED_SECRET, client_token: REDACTED_SECRET },
    hasOverride: true,
    isPresent: true,
  };

  it('submits exactly the declared names, dropping anything else the form holds', () => {
    const patch = buildDeliveryInputsPatch(
      {
        endpoint: 'https://acme.service-now.com/',
        secrets: ['a', 'b', 'c'],
      },
      ['sn_api_key', 'client_token'],
      unconfigured
    );

    expect(patch).toEqual({
      endpoint: 'https://acme.service-now.com/',
      secrets: { sn_api_key: 'a', client_token: 'b' },
    });
  });

  it('adds a declared name the form never rendered as empty', () => {
    expect(
      buildDeliveryInputsPatch(
        { endpoint: '', secrets: [] },
        ['sn_api_key'],
        unconfigured
      )
    ).toEqual({ secrets: { sn_api_key: '' } });
  });

  it('omits a blank endpoint so SEP keeps the baked receiver', () => {
    expect(
      buildDeliveryInputsPatch(
        { endpoint: '   ', secrets: ['a'] },
        ['sn_api_key'],
        unconfigured
      )
    ).toEqual({ secrets: { sn_api_key: 'a' } });
  });

  it('trims the endpoint it does send', () => {
    expect(
      buildDeliveryInputsPatch(
        { endpoint: '  https://acme.service-now.com/ ', secrets: [] },
        [],
        unconfigured
      )
    ).toEqual({ endpoint: 'https://acme.service-now.com/', secrets: {} });
  });

  it('resubmits an untouched mask so SEP restores the stored secret', () => {
    expect(
      buildDeliveryInputsPatch(
        {
          endpoint: '',
          secrets: [REDACTED_SECRET, 'new'],
        },
        ['sn_api_key', 'client_token'],
        configured
      )
    ).toEqual({
      secrets: { sn_api_key: REDACTED_SECRET, client_token: 'new' },
    });
  });

  it('never sends a mask there is nothing stored to restore', () => {
    expect(
      buildDeliveryInputsPatch(
        { endpoint: '', secrets: [REDACTED_SECRET] },
        ['sn_api_key'],
        unconfigured
      )
    ).toEqual({ secrets: { sn_api_key: '' } });
  });

  it('keys the payload by name even for a name that is not a valid form path', () => {
    expect(
      buildDeliveryInputsPatch(
        { endpoint: '', secrets: ['a', 'b'] },
        ['sn.api.key', 'client[token]'],
        unconfigured
      )
    ).toEqual({ secrets: { 'sn.api.key': 'a', 'client[token]': 'b' } });
  });

  it('sends empty strings, which is a valid unconfigured save', () => {
    expect(
      buildDeliveryInputsPatch(
        { endpoint: '', secrets: ['', ''] },
        ['sn_api_key', 'client_token'],
        configured
      )
    ).toEqual({ secrets: { sn_api_key: '', client_token: '' } });
  });
});

describe('connectionStatus', () => {
  it('is configured when every declared secret has a stored value', () => {
    expect(
      connectionStatus(['sn_api_key'], {
        endpoint: '',
        secrets: { sn_api_key: REDACTED_SECRET },
        hasOverride: true,
        isPresent: true,
      })
    ).toBe('configured');
  });

  it('is not configured when nothing was ever saved', () => {
    expect(
      connectionStatus(['sn_api_key'], {
        endpoint: '',
        secrets: {},
        hasOverride: false,
        isPresent: true,
      })
    ).toBe('not-configured');
  });

  it('reads a saved-but-empty secret as not configured, not as a failure', () => {
    expect(
      connectionStatus(['sn_api_key', 'client_token'], {
        endpoint: '',
        secrets: { sn_api_key: REDACTED_SECRET, client_token: '' },
        hasOverride: true,
        isPresent: true,
      })
    ).toBe('not-configured');
  });

  it('judges a plan that declares no secrets on the override alone', () => {
    expect(
      connectionStatus([], {
        endpoint: 'https://acme.service-now.com/',
        secrets: {},
        hasOverride: true,
        isPresent: true,
      })
    ).toBe('configured');
  });

  it('still reads as not configured with no secrets and no override', () => {
    expect(
      connectionStatus([], {
        endpoint: '',
        secrets: {},
        hasOverride: false,
        isPresent: true,
      })
    ).toBe('not-configured');
  });

  it('reports drift when the plan declares a name the stored inputs lack', () => {
    expect(
      connectionStatus(['sn_api_key', 'renamed_token'], {
        endpoint: '',
        secrets: { sn_api_key: REDACTED_SECRET, client_token: REDACTED_SECRET },
        hasOverride: true,
        isPresent: true,
      })
    ).toBe('drifted');
  });
});

describe('sepErrorMessage', () => {
  const httpError = (status: number, data?: unknown) =>
    new ApiError({ kind: 'http', status, message: `HTTP ${status}`, data });

  it('surfaces the per-field 422 message SEP returns', () => {
    const message = sepErrorMessage(
      httpError(422, {
        detail: [
          {
            loc: ['body', DELIVERY_INPUTS_KEY, 'secrets'],
            msg: 'undeclared secret names: extra_key',
            type: 'value_error',
          },
        ],
      })
    );

    expect(message).toBe('undeclared secret names: extra_key');
  });

  it('explains a 403 instead of leaving it unaccounted for', () => {
    expect(sepErrorMessage(httpError(403))).toBe(
      Messages.serviceNow.errors.forbidden
    );
  });

  it('explains a 401', () => {
    expect(sepErrorMessage(httpError(401))).toBe(
      Messages.serviceNow.errors.unauthenticated
    );
  });

  it('reports an unreachable SEP', () => {
    expect(
      sepErrorMessage(new ApiError({ kind: 'network', message: 'boom' }))
    ).toBe(Messages.serviceNow.errors.unreachable);
  });

  it('never leaks a raw HTTP message', () => {
    expect(sepErrorMessage(httpError(500))).toBe(
      Messages.serviceNow.errors.generic
    );
  });

  it('uses the caller fallback when one is given', () => {
    expect(sepErrorMessage(httpError(500), 'nope')).toBe('nope');
  });

  it('is empty without an error', () => {
    expect(sepErrorMessage(null)).toBe('');
  });
});

describe('secretLabel', () => {
  it.each([
    ['sn_api_key', 'ServiceNow API key'],
    ['client_token', 'Client token'],
  ])('uses the written copy for %s', (name, expected) => {
    expect(secretLabel(name)).toBe(expected);
  });

  it.each([
    ['instance_url', 'Instance URL'],
    ['token', 'Token'],
  ])('humanizes an undeclared name %s as %s', (name, expected) => {
    expect(secretLabel(name)).toBe(expected);
  });
});

describe('secretHelperText', () => {
  it('uses the written copy for a declared name', () => {
    expect(secretHelperText('sn_api_key', false)).toBe(
      Messages.serviceNow.secretCopy.sn_api_key.helper
    );
  });

  it('names the raw key for a name the UI has no copy for', () => {
    expect(secretHelperText('instance_url', false)).toContain('instance_url');
  });

  it('adds the keep-it note when a value is stored', () => {
    expect(secretHelperText('client_token', true)).toBe(
      `${Messages.serviceNow.secretCopy.client_token.helper} ${Messages.serviceNow.secretStoredSuffix}`
    );
  });
});
