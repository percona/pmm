export enum DistributionMethod {
  unspecified = 'DISTRIBUTION_METHOD_UNSPECIFIED',
  docker = 'DISTRIBUTION_METHOD_DOCKER',
  ovf = 'DISTRIBUTION_METHOD_OVF',
  ami = 'DISTRIBUTION_METHOD_AMI',
  azure = 'DISTRIBUTION_METHOD_AZURE',
  do = 'DISTRIBUTION_METHOD_DO',
}

export interface VersionInfo {
  version: string;
  fullVersion: string;
  timestamp: string;
}

export interface VersionResponse {
  version: string;
  server: VersionInfo;
  managed: VersionInfo;
  distributionMethod: DistributionMethod;
}
