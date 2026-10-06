import type { SvgIconProps } from '@mui/material/SvgIcon';
import type { ComponentType } from 'react';
import type { AddServiceType } from '../AddServicePage.types';

export interface SelectServiceTypeProps {
  showAzure: boolean;
}

export interface ServiceTypeCard {
  type: AddServiceType;
  icon: ComponentType<SvgIconProps>;
}
