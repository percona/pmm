import type {
  CommonOptionsFormValues,
  DatabaseServiceFormValues,
  DisableCollectorsFormValues,
  LabelsFormValues,
  MaxQueryLengthFormValues,
  MongodbTlsCertificatesFormValues,
  NodeFormValues,
  QueryAnalyticsOptionsFormValues,
  TlsCertificatesFormValues,
  TlsFormValues,
} from './ServiceForm.types';

export const MAX_CONNECTION_TIMEOUT_SECONDS = 3600;

export const MAX_INT32 = 2 ** 31 - 1;

export const LOCAL_ADDRESSES = ['127.0.0.1', 'localhost'];

export const NODE_DEFAULTS: NodeFormValues = { nodeId: '', pmmAgentId: '' };

export const LABELS_DEFAULTS: LabelsFormValues = {
  environment: '',
  cluster: '',
  replicationSet: '',
  region: '',
  az: '',
  customLabels: '',
};

export const COMMON_OPTIONS_DEFAULTS: CommonOptionsFormValues = {
  skipConnectionCheck: false,
  tlsSkipVerify: false,
};

export const TLS_DEFAULTS: TlsFormValues = { tls: false };

export const TLS_CERTIFICATES_DEFAULTS: TlsCertificatesFormValues = {
  tlsCa: '',
  tlsCert: '',
  tlsKey: '',
};

export const MONGODB_TLS_CERTIFICATES_DEFAULTS: MongodbTlsCertificatesFormValues =
  {
    tlsCertificateKeyFilePassword: '',
    tlsCertificateKey: '',
    tlsCa: '',
  };

export const QUERY_ANALYTICS_OPTIONS_DEFAULTS: QueryAnalyticsOptionsFormValues =
  {
    disableQueryExamples: false,
    disableCommentsParsing: false,
  };

export const DISABLE_COLLECTORS_DEFAULTS: DisableCollectorsFormValues = {
  disableCollectors: '',
};

export const MAX_QUERY_LENGTH_DEFAULTS: MaxQueryLengthFormValues = {
  maxQueryLength: '',
};

export const DATABASE_SERVICE_DEFAULTS: Omit<
  DatabaseServiceFormValues,
  'port'
> = {
  serviceName: '',
  address: '',
  username: '',
  password: '',
  connectionTimeout: '',
  ...NODE_DEFAULTS,
  ...LABELS_DEFAULTS,
  ...COMMON_OPTIONS_DEFAULTS,
  ...TLS_DEFAULTS,
};
