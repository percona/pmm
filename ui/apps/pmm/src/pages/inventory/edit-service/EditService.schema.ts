import type { RdsExporter } from 'types/agent.types';
import { areCustomLabelsValid } from 'utils/labels.utils';
import { z } from 'zod';
import { AWS_ROLE_ARN_RE } from '../Inventory.constants';
import { RdsAuthMode } from '../Inventory.types';
import { Messages } from './EditService.messages';

const editServiceFields = z.object({
  environment: z.string(),
  cluster: z.string(),
  replicationSet: z.string(),
  customLabels: z.string(),
  rdsAuthMode: z.enum(RdsAuthMode),
  awsAccessKey: z.string(),
  awsSecretKey: z.string(),
  awsRoleArn: z.string(),
});

export type EditServiceFormValues = z.infer<typeof editServiceFields>;

const getRdsCredentialsErrors = (
  values: EditServiceFormValues,
  exporter: RdsExporter
) => {
  const { required, secretKeyRequired, invalidRoleArn } = Messages.rds;
  const errors: [keyof EditServiceFormValues, string][] = [];

  if (values.rdsAuthMode === RdsAuthMode.accessKey) {
    const accessKey = values.awsAccessKey.trim();

    if (!accessKey) {
      errors.push(['awsAccessKey', required]);
    }

    // A stored secret key only goes with the stored access key
    if (
      !values.awsSecretKey &&
      !(exporter.isAwsSecretKeySet && accessKey === exporter.awsAccessKey)
    ) {
      errors.push(['awsSecretKey', secretKeyRequired]);
    }
  }

  if (values.rdsAuthMode === RdsAuthMode.iamRole) {
    const roleArn = values.awsRoleArn.trim();

    if (!roleArn) {
      errors.push(['awsRoleArn', required]);
    } else if (!AWS_ROLE_ARN_RE.test(roleArn)) {
      errors.push(['awsRoleArn', invalidRoleArn]);
    }
  }

  return errors;
};

// Stored labels are only checked once edited, see toServiceChanges
export const getEditServiceSchema = (
  exporter: RdsExporter | null,
  storedCustomLabels: string
) =>
  editServiceFields.superRefine((values, ctx) => {
    if (
      values.customLabels !== storedCustomLabels &&
      !areCustomLabelsValid(values.customLabels)
    ) {
      ctx.addIssue({
        code: 'custom',
        path: ['customLabels'],
        message: Messages.labels.customLabelsInvalid,
      });
    }

    if (!exporter) {
      return;
    }

    for (const [path, message] of getRdsCredentialsErrors(values, exporter)) {
      ctx.addIssue({ code: 'custom', path: [path], message });
    }
  });
