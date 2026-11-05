import {
  COMMON_OPTIONS_DEFAULTS,
  LABELS_DEFAULTS,
} from '../ServiceForm.constants';
import { toRadioOptions } from '../ServiceForm.utils';
import { Messages } from './ExternalForm.messages';
import {
  type ExternalFormValues,
  MetricsParameters,
  Scheme,
} from './ExternalForm.types';

export const DEFAULT_PORTS: Record<Scheme, number> = {
  [Scheme.https]: 443,
  [Scheme.http]: 80,
};

export const METRICS_PARAMETERS_OPTIONS = toRadioOptions(
  Object.values(MetricsParameters),
  Messages.options
);

export const SCHEME_OPTIONS = toRadioOptions(
  Object.values(Scheme),
  Messages.options
);

export const EXTERNAL_DEFAULTS: ExternalFormValues = {
  serviceName: '',
  group: '',
  metricsParameters: MetricsParameters.manually,
  url: '',
  // What the old page effectively used: it showed HTTPS but never sent the scheme.
  scheme: Scheme.http,
  address: '',
  metricsPath: '',
  port: String(DEFAULT_PORTS[Scheme.http]),
  username: '',
  password: '',
  ...LABELS_DEFAULTS,
  ...COMMON_OPTIONS_DEFAULTS,
};
