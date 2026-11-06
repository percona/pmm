import { exportDatasetSchema } from './ExportDataset.schema';
import { getDefaultValues, toStartDumpPayload } from './ExportDataset.utils';

describe('exportDatasetSchema', () => {
  it('accepts the default unencrypted form', () => {
    expect(exportDatasetSchema.safeParse(getDefaultValues()).success).toBe(
      true
    );
  });

  it('rejects an invalid date range', () => {
    const values = getDefaultValues();
    values.startTime = values.endTime;

    const result = exportDatasetSchema.safeParse(values);

    expect(result.success).toBe(false);
  });

  it('enforces encryption password complexity', () => {
    const values = {
      ...getDefaultValues(),
      enableEncryption: true,
      encryptionPassword: 'password',
    };

    const result = exportDatasetSchema.safeParse(values);

    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error.issues[0].path).toEqual(['encryptionPassword']);
    }
  });

  it('maps valid values to an API payload', () => {
    const values = {
      ...getDefaultValues(),
      serviceNames: ['mysql'],
      exportQan: true,
      enableEncryption: true,
      encryptionPassword: 'Password1!',
    };

    expect(toStartDumpPayload(values)).toMatchObject({
      serviceNames: ['mysql'],
      exportQan: true,
      ignoreLoad: true,
      enableEncryption: true,
      encryptionPassword: 'Password1!',
    });
  });
});
