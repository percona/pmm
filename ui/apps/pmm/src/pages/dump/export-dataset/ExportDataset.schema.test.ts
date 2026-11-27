import { createExportDatasetSchema } from './ExportDataset.schema';
import {
  fromWallClock,
  getDefaultValues,
  toStartDumpPayload,
  toWallClock,
} from './ExportDataset.utils';

const schema = createExportDatasetSchema('UTC');

const issuesFor = (values: Parameters<typeof schema.parse>[0]) => {
  const result = schema.safeParse(values);

  return result.success ? [] : result.error.issues;
};

describe('exportDatasetSchema', () => {
  it('accepts the default unencrypted form', () => {
    expect(schema.safeParse(getDefaultValues('UTC')).success).toBe(true);
  });

  it('accepts defaults in a zone ahead of the browser', () => {
    const zone = 'Pacific/Kiritimati';

    expect(
      createExportDatasetSchema(zone).safeParse(getDefaultValues(zone)).success
    ).toBe(true);
  });

  it('rejects an invalid date range', () => {
    const values = getDefaultValues('UTC');
    values.startTime = values.endTime;

    expect(schema.safeParse(values).success).toBe(false);
  });

  it('requires a service when only selected services are exported', () => {
    const issues = issuesFor({
      ...getDefaultValues('UTC'),
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
        ...getDefaultValues('UTC'),
        serviceScope: 'all',
        serviceNames: [],
      })
    ).toEqual([]);
  });

  it('reports every password rule at once', () => {
    const issues = issuesFor({
      ...getDefaultValues('UTC'),
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

  it.each([
    ['letters outside A-Z', 'ÄÖÜäöü1!'],
    ['digits outside 0-9', 'abcdef١!'],
  ])('accepts a password with %s, as PMM Server does', (_, password) => {
    expect(
      issuesFor({
        ...getDefaultValues('UTC'),
        enableEncryption: true,
        encryptionPassword: password,
      })
    ).toEqual([]);
  });

  it("doesn't count a space as a special character, as PMM Server doesn't", () => {
    expect(
      issuesFor({
        ...getDefaultValues('UTC'),
        enableEncryption: true,
        encryptionPassword: 'abcdefg1 ',
      })
    ).toEqual([expect.objectContaining({ path: ['encryptionPassword'] })]);
  });
});

describe('wall-clock time in the PMM timezone', () => {
  it('shows an instant as wall-clock time in the zone', () => {
    const wallClock = toWallClock(
      new Date('2026-10-26T09:00:00Z'),
      'Asia/Tokyo'
    );

    expect([wallClock.getDate(), wallClock.getHours()]).toEqual([26, 18]);
  });

  it('reads wall-clock time in the zone back as an instant', () => {
    expect(
      fromWallClock(new Date(2026, 9, 26, 18, 0), 'Asia/Tokyo').toISOString()
    ).toBe('2026-10-26T09:00:00.000Z');
  });
});

describe('toStartDumpPayload', () => {
  it('sends the selected services without the scope', () => {
    const payload = toStartDumpPayload(
      {
        ...getDefaultValues('UTC'),
        serviceScope: 'selected',
        serviceNames: ['mysql'],
        exportQan: true,
        enableEncryption: true,
        encryptionPassword: 'Password1!',
      },
      'UTC'
    );

    expect(payload).toEqual(
      expect.objectContaining({
        serviceNames: ['mysql'],
        exportQan: true,
        ignoreLoad: true,
        enableEncryption: true,
        encryptionPassword: 'Password1!',
      })
    );
    expect(payload).not.toHaveProperty('serviceScope');
  });

  it('sends an empty list for all services, even after picking some', () => {
    const payload = toStartDumpPayload(
      {
        ...getDefaultValues('UTC'),
        serviceScope: 'all',
        serviceNames: ['mysql'],
      },
      'UTC'
    );

    expect(payload.serviceNames).toEqual([]);
  });

  it('sends the picked window as instants in the PMM timezone', () => {
    const payload = toStartDumpPayload(
      {
        ...getDefaultValues('Asia/Tokyo'),
        startTime: new Date(2026, 9, 26, 18, 0),
        endTime: new Date(2026, 9, 27, 6, 0),
      },
      'Asia/Tokyo'
    );

    expect([payload.startTime, payload.endTime]).toEqual([
      '2026-10-26T09:00:00.000Z',
      '2026-10-26T21:00:00.000Z',
    ]);
  });
});
