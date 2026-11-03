import { PanelPlugin } from '@grafana/data';
import { withQueryClient } from 'plugins/shared/query';
import { UpdatePanel } from './UpdatePanel';

export const plugin = new PanelPlugin(withQueryClient(UpdatePanel));
