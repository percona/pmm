import { z } from 'zod';
import { AWS_ROLE_ARN_RE } from '../../../../Inventory.constants';
import { RdsAuthMode } from '../../../../Inventory.types';
import { Messages } from './RdsCredentials.messages';

// Only the chosen mode's fields are checked; PMM Server's identity needs none.
export const rdsCredentialsSchema = z
  .object({
    rdsAuthMode: z.enum(RdsAuthMode),
    awsAccessKey: z.string(),
    awsSecretKey: z.string(),
    awsRoleArn: z.string(),
  })
  .superRefine((values, ctx) => {
    const addIssue = (path: string, message: string) =>
      ctx.addIssue({ code: 'custom', message, path: [path] });

    if (values.rdsAuthMode === RdsAuthMode.accessKey) {
      if (!values.awsAccessKey.trim()) {
        addIssue('awsAccessKey', Messages.required);
      }
      if (!values.awsSecretKey.trim()) {
        addIssue('awsSecretKey', Messages.required);
      }
    }

    if (values.rdsAuthMode === RdsAuthMode.iamRole) {
      const roleArn = values.awsRoleArn.trim();

      if (!roleArn) {
        addIssue('awsRoleArn', Messages.required);
      } else if (!AWS_ROLE_ARN_RE.test(roleArn)) {
        addIssue('awsRoleArn', Messages.invalidRoleArn);
      }
    }
  });
