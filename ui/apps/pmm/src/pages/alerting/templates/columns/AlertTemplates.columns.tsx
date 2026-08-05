import { type MRT_ColumnDef } from '@percona/percona-ui';
import { Template } from 'types/alert-templates.types';
import { formatCreatedAt } from 'utils/alert-templates.utils';
import { Messages } from '../AlertTemplates.messages';
import { SourceCell } from './cell-source';
import { TemplateRowActions } from './cell-actions';
import { TEMPLATE_CATEGORY_MAP } from '../AlertTemplates.constants';

interface ColumnsOptions {
  canManage: boolean;
  onView: (template: Template) => void;
  onEdit: (template: Template) => void;
  onDelete: (template: Template) => void;
}

export const getAlertTemplatesColumns = ({
  canManage,
  onView,
  onEdit,
  onDelete,
}: ColumnsOptions): MRT_ColumnDef<Template>[] => [
  {
    accessorKey: 'summary',
    header: Messages.columns.name,
    Cell: ({ row }) => row.original.summary || row.original.name,
  },
  {
    size: 150,
    accessorKey: 'source',
    header: Messages.columns.source,
    Cell: ({ row }) => <SourceCell source={row.original.source} />,
  },
  {
    accessorKey: 'category',
    header: Messages.columns.category,
    Cell: ({ row }) =>
      TEMPLATE_CATEGORY_MAP[row.original.category] || row.original.category,
  },
  {
    accessorKey: 'createdAt',
    header: Messages.columns.createdAt,
    Cell: ({ row }) => formatCreatedAt(row.original.createdAt),
  },
  {
    id: 'actions',
    size: 100,
    header: Messages.columns.actions,
    enableSorting: false,
    enableColumnFilter: false,
    Cell: ({ row }) => (
      <TemplateRowActions
        template={row.original}
        canManage={canManage}
        onView={onView}
        onEdit={onEdit}
        onDelete={onDelete}
      />
    ),
  },
];
