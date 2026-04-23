// force timezone to UTC to allow tests to work regardless of local timezone
// generally used by snapshots, but can affect specific tests
const baseConfig = require('./.config/jest.config');
const path = require('path');

process.env.TZ = 'GMT';

module.exports = {
  // Jest configuration provided by Grafana scaffolding
  ...baseConfig,
  verbose: true,
  collectCoverage: true,
  coverageProvider: 'v8',
  collectCoverageFrom: [
    '**/*.{ts,tsx}',
    '!**/node_modules/**',
    '!**/*styles.{ts,tsx}',
    '!**/*constants.{ts,tsx}',
    '!**/*module.{ts,tsx}',
    '!**/*types.ts',
    '!webpack.config.ts',
    '!**/.config/**',
  ],
  moduleNameMapper: {
    ...baseConfig.moduleNameMapper,
    '^d3$': path.resolve(__dirname, '../../ui/node_modules/d3/dist/d3.min.js'),
    '^react$': require.resolve('react'),
    '^react-dom$': require.resolve('react-dom'),
  },
};
