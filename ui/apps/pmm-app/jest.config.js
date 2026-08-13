// force timezone to UTC to allow tests to work regardless of local timezone
// generally used by snapshots, but can affect specific tests
const path = require('path');
const fs = require('fs');
const baseConfig = require('./.config/jest.config');

process.env.TZ = 'GMT';

// d3's package.json "exports" map doesn't expose ./dist/d3.min.js as a subpath, and its
// install location varies with PNPM workspace hoisting, so resolve the package root by
// walking up from its entry point instead of assuming a fixed node_modules layout.
function resolvePackageRoot(pkgName) {
  let dir = path.dirname(require.resolve(pkgName));
  for (;;) {
    const pkgJsonPath = path.join(dir, 'package.json');
    if (
      fs.existsSync(pkgJsonPath) &&
      JSON.parse(fs.readFileSync(pkgJsonPath, 'utf8')).name === pkgName
    ) {
      return dir;
    }
    const parent = path.dirname(dir);
    if (parent === dir) {
      throw new Error(`Could not resolve package root for ${pkgName}`);
    }
    dir = parent;
  }
}

module.exports = {
  // Jest configuration provided by Grafana scaffolding
  ...baseConfig,
  verbose: true,
  collectCoverage: true,
  collectCoverageFrom: [
    '**/*.{ts,tsx}',
    '!**/node_modules/**',
    '!**/*styles.{ts,tsx}',
    '!**/*constants.{ts,tsx}',
    '!**/*module.{ts,tsx}',
    '!**/*types.ts',
  ],
  moduleNameMapper: {
    ...baseConfig.moduleNameMapper,
    d3: path.join(resolvePackageRoot('d3'), 'dist', 'd3.min.js'),
  },
};
