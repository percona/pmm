import { PerconaMoIcon } from '@percona/peak-ui';
import { NAV_MONGO } from 'contexts/navigation/navigation.constants';
import { MONGODB_HUB_PATH } from 'lib/constants';
import type { TechnologyHubConfig } from 'hubs/template/hub.types';
import { MONGODB_ATTENTION_ORDER } from './mongodb.attention';
import { deriveMongoHub } from './mongodb.derive';
import { MONGODB_HUB_MESSAGES, Messages } from './mongodb.messages';
import { useMongoLiveSnapshot } from './mongodb.sources';
import { MONGODB_FIXTURES } from './fixtures';
import type { MongoSnapshot } from './mongodb.types';

export const MONGODB_HUB: TechnologyHubConfig<MongoSnapshot> = {
  id: 'mongodb',
  navId: NAV_MONGO.id,
  path: MONGODB_HUB_PATH,
  name: MONGODB_HUB_MESSAGES.name,
  icon: PerconaMoIcon,
  messages: MONGODB_HUB_MESSAGES,
  treeColumns: [
    { id: 'lag', ...Messages.columns.lag },
    { id: 'oplogWindow', ...Messages.columns.oplogWindow },
  ],
  attentionOrder: MONGODB_ATTENTION_ORDER,
  useLiveSnapshot: useMongoLiveSnapshot,
  fixtures: MONGODB_FIXTURES,
  derive: deriveMongoHub,
};
