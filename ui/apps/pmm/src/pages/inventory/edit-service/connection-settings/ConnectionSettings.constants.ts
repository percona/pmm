import { METRICS_SCHEMES } from '../EditService.constants';

export const SCHEME_OPTIONS = METRICS_SCHEMES.map((scheme) => ({
  value: scheme,
  label: scheme,
}));
