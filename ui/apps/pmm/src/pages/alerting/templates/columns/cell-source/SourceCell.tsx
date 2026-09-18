import { FC } from 'react';
import { ChipProps } from '@mui/material/Chip';
import { Chip } from '@percona/percona-ui';
import { TemplateSource } from 'types/alert-templates.types';
import { SOURCE_MAP } from '../../AlertTemplates.constants';

interface Props {
  source: TemplateSource;
}

const SOURCE_COLOR: Record<TemplateSource, string> = {
  [TemplateSource.BUILT_IN]: 'default',
  [TemplateSource.SAAS]: 'info',
  [TemplateSource.USER_FILE]: 'secondary',
  [TemplateSource.USER_API]: 'primary',
  [TemplateSource.UNSPECIFIED]: 'default',
};

export const SourceCell: FC<Props> = ({ source }) => (
  <Chip
    size="small"
    variant="outlined"
    color={SOURCE_COLOR[source] as ChipProps['color']}
    label={SOURCE_MAP[source]}
    data-testid="alert-template-source"
  />
);

export default SourceCell;
