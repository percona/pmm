import { PanelPlugin } from '@grafana/data';
import { AuroraTreePanel } from 'pmm-aurora-tree/panel/AuroraTreePanel';
import { DEFAULT_OPTIONS } from 'pmm-aurora-tree/panel/AuroraTreePanel.constants';
import { Messages } from 'pmm-aurora-tree/panel/AuroraTreePanel.messages';
import { AuroraTreeOptions } from 'pmm-aurora-tree/panel/AuroraTreePanel.types';

// One row per Aurora MySQL instance: the labels region, aws_account_id, aws_cluster_id and service_name
// are read by name, and the four metric values from the fields named in these options.
export const plugin = new PanelPlugin<AuroraTreeOptions>(AuroraTreePanel).setPanelOptions((builder) => builder
  .addFieldNamePicker({
    path: 'upField',
    name: Messages.options.upField,
    description: Messages.options.upFieldDescription,
    defaultValue: DEFAULT_OPTIONS.upField,
  })
  .addFieldNamePicker({
    path: 'readOnlyField',
    name: Messages.options.readOnlyField,
    description: Messages.options.readOnlyFieldDescription,
    defaultValue: DEFAULT_OPTIONS.readOnlyField,
  })
  .addFieldNamePicker({
    path: 'qpsField',
    name: Messages.options.qpsField,
    description: Messages.options.qpsFieldDescription,
    defaultValue: DEFAULT_OPTIONS.qpsField,
  })
  .addFieldNamePicker({
    path: 'storageField',
    name: Messages.options.storageField,
    description: Messages.options.storageFieldDescription,
    defaultValue: DEFAULT_OPTIONS.storageField,
  }));
