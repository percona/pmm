import { z } from 'zod';
import { Messages } from '../../Settings.messages';
import {
  RESOLUTION_MAX,
  RESOLUTION_MIN,
  RESOLUTION_PRESETS,
} from './MetricsResolution.constants';

const { required, minMax } = Messages.metrics.validation;
const rangeMessage = minMax(RESOLUTION_MIN, RESOLUTION_MAX);

const resolutionField = z
  .string()
  .refine((v) => v !== '' && !isNaN(Number(v)), { message: required })
  .refine(
    (v) => {
      const n = Number(v);
      return n >= RESOLUTION_MIN && n <= RESOLUTION_MAX;
    },
    { message: rangeMessage }
  );

const resolutionApiField = z
  .string()
  .regex(/^\d+s$/, required)
  .refine((v) => {
    const n = parseInt(v, 10);
    return n >= RESOLUTION_MIN && n <= RESOLUTION_MAX;
  }, rangeMessage);

export const metricsResolutionsSchema = z.object({
  hr: resolutionApiField,
  mr: resolutionApiField,
  lr: resolutionApiField,
});

export const metricsResolutionSchema = z.object({
  preset: z.enum(RESOLUTION_PRESETS),
  lr: resolutionField,
  mr: resolutionField,
  hr: resolutionField,
});

export type MetricsResolutionFormValues = z.infer<
  typeof metricsResolutionSchema
>;
