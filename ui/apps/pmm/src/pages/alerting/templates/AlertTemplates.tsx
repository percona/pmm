import { FC, useMemo, useState } from 'react';
import Button from '@mui/material/Button';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import { boxClasses } from '@mui/material/Box';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import FileDownloadOutlinedIcon from '@mui/icons-material/FileDownloadOutlined';
import { Table } from '@percona/percona-ui';
import { useUser } from 'contexts/user';
import { useAlertTemplates } from 'hooks/api/useAlertTemplates';
import { Template } from 'types/alert-templates.types';
import { downloadTextFile } from 'utils/file.utils';
import { Page } from 'components/page';
import { Messages } from './AlertTemplates.messages';
import {
  ALERT_TEMPLATES_TABLE_NAME,
  ALL_TEMPLATE_CATEGORIES,
  TEMPLATE_CATEGORY_OPTIONS,
} from './AlertTemplates.constants';
import { getAlertTemplatesColumns } from './columns/AlertTemplates.columns';
import { CreateTemplateModal } from './modal-create-template';
import { EditTemplateModal } from './modal-edit-template';
import { DeleteTemplateModal } from './modal-delete-template';
import { ViewTemplateModal } from './modal-view-template';
import { getTemplateExportFilename } from 'utils/alert-templates.utils';

type ModalType = 'create' | 'view' | 'edit' | 'delete' | null;

export const AlertTemplates: FC = () => {
  const { user } = useUser();
  const canManage = !!user?.isPMMAdmin;
  const { data, isLoading } = useAlertTemplates({ reload: true });
  const [modal, setModal] = useState<ModalType>(null);
  const [selected, setSelected] = useState<Template | null>(null);
  const [category, setCategory] = useState<string>(ALL_TEMPLATE_CATEGORIES);
  const [rowSelection, setRowSelection] = useState<Record<string, boolean>>({});

  const templates = useMemo(() => data?.templates ?? [], [data]);
  const rows = useMemo(
    () =>
      category === ALL_TEMPLATE_CATEGORIES
        ? templates
        : templates.filter((t) => t.category === category),
    [templates, category]
  );
  const selectedTemplates = useMemo(
    () => rows.filter((t) => rowSelection[t.name]),
    [rows, rowSelection]
  );

  const closeModal = () => {
    setModal(null);
    setSelected(null);
  };

  const handleExportSelected = () => {
    selectedTemplates.forEach((template) =>
      downloadTextFile(getTemplateExportFilename(template), template.yaml)
    );
  };

  const columns = useMemo(
    () =>
      getAlertTemplatesColumns({
        canManage,
        onView: (template) => {
          setSelected(template);
          setModal('view');
        },
        onEdit: (template) => {
          setSelected(template);
          setModal('edit');
        },
        onDelete: (template) => {
          setSelected(template);
          setModal('delete');
        },
        onDuplicate: (template) => {
          setSelected(template);
          setModal('create');
        },
      }),
    [canManage]
  );

  return (
    <Page title={Messages.title} fullWidth>
      <Table
        tableName={ALERT_TEMPLATES_TABLE_NAME}
        columns={columns}
        data={rows}
        state={{ isLoading, rowSelection }}
        getRowId={(row) => row.name}
        noDataMessage={Messages.empty}
        enableGlobalFilter
        enableHiding={false}
        enableRowSelection
        onRowSelectionChange={setRowSelection}
        positionToolbarAlertBanner="none"
        initialState={{ pagination: { pageSize: 25, pageIndex: 0 } }}
        muiTopToolbarProps={{
          sx: {
            [`& > .${boxClasses.root}`]: {
              alignItems: 'center',
              flexDirection: 'row-reverse',
            },
          },
        }}
        renderTopToolbarCustomActions={() => (
          <Stack
            gap={2}
            flex={1}
            direction="row"
            alignItems="center"
            justifyContent="space-between"
          >
            <TextField
              select
              size="small"
              label={Messages.filters.category}
              value={category}
              onChange={(event) => setCategory(event.target.value)}
              sx={{ minWidth: 200 }}
              slotProps={{ htmlInput: { 'data-testid': 'category-filter' } }}
            >
              {TEMPLATE_CATEGORY_OPTIONS.map((opt) => (
                <MenuItem key={opt.value} value={opt.value}>
                  {opt.label}
                </MenuItem>
              ))}
            </TextField>
            <Stack direction="row" gap={2} alignItems="center">
              {selectedTemplates.length > 0 && (
                <Button
                  variant="outlined"
                  startIcon={<FileDownloadOutlinedIcon />}
                  data-testid="export-selected-templates"
                  onClick={handleExportSelected}
                >
                  {Messages.actions.export}
                </Button>
              )}
              {canManage && (
                <Button
                  variant="contained"
                  startIcon={<AddOutlinedIcon />}
                  data-testid="add-alert-template"
                  onClick={() => setModal('create')}
                >
                  {Messages.addButton}
                </Button>
              )}
            </Stack>
          </Stack>
        )}
      />
      <CreateTemplateModal
        open={modal === 'create'}
        initialYaml={selected?.yaml}
        onClose={closeModal}
      />
      <ViewTemplateModal
        open={modal === 'view'}
        template={selected}
        onClose={closeModal}
      />
      <EditTemplateModal
        open={modal === 'edit'}
        template={selected}
        onClose={closeModal}
      />
      <DeleteTemplateModal
        open={modal === 'delete'}
        template={selected}
        onClose={closeModal}
      />
    </Page>
  );
};

export default AlertTemplates;
