import { AppPlugin } from '@grafana/data';
import { CrossFrameMessenger } from '@pmm/shared';
import { initialize } from './compat';

export const plugin = new AppPlugin<{}>();

console.log('pmm-compat-loaded');

const messenger = new CrossFrameMessenger();

console.log('CrossFrameMessenger - from Grafana Plugin', messenger);

initialize();
