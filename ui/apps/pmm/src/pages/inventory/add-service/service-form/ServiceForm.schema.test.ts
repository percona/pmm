import { z } from 'zod';
import {
  connectionSchema,
  optionalIntegerText,
  requireCredentials,
} from './ServiceForm.schema';

const credentialsSchema = z
  .object({
    tls: z.boolean(),
    tlsCert: z.string().optional(),
    username: z.string(),
    password: z.string(),
  })
  .superRefine(requireCredentials);

const errorPaths = (result: { success: boolean; error?: z.ZodError }) =>
  result.error?.issues.map((issue) => issue.path.join('.')) ?? [];

describe('ServiceForm.schema', () => {
  it('requires the username and password without TLS', () => {
    const result = credentialsSchema.safeParse({
      tls: false,
      username: '',
      password: ' ',
    });

    expect(errorPaths(result)).toEqual(['username', 'password']);
  });

  it.each([undefined, ' '])(
    'requires the password with TLS but no client certificate (%j)',
    (tlsCert) => {
      const result = credentialsSchema.safeParse({
        tls: true,
        tlsCert,
        username: '',
        password: '',
      });

      expect(errorPaths(result)).toEqual(['username', 'password']);
    }
  );

  it('requires only the username with a client certificate over TLS', () => {
    const result = credentialsSchema.safeParse({
      tls: true,
      tlsCert: 'CERT',
      username: '',
      password: '',
    });

    expect(errorPaths(result)).toEqual(['username']);
  });

  it('requires the password when TLS is off, whatever the certificate', () => {
    const result = credentialsSchema.safeParse({
      tls: false,
      tlsCert: 'CERT',
      username: 'pmm',
      password: '',
    });

    expect(errorPaths(result)).toEqual(['password']);
  });

  it.each([
    ['', true],
    ['-1', true],
    ['2048', true],
    ['-2', false],
    ['1.5', false],
    ['2147483647', true],
    ['2147483648', false],
    ['abc', false],
  ])('checks %j as a whole number of at least -1', (value, valid) => {
    expect(optionalIntegerText(-1).safeParse(value).success).toBe(valid);
  });

  it('reports the old message below the minimum', () => {
    const result = optionalIntegerText(-1).safeParse('-5');

    expect(result.error?.issues[0].message).toBe(
      'Value should be greater or equal to -1'
    );
  });

  it.each([
    ['1.5', 'Must be an Integer number'],
    ['2147483648', 'Value should be lower or equal to 2147483647'],
  ])('explains why %j is rejected', (value, message) => {
    expect(optionalIntegerText(-1).safeParse(value).error?.issues).toEqual([
      expect.objectContaining({ message }),
    ]);
  });

  it('requires the address and a valid port', () => {
    const result = connectionSchema.safeParse({
      serviceName: '',
      address: ' ',
      port: '70000',
      username: '',
      password: '',
      connectionTimeout: '10m',
    });

    expect(errorPaths(result)).toEqual([
      'address',
      'port',
      'connectionTimeout',
    ]);
  });
});
