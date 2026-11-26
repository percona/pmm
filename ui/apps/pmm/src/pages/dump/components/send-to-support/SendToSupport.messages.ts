export const Messages = {
  title: (count: number) =>
    count === 1
      ? 'Send this dataset to Percona Support'
      : `Send ${count} datasets to Percona Support`,
  sftpInfo:
    'Percona Support gives you these SFTP details in your support ticket.',
  address: 'SFTP server',
  addressHelp: 'host:port',
  user: 'Username',
  password: 'Password',
  directory: 'Directory (optional)',
  encryptedWarning:
    'Some of these datasets are encrypted. Share their password with Percona Support in your ticket, never alongside the archive.',
  required: 'Required field',
  send: 'Send',
  sending: 'Sending...',
  cancel: 'Cancel',
  success: 'Datasets were sent to Support.',
};
