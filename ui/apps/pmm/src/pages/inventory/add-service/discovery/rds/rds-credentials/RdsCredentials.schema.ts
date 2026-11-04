import { z } from 'zod';
import { ROLE_ARN_RE } from './RdsCredentials.constants';
import { Messages } from './RdsCredentials.messages';

// None of the fields is required: without them PMM Server uses its own AWS identity.
export const rdsCredentialsSchema = z
  .object({
    awsAccessKey: z.string(),
    awsSecretKey: z.string(),
    awsRoleArn: z.string(),
  })
  .superRefine((values, ctx) => {
    const roleArn = values.awsRoleArn.trim();
    const accessKey = values.awsAccessKey.trim();
    const secretKey = values.awsSecretKey.trim();
    const hasKeys = !!(accessKey || secretKey);

    if (roleArn && !ROLE_ARN_RE.test(roleArn)) {
      ctx.addIssue({
        code: 'custom',
        message: Messages.invalidRoleArn,
        path: ['awsRoleArn'],
      });
    } else if (roleArn && hasKeys) {
      for (const path of [
        accessKey ? 'awsAccessKey' : 'awsSecretKey',
        'awsRoleArn',
      ]) {
        ctx.addIssue({
          code: 'custom',
          message: Messages.keysOrRole,
          path: [path],
        });
      }
    } else if (hasKeys && !(accessKey && secretKey)) {
      ctx.addIssue({
        code: 'custom',
        message: Messages.bothKeys,
        path: [accessKey ? 'awsSecretKey' : 'awsAccessKey'],
      });
    }
  });
