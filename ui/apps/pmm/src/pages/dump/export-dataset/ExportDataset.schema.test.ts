import { exportDatasetSchema } from './ExportDataset.schema';
import { getDefaultValues, toStartDumpPayload } from './ExportDataset.utils';

const issuesFor = (values: Parameters<typeof exportDatasetSchema.parse>[0]) => {
  const result = exportDatasetSchema.safeParse(values);

  return result.success ? [] : result.error.issues;
};

describe('exportDatasetSchema', () => {
  it('accepts the default unencrypted form', () => {
    expect(exportDatasetSchema.safeParse(getDefaultValues()).success).toBe(
      true
    );
  });

  it('rejects an invalid date range', () => {
    const values = getDefaultValues();
    values.startTime = values.endTime;

    expect(exportDatasetSchema.safeParse(values).success).toBe(false);
  });

  it('requires a service when only selected services are exported', () => {
    const issues = issuesFor({
      ...getDefaultValues(),
      serviceScope: 'selected',
      serviceNames: [],
    });

    expect(issues).toEqual([
      expect.objectContaining({
        path: ['serviceNames'],
        message: 'Select at least one service.',
      }),
    ]);
  });

  it('accepts all services with an empty list', () => {
    expect(
      issuesFor({
        ...getDefaultValues(),
        serviceScope: 'all',
        serviceNames: [],
      })
    ).toEqual([]);
  });

  it('reports every password rule at once', () => {
    const issues = issuesFor({
      ...getDefaultValues(),
      enableEncryption: true,
      encryptionPassword: 'abc',
    });

    expect(issues).toEqual([
      expect.objectContaining({
        path: ['encryptionPassword'],
        message:
          'At least 8 characters, with a letter, a number and a special character.',
      }),
    ]);
  });
});

describe('toStartDumpPayload', () => {
  it('sends the selected services without the scope', () => {
    const payload = toStartDumpPayload({
      ...getDefaultValues(),
      serviceScope: 'selected',
      serviceNames: ['mysql'],
      exportQan: true,
      enableEncryption: true,
      encryptionPassword: 'Password1!',
    });

    expect(payload).toEqual(
      expect.objectContaining({
        serviceNames: ['mysql'],
        exportQan: true,
        ignoreLoad: false,
        enableEncryption: true,
        encryptionPassword: 'Password1!',
      })
    );
    expect(payload).not.toHaveProperty('serviceScope');
  });

  it('sends an empty list for all services, even after picking some', () => {
    const payload = toStartDumpPayload({
      ...getDefaultValues(),
      serviceScope: 'all',
      serviceNames: ['mysql'],
    });

    expect(payload.serviceNames).toEqual([]);
  });
});
