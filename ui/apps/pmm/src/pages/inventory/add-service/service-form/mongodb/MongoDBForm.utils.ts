import type { AddServicePayload } from 'types/services.types';
import {
  toCollectors,
  toDatabaseService,
  toOptionalNumber,
  whenTls,
} from '../ServiceForm.utils';
import { X509_AUTHENTICATION } from './MongoDBForm.constants';
import type { MongoDBFormValues } from './MongoDBForm.types';

export const toMongoDBPayload = (
  values: MongoDBFormValues
): AddServicePayload => ({
  mongodb: {
    ...toDatabaseService(values),
    ...whenTls(values.tls, {
      tlsCertificateKeyFilePassword: values.tlsCertificateKeyFilePassword,
      tlsCertificateKey: values.tlsCertificateKey,
      tlsCa: values.tlsCa,
    }),
    // A client certificate without a username is what authenticates.
    ...(values.tls &&
      !!values.tlsCertificateKey.trim() &&
      !values.username.trim() && {
        authenticationMechanism: X509_AUTHENTICATION,
      }),
    qanMongodbProfiler: values.qanMongodbProfiler,
    maxQueryLength: toOptionalNumber(values.maxQueryLength),
    disableCollectors: toCollectors(values.disableCollectors),
  },
});
