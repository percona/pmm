// force timezone to UTC to allow tests to work regardless of local timezone
// generally used by snapshots, but can affect specific tests
process.env.TZ = 'UTC';

const {
  grafanaESModules,
  nodeModulesToTransform,
} = require('./.config/jest/utils');

module.exports = {
  // Jest configuration provided by Grafana scaffolding
  ...require('./.config/jest.config'),
  // @grafana/data loads marked, which ships only as an ES module
  transformIgnorePatterns: [
    nodeModulesToTransform([...grafanaESModules, 'marked']),
  ],
};
