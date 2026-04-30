import type { Configuration } from 'webpack';
import { merge } from 'webpack-merge';
import grafanaConfig from './.config/webpack/webpack.config';
import ForkTsCheckerWebpackPlugin from 'fork-ts-checker-webpack-plugin';
import path from 'path';

const config = async (env: any): Promise<Configuration> => {
  const baseConfig = await grafanaConfig(env);

  // Remove the scaffolded ForkTsCheckerWebpackPlugin — its compiled code calls
  // minimatch_1.default() which is undefined with minimatch v9 (hoisted by the
  // workspace resolution). Re-added below without the issue.include filter so
  // minimatch is never invoked.
  if (baseConfig.plugins) {
    baseConfig.plugins = baseConfig.plugins.filter((p) => !(p instanceof ForkTsCheckerWebpackPlugin));
  }

  return merge(baseConfig, {
    resolve: {
      extensions: ['.js', '.jsx', '.ts', '.tsx'],
      alias: {
        '@pmm/shared': path.resolve(__dirname, '../../packages/shared/src'),
      },
    },
    plugins: env.development
      ? [
          new ForkTsCheckerWebpackPlugin({
            async: true,
            typescript: { configFile: path.join(process.cwd(), 'tsconfig.json') },
          }),
        ]
      : [],
  });
};

export default config;
