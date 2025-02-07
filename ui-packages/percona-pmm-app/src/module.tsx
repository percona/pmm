import { AppPlugin } from '@grafana/data';
import { getAppEvents, ThemeChangedEvent } from '@grafana/runtime';
import { initialize } from 'compat';

getAppEvents().subscribe(ThemeChangedEvent, (e) => {
  console.log('ThemeChangedEvent', e);
});

export const plugin = new AppPlugin<{}>();

console.log('percona-pmm-app-module-loaded');

initialize();
