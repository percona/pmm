import { FC } from 'react';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
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
  onEdit: (template: Template) => void;
  onDelete: (template: Template) => void;
}

export const TemplateRowActions: FC<Props> = ({
  template,
  canManage,
  onEdit,
  onDelete,
}) => {
  const navigate = useNavigate();
  const editable = canManage && isTemplateEditable(template);

  return (
    <Stack direction="row" alignItems="center" gap={1}>
      <Button
        size="small"
        color="primary"
        startIcon={<AddOutlinedIcon />}
        data-testid="create-alert-rule"
        onClick={() =>
          navigate(
            `${PMM_ALERTING_NEW_FROM_TEMPLATE_PATH}?template=${encodeURIComponent(
              template.name
            )}`
          )
        }
      >
        {Messages.actions.createRule}
      </Button>
      {canManage && (
        <>
          <Tooltip title={Messages.actions.edit}>
            <span>
              <IconButton
                size="small"
                disabled={!editable}
                data-testid="edit-alert-template"
                onClick={() => onEdit(template)}
              >
                <EditOutlinedIcon fontSize="small" />
              </IconButton>
            </span>
          </Tooltip>
          <Tooltip title={Messages.actions.delete}>
            <span>
              <IconButton
                size="small"
                disabled={!editable}
                data-testid="delete-alert-template"
                onClick={() => onDelete(template)}
              >
                <DeleteOutlineIcon fontSize="small" />
              </IconButton>
            </span>
          </Tooltip>
        </>
      )}
    </Stack>
  );
};

export default TemplateRowActions;
