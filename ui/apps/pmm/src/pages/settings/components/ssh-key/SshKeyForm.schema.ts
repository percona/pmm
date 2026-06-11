import { z } from 'zod';
import { Messages } from '../../Settings.messages';

const { invalidFormat } = Messages.ssh.validation;

export const sshKeySchema = z.object({
  sshKey: z
    .string()
    .refine(
      (v) =>
        v === '' ||
        /^(ssh-rsa|ssh-ed25519|ssh-dss|ecdsa-sha2-\S+)\s+\S+/.test(v),
      { message: invalidFormat }
    ),
});

export type SshKeyFormValues = z.infer<typeof sshKeySchema>;
