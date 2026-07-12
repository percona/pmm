import { z } from 'zod';
import { FilterType, Severity } from 'types/alert-templates.types';
import { Messages } from './CreateAlertFromTemplate.messages';
import { CREATE_FOLDER_VALUE } from './CreateAlertFromTemplate.constants';

const { required, minDuration } = Messages.validation;

const positiveSeconds = z
  .string()
  .refine((value) => Number(value) >= 1, { message: minDuration });

// Static-field validation only. Per-template dynamic params are validated with
// field-level react-hook-form rules (see ParamsSection).
export const createRuleSchema = z
  .object({
    template: z.string().min(1, { message: required }),
    name: z.string().trim().min(1, { message: required }),
    severity: z.nativeEnum(Severity),
    duration: positiveSeconds,
    folderUid: z.string().min(1, { message: required }),
    newFolderTitle: z.string(),
    group: z.string().trim().min(1, { message: required }),
    interval: positiveSeconds,
    filters: z.array(
      z.object({
        type: z.nativeEnum(FilterType),
        label: z.string(),
        regexp: z.string(),
      })
    ),
    params: z.record(
      z.string(),
      z.union([z.number(), z.boolean(), z.string()])
    ),
  })
  .superRefine((values, ctx) => {
    if (
      values.folderUid === CREATE_FOLDER_VALUE &&
      values.newFolderTitle.trim() === ''
    ) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['newFolderTitle'],
        message: required,
      });
    }
  });
