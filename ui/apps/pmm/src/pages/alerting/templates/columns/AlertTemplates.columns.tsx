import { type MRT_ColumnDef } from '@percona/percona-ui';
import { Template } from 'types/alert-templates.types';
import { formatTimestamp } from 'utils/datetime.utils';
import { Messages } from '../AlertTemplates.messages';
import { SourceCell } from './cell-source';
import { TemplateRowActions } from './cell-actions';

interface ColumnsOptions {
  canManage: boolean;
  onEdit: (template: Template) => void;
  onDelete: (template: Template) => void;
}

export const getAlertTemplatesColumns = ({
  canManage,
  onEdit,
  onDelete,
}: ColumnsOptions): MRT_ColumnDef<Template>[] => [
  {
    accessorKey: 'summary',
    header: Messages.columns.name,
    Cell: ({ row }) => row.original.summary || row.original.name,
  },
  {
    accessorKey: 'source',
    header: Messages.columns.source,
    Cell: ({ row }) => <SourceCell source={row.original.source} />,
  },
  {
    accessorKey: 'createdAt',
    header: Messages.columns.createdAt,
    Cell: ({ row }) =>
      row.original.createdAt ? formatTimestamp(row.original.createdAt) : '—',
  },
  {
    id: 'actions',
    header: Messages.columns.actions,
    enableSorting: false,
    enableColumnFilter: false,
    Cell: ({ row }) => (
      <TemplateRowActions
        template={row.original}
        canManage={canManage}
        onEdit={onEdit}
        onDelete={onDelete}
      />
    ),
  },
];
