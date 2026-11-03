import { z } from 'zod';
import { Messages as FormMessages } from '../ServiceForm.messages';
import {
  commonOptionsSchema,
  labelsSchema,
  requireWhen,
} from '../ServiceForm.schema';
import { isValidPort } from '../ServiceForm.utils';
import { Messages } from './ExternalForm.messages';
import { MetricsParameters, Scheme } from './ExternalForm.types';
import { isValidExternalUrl } from './ExternalForm.utils';

const isParsed = (values: { metricsParameters: MetricsParameters }) =>
  values.metricsParameters === MetricsParameters.parsed;

// Only the fields of the chosen way of entering the endpoint are shown and checked.
export const externalSchema = z
  .object({
    serviceName: z.string(),
    group: z.string(),
    metricsParameters: z.enum(MetricsParameters),
    url: z.string(),
    scheme: z.enum(Scheme),
    address: z.string(),
    metricsPath: z.string(),
    port: z.string(),
    username: z.string(),
    password: z.string(),
    ...labelsSchema.shape,
    ...commonOptionsSchema.shape,
  })
  .superRefine(requireWhen((values) => isParsed(values), 'url'))
  .superRefine(requireWhen((values) => !isParsed(values), 'address'))
  .superRefine((values, ctx) => {
    if (isParsed(values)) {
      if (values.url.trim() && !isValidExternalUrl(values.url)) {
        ctx.addIssue({
          code: 'custom',
          message: Messages.invalidUrl,
          path: ['url'],
        });
      }
    } else if (!isValidPort(values.port)) {
      ctx.addIssue({
        code: 'custom',
        message: FormMessages.validation.port,
        path: ['port'],
      });
    }
  });
