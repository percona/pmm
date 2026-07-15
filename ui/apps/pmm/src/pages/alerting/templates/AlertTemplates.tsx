import { FC, useMemo, useState } from 'react';
import Button from '@mui/material/Button';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import { boxClasses } from '@mui/material/Box';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import { Table } from '@percona/percona-ui';
import { useUser } from 'contexts/user';
import { useAlertTemplates } from 'hooks/api/useAlertTemplates';
import { Template } from 'types/alert-templates.types';
import {
  getTemplateCategories,
  getTemplateCategory,
} from 'utils/alert-templates.utils';
import { Page } from 'components/page';
import { Messages } from './AlertTemplates.messages';
import { ALERT_TEMPLATES_TABLE_NAME } from './AlertTemplates.constants';
import { getAlertTemplatesColumns } from './columns/AlertTemplates.columns';
import { CreateTemplateModal } from './modal-create-template';
import { EditTemplateModal } from './modal-edit-template';
import { DeleteTemplateModal } from './modal-delete-template';
import { ViewTemplateModal } from './modal-view-template';

type ModalType = 'create' | 'view' | 'edit' | 'delete' | null;

const ALL_CATEGORIES = '__all__';

export const AlertTemplates: FC = () => {
  const { user } = useUser();
  const canManage = !!user?.isPMMAdmin;
  const { data, isLoading } = useAlertTemplates({ reload: true });
  const [modal, setModal] = useState<ModalType>(null);
  const [selected, setSelected] = useState<Template | null>(null);
  const [category, setCategory] = useState<string>(ALL_CATEGORIES);

  const templates = useMemo(() => data?.templates ?? [], [data]);
  const categories = useMemo(
    () => getTemplateCategories(templates),
    [templates]
  );
  const rows = useMemo(
    () =>
      category === ALL_CATEGORIES
        ? templates
        : templates.filter((t) => getTemplateCategory(t) === category),
    [templates, category]
  );

  const closeModal = () => {
    setModal(null);
    setSelected(null);
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
      }),
    [canManage]
  );

  return (
    <Page title={Messages.title} fullWidth>
      <Table
        tableName={ALERT_TEMPLATES_TABLE_NAME}
        columns={columns}
        data={rows}
        state={{ isLoading }}
        getRowId={(row) => row.name}
        noDataMessage={Messages.empty}
        enableGlobalFilter
        enableHiding={false}
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
          <Stack direction="row" gap={2} alignItems="center">
            <TextField
              select
              size="small"
              label={Messages.filters.category}
              value={category}
              onChange={(event) => setCategory(event.target.value)}
              sx={{ minWidth: 200 }}
              slotProps={{ htmlInput: { 'data-testid': 'category-filter' } }}
            >
              <MenuItem value={ALL_CATEGORIES}>
                {Messages.filters.allCategories}
              </MenuItem>
              {categories.map((c) => (
                <MenuItem key={c} value={c}>
                  {c}
                </MenuItem>
              ))}
            </TextField>
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
        )}
      />
      <CreateTemplateModal open={modal === 'create'} onClose={closeModal} />
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
