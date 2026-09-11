import { FC, useState } from 'react';
import IconButton from '@mui/material/IconButton';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import FileDownloadOutlinedIcon from '@mui/icons-material/FileDownloadOutlined';
import LibraryAddOutlinedIcon from '@mui/icons-material/LibraryAddOutlined';
import { useNavigate } from 'react-router-dom';
import { enqueueSnackbar } from 'notistack';
import { Template } from 'types/alert-templates.types';
import {
  getTemplateExportFilename,
  isTemplateEditable,
} from 'utils/alert-templates.utils';
import { copyToClipboard } from 'utils/clipboard.utils';
import { downloadTextFile } from 'utils/file.utils';
import { PMM_ALERTING_NEW_FROM_TEMPLATE_PATH } from 'lib/constants';
import { Messages } from '../../AlertTemplates.messages';
import Stack from '@mui/material/Stack';

interface Props {
  template: Template;
  canManage: boolean;
  onView: (template: Template) => void;
  onEdit: (template: Template) => void;
  onDelete: (template: Template) => void;
  onDuplicate: (template: Template) => void;
}

export const TemplateRowActions: FC<Props> = ({
  template,
  canManage,
  onView,
  onEdit,
  onDelete,
  onDuplicate,
}) => {
  const navigate = useNavigate();
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
  const open = !!anchorEl;
  const editable = canManage && isTemplateEditable(template);

  const close = () => setAnchorEl(null);

  const run = (action: () => void) => () => {
    close();
    action();
  };

  const handleCopy = async () => {
    const copied = await copyToClipboard(template.yaml);
    enqueueSnackbar(copied ? Messages.copy.success : Messages.copy.error, {
      variant: copied ? 'success' : 'error',
    });
  };

  const handleExport = () => {
    downloadTextFile(getTemplateExportFilename(template), template.yaml);
  };

  return (
    <Stack flex={1} direction="row" justifyContent="flex-end">
      <IconButton
        size="small"
        aria-label={Messages.columns.actions}
        data-testid="template-actions-menu"
        aria-haspopup="true"
        aria-expanded={open ? 'true' : undefined}
        onClick={(event) => setAnchorEl(event.currentTarget)}
      >
        <MoreVertIcon fontSize="small" />
      </IconButton>
      <Menu anchorEl={anchorEl} open={open} onClose={close}>
        <MenuItem
          data-testid="create-alert-rule"
          onClick={run(() =>
            navigate(
              `${PMM_ALERTING_NEW_FROM_TEMPLATE_PATH}?template=${encodeURIComponent(
                template.name
              )}`
            )
          )}
        >
          <ListItemIcon>
            <AddOutlinedIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>{Messages.actions.createRule}</ListItemText>
        </MenuItem>
        <MenuItem
          data-testid="view-alert-template"
          onClick={run(() => onView(template))}
        >
          <ListItemIcon>
            <VisibilityOutlinedIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>{Messages.actions.view}</ListItemText>
        </MenuItem>
        <MenuItem data-testid="copy-alert-template" onClick={run(handleCopy)}>
          <ListItemIcon>
            <ContentCopyIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>{Messages.actions.copy}</ListItemText>
        </MenuItem>
        <MenuItem
          data-testid="export-alert-template"
          onClick={run(handleExport)}
        >
          <ListItemIcon>
            <FileDownloadOutlinedIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>{Messages.actions.export}</ListItemText>
        </MenuItem>
        {canManage && (
          <MenuItem
            data-testid="duplicate-alert-template"
            onClick={run(() => onDuplicate(template))}
          >
            <ListItemIcon>
              <LibraryAddOutlinedIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText>{Messages.actions.duplicate}</ListItemText>
          </MenuItem>
        )}
        {editable && (
          <MenuItem
            data-testid="edit-alert-template"
            onClick={run(() => onEdit(template))}
          >
            <ListItemIcon>
              <EditOutlinedIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText>{Messages.actions.edit}</ListItemText>
          </MenuItem>
        )}
        {editable && (
          <MenuItem
            data-testid="delete-alert-template"
            onClick={run(() => onDelete(template))}
          >
            <ListItemIcon>
              <DeleteOutlineIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText>{Messages.actions.delete}</ListItemText>
          </MenuItem>
        )}
      </Menu>
    </Stack>
  );
};

export default TemplateRowActions;
