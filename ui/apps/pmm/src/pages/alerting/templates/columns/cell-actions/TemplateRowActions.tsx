import { FC, useState } from 'react';
import IconButton from '@mui/material/IconButton';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import { useNavigate } from 'react-router-dom';
import { Template } from 'types/alert-templates.types';
import { isTemplateEditable } from 'utils/alert-templates.utils';
import { PMM_ALERTING_NEW_FROM_TEMPLATE_PATH } from 'lib/constants';
import { Messages } from '../../AlertTemplates.messages';

interface Props {
  template: Template;
  canManage: boolean;
  onView: (template: Template) => void;
  onEdit: (template: Template) => void;
  onDelete: (template: Template) => void;
}

export const TemplateRowActions: FC<Props> = ({
  template,
  canManage,
  onView,
  onEdit,
  onDelete,
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

  return (
    <>
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
    </>
  );
};

export default TemplateRowActions;
