// force timezone to UTC to allow tests to work regardless of local timezone
// generally used by snapshots, but can affect specific tests
process.env.TZ = 'UTC';

module.exports = {
  // Jest configuration provided by Grafana scaffolding
  ...require('./.config/jest.config'),
  // @grafana/ui pulls in ESM-only packages through pnpm's nested
  // node_modules, which the scaffold's allow-list doesn't match
  transformIgnorePatterns: [],
};
