import { PanelPlugin } from '@grafana/data';
import { AuroraTreePanel } from 'pmm-aurora-tree/panel/AuroraTreePanel';
import { DEFAULT_OPTIONS } from 'pmm-aurora-tree/panel/AuroraTreePanel.constants';
import { Messages } from 'pmm-aurora-tree/panel/AuroraTreePanel.messages';
import { AuroraTreeOptions } from 'pmm-aurora-tree/panel/AuroraTreePanel.types';

const FIELD_OPTIONS = Object.keys(DEFAULT_OPTIONS) as Array<keyof AuroraTreeOptions>;

// One row per RDS service: frames with service_name carry service labels and metrics, frames with only
// node_id the aws_rds_instance_info labels and storage of the node. The metric values are read from the
// fields named in these options.
export const plugin = new PanelPlugin<AuroraTreeOptions>(AuroraTreePanel).setPanelOptions((builder) => {
  FIELD_OPTIONS.forEach((path) => builder.addFieldNamePicker({
    path,
    ...Messages.options[path],
    defaultValue: DEFAULT_OPTIONS[path],
  }));
});
