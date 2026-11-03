import { z } from 'zod';
import { MAX_INT32 } from './ServiceForm.constants';
import { Messages } from './ServiceForm.messages';
import {
  getConnectionTimeoutError,
  isValidCollectors,
  isValidKeyValues,
  isValidPort,
} from './ServiceForm.utils';

const { validation } = Messages;

export const requiredText = z.string().trim().min(1, validation.required);

export const portText = z.string().refine(isValidPort, validation.port);

export const keyValuesText = z
  .string()
  .refine(isValidKeyValues, validation.keyValue);

export const getIntegerError = (value: string, min: number) => {
  const number = Number(value);

  if (!Number.isInteger(number)) {
    return validation.integer;
  }
  if (number < min) {
    return validation.min(min);
  }
  if (number > MAX_INT32) {
    return validation.max(MAX_INT32);
  }
  return undefined;
};

// An optional int32, at least `min`; numbers stay text in the form like the other fields.
export const optionalIntegerText = (min: number) =>
  z.string().superRefine((value, ctx) => {
    const message = value.trim() && getIntegerError(value, min);

    if (message) {
      ctx.addIssue({ code: 'custom', message });
    }
  });

export const maxQueryLengthSchema = z.object({
  maxQueryLength: optionalIntegerText(-1),
});

export const connectionSchema = z.object({
  serviceName: z.string(),
  address: requiredText,
  port: portText,
  username: z.string(),
  password: z.string(),
  connectionTimeout: z.string().superRefine((value, ctx) => {
    const message = getConnectionTimeoutError(value);
    if (message) {
      ctx.addIssue({ code: 'custom', message });
    }
  }),
});

export const nodeSchema = z.object({
  nodeId: requiredText,
  pmmAgentId: requiredText,
  // Set by the node picker; a service on the PMM Server node is pulled, any other pushes.
  isPmmServerNode: z.boolean(),
});

export const labelsSchema = z.object({
  environment: z.string(),
  cluster: z.string(),
  replicationSet: z.string(),
  region: z.string(),
  az: z.string(),
  customLabels: keyValuesText,
});

export const commonOptionsSchema = z.object({
  skipConnectionCheck: z.boolean(),
  tlsSkipVerify: z.boolean(),
});

export const tlsSchema = z.object({
  tls: z.boolean(),
});

// The base of every database service form (MySQL, PostgreSQL, MongoDB, ProxySQL, Valkey).
export const databaseServiceSchema = z.object({
  ...connectionSchema.shape,
  ...nodeSchema.shape,
  ...labelsSchema.shape,
  ...commonOptionsSchema.shape,
  ...tlsSchema.shape,
});

export const tlsCertificatesSchema = z.object({
  tlsCa: z.string(),
  tlsCert: z.string(),
  tlsKey: z.string(),
});

export const mongodbTlsCertificatesSchema = z.object({
  tlsCertificateKeyFilePassword: z.string(),
  tlsCertificateKey: z.string(),
  tlsCa: z.string(),
});

export const queryAnalyticsOptionsSchema = z.object({
  disableQueryExamples: z.boolean(),
  disableCommentsParsing: z.boolean(),
});

export const disableCollectorsSchema = z.object({
  disableCollectors: z
    .string()
    .refine(isValidCollectors, validation.collectors),
});

// Client certificates replace the username and password when TLS is on.
export const requireCredentialsWithoutTls = (
  values: { tls: boolean; username: string; password: string },
  ctx: z.RefinementCtx
) => {
  if (values.tls) {
    return;
  }

  for (const field of ['username', 'password'] as const) {
    if (!values[field].trim()) {
      ctx.addIssue({
        code: 'custom',
        message: validation.required,
        path: [field],
      });
    }
  }
};

// A field the form only shows, and so only checks, for some choice (e.g. a custom limit).
export const requireWhen =
  <T extends Record<string, unknown>>(
    isShown: (values: T) => boolean,
    field: keyof T & string,
    check?: (value: string) => string | undefined
  ) =>
  (values: T, ctx: z.RefinementCtx) => {
    if (!isShown(values)) {
      return;
    }

    const value = String(values[field] ?? '').trim();
    const message = value ? check?.(value) : validation.required;

    if (message) {
      ctx.addIssue({ code: 'custom', message, path: [field] });
    }
  };
