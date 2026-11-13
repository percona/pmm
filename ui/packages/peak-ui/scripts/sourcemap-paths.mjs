/**
 * pnpm's store folder names carry peer versions, so without this a React bump
 * would change the sourcemap of an otherwise identical package.
 * @param {string} path
 * @returns {string}
 */
export const stripPnpmStore = (path) =>
  path.replace(/node_modules\/\.pnpm\/[^/]+\/node_modules\//, 'node_modules/');
