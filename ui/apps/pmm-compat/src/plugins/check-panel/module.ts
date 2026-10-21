import { PanelPlugin } from '@grafana/data';
import { withQueryClient } from 'plugins/shared/query';
import { CheckPanel } from './CheckPanel';

export const plugin = new PanelPlugin(withQueryClient(CheckPanel));
