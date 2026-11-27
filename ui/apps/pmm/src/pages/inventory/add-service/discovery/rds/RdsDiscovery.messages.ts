export const Messages = {
  noCredentials: 'No credentials provided and IAM role is not defined',
  serverDiscovery: {
    loading: "Looking for instances with PMM Server's AWS identity…",
    failed:
      "PMM Server couldn't list instances with its own AWS identity. Choose an access key or an IAM role and select Discover.",
  },
  docs: {
    credentials:
      'Where do I get the security credentials for my Amazon RDS DB instances',
    iamRole:
      'How to configure IAM role to automatically discover Amazon RDS DB instances',
  },
  columns: {
    region: 'Region',
    az: 'Availability Zone',
    engine: 'Engine',
    instanceId: 'Instance ID',
    address: 'Address',
  },
};
