// The server rejects mixing them, so the form offers one choice
export enum RdsAuthMode {
  accessKey = 'access_key',
  iamRole = 'iam_role',
  hostCredentials = 'host_credentials',
}
