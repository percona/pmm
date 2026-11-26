import { AppPlugin } from '@grafana/data';
import { config } from '@grafana/runtime';
import { initialize } from './compat';

export const plugin = new AppPlugin<{}>();

// check if plugin is enabled
if (config.apps['pmm-compat-app']?.preload) {
  initialize();
}
