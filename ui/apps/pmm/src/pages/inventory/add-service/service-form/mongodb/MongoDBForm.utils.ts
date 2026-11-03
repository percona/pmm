import type { AddServicePayload } from 'types/services.types';
import type { ServiceNode } from '../ServiceForm.types';
import {
  toCollectors,
  toDatabaseService,
  toOptionalNumber,
  whenTls,
} from '../ServiceForm.utils';
import { X509_AUTHENTICATION } from './MongoDBForm.constants';
import type { MongoDBFormValues } from './MongoDBForm.types';

export const toMongoDBPayload = (
  values: MongoDBFormValues,
  nodes: ServiceNode[]
): AddServicePayload => ({
  mongodb: {
    ...toDatabaseService(values, nodes),
    ...whenTls(values.tls, {
      tlsCertificateKeyFilePassword: values.tlsCertificateKeyFilePassword,
      tlsCertificateKey: values.tlsCertificateKey,
      tlsCa: values.tlsCa,
      authenticationMechanism: X509_AUTHENTICATION,
    }),
    qanMongodbProfiler: values.qanMongodbProfiler,
    maxQueryLength: toOptionalNumber(values.maxQueryLength),
    disableCollectors: toCollectors(values.disableCollectors),
  },
});
