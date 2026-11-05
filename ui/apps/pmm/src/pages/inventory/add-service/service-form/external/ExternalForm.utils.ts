import type { AddServicePayload } from 'types/services.types';
import { toOptionalString, toScrapedService } from '../ServiceForm.utils';
import { DEFAULT_PORTS } from './ExternalForm.constants';
import { Messages } from './ExternalForm.messages';
import {
  type ExporterEndpoint,
  type ExternalFormValues,
  MetricsParameters,
  Scheme,
} from './ExternalForm.types';

const toScheme = (protocol: string) => protocol.replace(/:$/, '');

const isScheme = (value: string): value is Scheme =>
  Object.values<string>(Scheme).includes(value);

const toUrl = (text: string) => {
  try {
    return new URL(text.trim());
  } catch {
    return undefined;
  }
};

export const getExternalUrlError = (text: string) => {
  const url = toUrl(text);

  if (!url || !isScheme(toScheme(url.protocol)) || !url.hostname) {
    return Messages.invalidUrl;
  }
  // Query parameters would be dropped from the metrics path.
  if (url.search) {
    return Messages.urlWithQuery;
  }
  return undefined;
};

// A port left at a scheme's default follows the scheme; one the user typed stays.
export const toSchemePort = (port: string, scheme: Scheme) =>
  !port.trim() || Object.values(DEFAULT_PORTS).map(String).includes(port.trim())
    ? String(DEFAULT_PORTS[scheme])
    : port;

// The scheme the exporter is scraped with, once the endpoint names one.
export const getEndpointScheme = ({
  metricsParameters,
  url,
  scheme,
}: Pick<ExternalFormValues, 'metricsParameters' | 'url' | 'scheme'>) => {
  if (metricsParameters !== MetricsParameters.parsed) {
    return scheme;
  }

  const protocol = toUrl(url)?.protocol;
  const parsed = protocol && toScheme(protocol);
  return parsed && isScheme(parsed) ? parsed : undefined;
};

// An endpoint without a path keeps the server's default metrics path.
export const parseExternalUrl = (text: string): ExporterEndpoint => {
  const url = new URL(text.trim());
  const scheme = toScheme(url.protocol) as Scheme;

  return {
    scheme,
    address: url.hostname.replace(/^\[(.*)\]$/, '$1'),
    listenPort: Number(url.port) || DEFAULT_PORTS[scheme],
    metricsPath: url.pathname === '/' ? undefined : url.pathname,
    username: decodeURIComponent(url.username),
    password: decodeURIComponent(url.password),
  };
};

const toEndpoint = (values: ExternalFormValues): ExporterEndpoint =>
  values.metricsParameters === MetricsParameters.parsed
    ? parseExternalUrl(values.url)
    : {
        scheme: values.scheme,
        address: values.address.trim(),
        listenPort: Number(values.port),
        metricsPath: toOptionalString(values.metricsPath),
        username: values.username.trim(),
        password: values.password.trim(),
      };

export const toExternalPayload = (
  values: ExternalFormValues
): AddServicePayload => {
  const endpoint = toEndpoint(values);

  return {
    external: {
      ...toScrapedService(values, endpoint.address),
      ...endpoint,
      // Only offered over HTTPS, so a choice made before switching to HTTP is dropped.
      tlsSkipVerify: endpoint.scheme === Scheme.https && values.tlsSkipVerify,
      group: toOptionalString(values.group),
    },
  };
};
