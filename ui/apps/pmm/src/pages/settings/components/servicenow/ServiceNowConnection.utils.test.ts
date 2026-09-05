import {
  ApiError,
  type ConnectivityResult,
  type ConnectivityStatus,
  REDACTED_SECRET,
  SettingClassGroup,
} from '@sep/api';
import { Messages } from '../../Settings.messages';
import {
  DELIVERY_INPUTS_KEY,
  DELIVERY_PLAN_KEY,
} from './ServiceNowConnection.constants';
import {
  buildDeliveryInputsPatch,
  connectionIdentity,
  connectionStatus,
  connectivityOutcome,
  declaredSecretNames,
  deliveryResult,
  normalizeEndpoint,
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
  it('seeds one empty field per declared name, keeping the stored endpoint', () => {
    expect(
      toFormValues(['sn_api_key', 'client_token'], {
        endpoint: 'https://acme.service-now.com/',
        secrets: { sn_api_key: REDACTED_SECRET },
        hasOverride: true,
        isPresent: true,
      })
    ).toEqual({
      endpoint: 'https://acme.service-now.com/',
      secrets: ['', ''],
    });
  });

  it('never seeds a stored secret back into the form', () => {
    expect(
      toFormValues(['sn_api_key'], {
        endpoint: '',
        secrets: { sn_api_key: REDACTED_SECRET },
        hasOverride: true,
        isPresent: true,
      })
    ).toEqual({ endpoint: '', secrets: [''] });
  });
});

describe('buildDeliveryInputsPatch', () => {
  it('submits exactly the declared names, dropping anything else the form holds', () => {
    const patch = buildDeliveryInputsPatch(
      {
        endpoint: 'https://acme.service-now.com/',
        secrets: ['a', 'b', 'c'],
      },
      ['sn_api_key', 'client_token']
    );

    expect(patch).toEqual({
      endpoint: 'https://acme.service-now.com',
      secrets: { sn_api_key: 'a', client_token: 'b' },
    });
  });

  it('adds a declared name the form never rendered as empty', () => {
    expect(
      buildDeliveryInputsPatch({ endpoint: '', secrets: [] }, ['sn_api_key'])
    ).toEqual({ secrets: { sn_api_key: '' } });
  });

  it('omits a blank endpoint so SEP keeps the baked receiver', () => {
    expect(
      buildDeliveryInputsPatch({ endpoint: '   ', secrets: ['a'] }, [
        'sn_api_key',
      ])
    ).toEqual({ secrets: { sn_api_key: 'a' } });
  });

  it('stores the endpoint as SEP will use it, not as it was typed', () => {
    expect(
      buildDeliveryInputsPatch(
        { endpoint: '  https://acme.service-now.com// ', secrets: [] },
        []
      )
    ).toEqual({ endpoint: 'https://acme.service-now.com', secrets: {} });
  });

  it('keys the payload by name even for a name that is not a valid form path', () => {
    expect(
      buildDeliveryInputsPatch({ endpoint: '', secrets: ['a', 'b'] }, [
        'sn.api.key',
        'client[token]',
      ])
    ).toEqual({ secrets: { 'sn.api.key': 'a', 'client[token]': 'b' } });
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
    expect(secretHelperText('sn_api_key')).toBe(
      Messages.serviceNow.secretCopy.sn_api_key.helper
    );
  });

  it('names the raw key for a name the UI has no copy for', () => {
    expect(secretHelperText('instance_url')).toContain('instance_url');
  });
});

describe('normalizeEndpoint', () => {
  it.each([
    ['https://acme.service-now.com/', 'https://acme.service-now.com'],
    ['https://acme.service-now.com//', 'https://acme.service-now.com'],
    ['  https://acme.service-now.com  ', 'https://acme.service-now.com'],
    [
      'https://acme.service-now.com/api/now/',
      'https://acme.service-now.com/api/now',
    ],
  ])('reduces %s to what SEP keeps: %s', (typed, stored) => {
    expect(normalizeEndpoint(typed)).toBe(stored);
  });

  it('keeps the query exactly as typed, rewriting nothing SEP reads itself', () => {
    expect(
      normalizeEndpoint('https://acme.service-now.com/?a=1&a=2&b=x%20y')
    ).toBe('https://acme.service-now.com?a=1&a=2&b=x%20y');
  });

  it('leaves an endpoint carrying credentials alone rather than dropping them', () => {
    // `URL.origin` omits userinfo where Python's `netloc` keeps it, so
    // normalizing this would store an endpoint that authenticates differently
    // from the one SEP receives.
    expect(normalizeEndpoint('https://user:pass@acme.service-now.com/')).toBe(
      'https://user:pass@acme.service-now.com/'
    );
  });

  it('drops a fragment, which never reaches the receiver', () => {
    expect(normalizeEndpoint('https://acme.service-now.com/#section')).toBe(
      'https://acme.service-now.com'
    );
  });

  it('leaves a blank endpoint blank so the baked receiver is kept', () => {
    expect(normalizeEndpoint('   ')).toBe('');
  });

  it.each(['not-a-url', 'ftp://acme.service-now.com/'])(
    'passes %s through rather than inventing a shape for it',
    (value) => {
      expect(normalizeEndpoint(value)).toBe(value);
    }
  );
});

describe('deliveryResult', () => {
  const result = (service: string): ConnectivityResult => ({
    service,
    reachable: true,
    status: 'reachable',
    detail: 'Reachable.',
    version: null,
  });

  it('picks the delivery entry out of the response', () => {
    expect(deliveryResult([result('pmm'), result('delivery')])?.service).toBe(
      'delivery'
    );
  });

  it('reports the only answer there is when delivery is not named', () => {
    expect(deliveryResult([result('pmm')])?.service).toBe('pmm');
  });

  it('has nothing to report for an empty or absent response', () => {
    expect(deliveryResult([])).toBeUndefined();
    expect(deliveryResult(undefined)).toBeUndefined();
  });
});

describe('connectivityOutcome', () => {
  const probed = (
    status: ConnectivityStatus,
    reachable = false
  ): ConnectivityResult => ({
    service: 'delivery',
    reachable,
    status,
    detail: 'whatever SEP said',
    version: null,
  });

  it.each([
    ['reachable', 'success'],
    ['auth_failed', 'error'],
    ['error', 'error'],
    ['unreachable', 'error'],
    ['ssl_error', 'error'],
    ['timeout', 'warning'],
    ['not_configured', 'info'],
    ['inputs_drifted', 'warning'],
    ['probe_undeclared', 'info'],
  ] as [ConnectivityStatus, string][])(
    'gives %s its own label at severity %s',
    (status, severity) => {
      const outcome = connectivityOutcome(
        probed(status, status === 'reachable')
      );

      expect(outcome.severity).toBe(severity);
      expect(outcome.message).toBe(Messages.serviceNow.test.statuses[status]);
    }
  );

  it('never echoes what the receiver said', () => {
    expect(connectivityOutcome(probed('error')).message).not.toContain(
      'whatever SEP said'
    );
  });

  it('falls back on the reachable flag for a status this UI has no copy for', () => {
    const status = 'quantum_tunnelled' as ConnectivityStatus;

    expect(connectivityOutcome(probed(status, true))).toEqual({
      message: Messages.serviceNow.test.unknown.reachable,
      severity: 'success',
    });
    expect(connectivityOutcome(probed(status))).toEqual({
      message: Messages.serviceNow.test.unknown.unreachable,
      severity: 'error',
    });
  });
});

describe('connectionIdentity', () => {
  const stored = (
    endpoint: string,
    secrets: Record<string, string> = { sn_api_key: REDACTED_SECRET }
  ) => ({ endpoint, secrets, hasOverride: true, isPresent: true });

  it('is stable across a refetch that changed nothing', () => {
    expect(connectionIdentity(stored('https://acme.service-now.com'))).toBe(
      connectionIdentity(stored('https://acme.service-now.com'))
    );
  });

  it('ignores the order the secret names came back in', () => {
    expect(
      connectionIdentity(
        stored('', {
          sn_api_key: REDACTED_SECRET,
          client_token: REDACTED_SECRET,
        })
      )
    ).toBe(
      connectionIdentity(
        stored('', {
          client_token: REDACTED_SECRET,
          sn_api_key: REDACTED_SECRET,
        })
      )
    );
  });

  it.each([
    ['a changed endpoint', stored('https://other.service-now.com')],
    [
      'a renamed secret',
      stored('https://acme.service-now.com', { renamed: REDACTED_SECRET }),
    ],
    [
      'an added secret',
      stored('https://acme.service-now.com', {
        sn_api_key: REDACTED_SECRET,
        client_token: REDACTED_SECRET,
      }),
    ],
  ])('changes for %s', (_case, changed) => {
    expect(connectionIdentity(changed)).not.toBe(
      connectionIdentity(stored('https://acme.service-now.com'))
    );
  });
});
