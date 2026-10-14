import type { RdsExporter } from 'types/agent.types';
import { z } from 'zod';
import { Messages } from './EditService.messages';
import { RdsAuthMode } from './EditService.types';
import { splitCustomLabel, splitCustomLabelEntries } from './EditService.utils';

// Prometheus label names, see https://prometheus.io/docs/concepts/data_model
const LABEL_NAME_RE = /^[a-zA-Z_][a-zA-Z0-9_]*$/;
// Mirrors the server's rule on aws_role_arn in api/inventory/v1/agents.proto
const AWS_ROLE_ARN_RE = /^arn:aws[a-zA-Z0-9-]*:iam::[0-9]{12}:role\/.+$/;

export const areCustomLabelsValid = (text: string) =>
  splitCustomLabelEntries(text).every((entry) => {
    const [name, value] = splitCustomLabel(entry);

    return LABEL_NAME_RE.test(name) && !name.startsWith('__') && !!value;
  });

const editServiceFields = z.object({
  environment: z.string(),
  cluster: z.string(),
  replicationSet: z.string(),
  customLabels: z.string().refine(areCustomLabelsValid, {
    message: Messages.labels.customLabelsInvalid,
  }),
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

export const getEditServiceSchema = (exporter: RdsExporter | null) =>
  editServiceFields.superRefine((values, ctx) => {
    if (!exporter) {
      return;
    }

    for (const [path, message] of getRdsCredentialsErrors(values, exporter)) {
      ctx.addIssue({ code: 'custom', path: [path], message });
    }
  });
