import { getSeverityBand } from './AlertSeverity.utils';

describe('getSeverityBand', () => {
  it.each([
    ['emergency', 'critical'],
    ['alert', 'critical'],
    ['critical', 'critical'],
    ['error', 'error'],
    ['warning', 'warning'],
    ['notice', 'info'],
    ['info', 'info'],
    ['informational', 'info'],
    ['debug', 'info'],
  ])('maps %s to the %s band', (severity, band) => {
    expect(getSeverityBand(severity)).toBe(band);
  });

  it('ignores case and surrounding whitespace', () => {
    expect(getSeverityBand(' Emergency ')).toBe('critical');
    expect(getSeverityBand('WARNING')).toBe('warning');
  });

  it('returns undefined for missing or unrecognized severities', () => {
    expect(getSeverityBand(undefined)).toBeUndefined();
    expect(getSeverityBand('')).toBeUndefined();
    expect(getSeverityBand('unknown')).toBeUndefined();
    expect(getSeverityBand('constructor')).toBeUndefined();
  });
});
