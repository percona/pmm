import { describe, expect, it } from 'vitest';
import type { RdsExporter } from 'types/agent.types';
import { TEST_MANAGED_SERVICE } from 'utils/testStubs';
import {
  type EditServiceFormValues,
  areCustomLabelsValid,
  getEditServiceSchema,
} from './EditService.schema';
import { RdsAuthMode } from './EditService.types';
import {
  getInitialValues,
  getRdsAuthMode,
  parseCustomLabels,
  toCustomLabelsText,
  toRdsCredentials,
  toServiceChanges,
} from './EditService.utils';

const exporter = (overrides: Partial<RdsExporter> = {}): RdsExporter => ({
  agentId: 'rds-exporter-1',
  awsAccessKey: '',
  awsRoleArn: '',
  isAwsSecretKeySet: false,
  ...overrides,
});

const SERVICE = {
  ...TEST_MANAGED_SERVICE,
  environment: 'prod',
  cluster: 'cluster-1',
  replicationSet: 'rs-1',
  customLabels: { team: 'db', region_name: 'eu' },
};

const values = (
  overrides: Partial<EditServiceFormValues> = {}
): EditServiceFormValues => ({
  ...getInitialValues(SERVICE, null),
  ...overrides,
});

const ROLE_ARN = 'arn:aws:iam::123456789012:role/PmmRdsMonitoring';

describe('custom labels', () => {
  it('round-trips between the API map and the text', () => {
    expect(toCustomLabelsText(SERVICE.customLabels)).toBe(
      'team:db\nregion_name:eu'
    );
    expect(parseCustomLabels('team:db\n region_name:eu  ')).toEqual(
      SERVICE.customLabels
    );
  });

  it('keeps colons in label values', () => {
    const labels = { endpoint: 'db:3306', url: 'http://db' };

    expect(parseCustomLabels(toCustomLabelsText(labels))).toEqual(labels);
  });

  it('keeps spaces in label values', () => {
    const labels = { owner: 'DB Team', team: 'db' };

    expect(parseCustomLabels(toCustomLabelsText(labels))).toEqual(labels);
    expect(parseCustomLabels('owner:DB Team env:prod team:db')).toEqual({
      owner: 'DB Team',
      env: 'prod',
      team: 'db',
    });
  });

  it.each([
    ['', true],
    ['team:db\nregion_name:eu', true],
    ['team:db region_name:eu', true],
    ['team', false],
    ['team:', false],
    ['endpoint:db:3306', true],
    ['owner:DB Team', true],
    ['team:db 1team:db', false],
    ['1team:db', false],
    ['__reserved:db', false],
  ])('validates %j as %s', (text, valid) => {
    expect(areCustomLabelsValid(text)).toBe(valid);
  });
});

describe('getRdsAuthMode', () => {
  it('derives the mode from what the exporter holds', () => {
    expect(getRdsAuthMode(null)).toBe(RdsAuthMode.hostCredentials);
    expect(getRdsAuthMode(exporter())).toBe(RdsAuthMode.hostCredentials);
    expect(getRdsAuthMode(exporter({ awsRoleArn: ROLE_ARN }))).toBe(
      RdsAuthMode.iamRole
    );
    expect(getRdsAuthMode(exporter({ awsAccessKey: 'AKIA' }))).toBe(
      RdsAuthMode.accessKey
    );
  });

  it('treats a stored secret without access key as key authentication', () => {
    expect(getRdsAuthMode(exporter({ isAwsSecretKeySet: true }))).toBe(
      RdsAuthMode.accessKey
    );
  });
});

describe('toRdsCredentials', () => {
  it('does nothing without an exporter', () => {
    expect(
      toRdsCredentials(values({ rdsAuthMode: RdsAuthMode.iamRole }), null)
    ).toBeUndefined();
  });

  describe('migrating to an IAM role', () => {
    const keyExporter = exporter({
      awsAccessKey: 'AKIA',
      isAwsSecretKeySet: true,
    });

    it('clears both keys in the same call, trimming the ARN', () => {
      expect(
        toRdsCredentials(
          values({
            rdsAuthMode: RdsAuthMode.iamRole,
            awsRoleArn: ` ${ROLE_ARN} `,
          }),
          keyExporter
        )
      ).toEqual({ awsRoleArn: ROLE_ARN, awsAccessKey: '', awsSecretKey: '' });
    });

    it('skips the call when the role is unchanged', () => {
      expect(
        toRdsCredentials(
          values({ rdsAuthMode: RdsAuthMode.iamRole, awsRoleArn: ROLE_ARN }),
          exporter({ awsRoleArn: ROLE_ARN })
        )
      ).toBeUndefined();
    });
  });

  describe('migrating to an access key', () => {
    it('clears the role ARN in the same call', () => {
      expect(
        toRdsCredentials(
          values({
            rdsAuthMode: RdsAuthMode.accessKey,
            awsAccessKey: 'AKIA',
            awsSecretKey: 'secret',
          }),
          exporter({ awsRoleArn: ROLE_ARN })
        )
      ).toEqual({
        awsAccessKey: 'AKIA',
        awsRoleArn: '',
        awsSecretKey: 'secret',
      });
    });

    it('omits an untouched secret key so the stored one is kept', () => {
      expect(
        toRdsCredentials(
          values({ rdsAuthMode: RdsAuthMode.accessKey, awsAccessKey: 'AKIA2' }),
          exporter({ awsAccessKey: 'AKIA', isAwsSecretKeySet: true })
        )
      ).toEqual({ awsAccessKey: 'AKIA2', awsRoleArn: '' });
    });

    it('skips the call when nothing about the key changed', () => {
      expect(
        toRdsCredentials(
          values({ rdsAuthMode: RdsAuthMode.accessKey, awsAccessKey: 'AKIA' }),
          exporter({ awsAccessKey: 'AKIA', isAwsSecretKeySet: true })
        )
      ).toBeUndefined();
    });

    it('sends a new secret for the same access key', () => {
      expect(
        toRdsCredentials(
          values({
            rdsAuthMode: RdsAuthMode.accessKey,
            awsAccessKey: 'AKIA',
            awsSecretKey: 'new-secret',
          }),
          exporter({ awsAccessKey: 'AKIA', isAwsSecretKeySet: true })
        )
      ).toEqual({
        awsAccessKey: 'AKIA',
        awsRoleArn: '',
        awsSecretKey: 'new-secret',
      });
    });
  });

  describe('falling back to host credentials', () => {
    const hostValues = values({ rdsAuthMode: RdsAuthMode.hostCredentials });

    it('clears everything PMM holds', () => {
      expect(
        toRdsCredentials(hostValues, exporter({ awsRoleArn: ROLE_ARN }))
      ).toEqual({ awsAccessKey: '', awsSecretKey: '', awsRoleArn: '' });
    });

    it('skips the call when the exporter already uses them', () => {
      expect(toRdsCredentials(hostValues, exporter())).toBeUndefined();
    });

    it('still clears a stored secret that has no access key beside it', () => {
      expect(
        toRdsCredentials(hostValues, exporter({ isAwsSecretKeySet: true }))
      ).toEqual({ awsAccessKey: '', awsSecretKey: '', awsRoleArn: '' });
    });
  });
});

describe('toServiceChanges', () => {
  it('returns nothing when no label changed', () => {
    expect(toServiceChanges(SERVICE, values())).toBeUndefined();
  });

  it('leaves stored labels alone when only another field changed', () => {
    const service = { ...SERVICE, customLabels: { owner: 'DB Team:Lead' } };

    expect(
      toServiceChanges(service, {
        ...getInitialValues(service, null),
        environment: 'staging',
      })
    ).toEqual({ serviceId: SERVICE.serviceId, environment: 'staging' });
  });

  it('sends only the changed labels, clearing emptied ones', () => {
    expect(
      toServiceChanges(
        SERVICE,
        values({ cluster: 'cluster-2', environment: '' })
      )
    ).toEqual({
      serviceId: SERVICE.serviceId,
      cluster: 'cluster-2',
      environment: '',
    });
  });

  it('sends the whole custom labels map when it changed', () => {
    expect(
      toServiceChanges(SERVICE, values({ customLabels: 'team:db' }))
    ).toEqual({ serviceId: SERVICE.serviceId, customLabels: { team: 'db' } });
  });
});

describe('getEditServiceSchema', () => {
  const errors = (
    formValues: EditServiceFormValues,
    rdsExporter: RdsExporter
  ) => {
    const result = getEditServiceSchema(
      rdsExporter,
      toCustomLabelsText(SERVICE.customLabels)
    ).safeParse(formValues);

    return result.success
      ? {}
      : Object.fromEntries(
          result.error.issues.map(({ path, message }) => [path[0], message])
        );
  };

  it('checks the custom labels only once they are edited', () => {
    const schema = getEditServiceSchema(null, 'note:see y:');

    expect(
      schema.safeParse(values({ customLabels: 'note:see y:' })).success
    ).toBe(true);
    expect(
      schema.safeParse(values({ customLabels: 'note:see y: team:db' })).success
    ).toBe(false);
  });

  it('requires a valid role ARN in IAM role mode', () => {
    expect(
      errors(
        values({ rdsAuthMode: RdsAuthMode.iamRole, awsRoleArn: 'arn:bad' }),
        exporter()
      )
    ).toEqual({
      awsRoleArn:
        'Enter an IAM role ARN, for example arn:aws:iam::123456789012:role/RoleName',
    });
    expect(
      errors(
        values({ rdsAuthMode: RdsAuthMode.iamRole, awsRoleArn: ROLE_ARN }),
        exporter()
      )
    ).toEqual({});
  });

  it('keeps the stored secret while the access key is untouched', () => {
    const keyExporter = exporter({
      awsAccessKey: 'AKIA',
      isAwsSecretKeySet: true,
    });

    expect(
      errors(
        values({ rdsAuthMode: RdsAuthMode.accessKey, awsAccessKey: 'AKIA' }),
        keyExporter
      )
    ).toEqual({});
    expect(
      errors(
        values({ rdsAuthMode: RdsAuthMode.accessKey, awsAccessKey: 'AKIA2' }),
        keyExporter
      )
    ).toEqual({
      awsSecretKey: 'Enter the secret key that goes with this access key',
    });
  });

  it('ignores the credentials fields of the other modes', () => {
    expect(
      errors(values({ rdsAuthMode: RdsAuthMode.hostCredentials }), exporter())
    ).toEqual({});
  });
});
