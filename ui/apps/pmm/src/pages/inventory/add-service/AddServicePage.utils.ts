import { PMM_ADD_SERVICE_PATH } from 'lib/constants';
import { Messages } from './AddServicePage.messages';
import { AddServiceType } from './AddServicePage.types';

export const isAddServiceType = (value?: string): value is AddServiceType =>
  Object.values<string | undefined>(AddServiceType).includes(value);

export const getAddServicePath = (type: AddServiceType) =>
  `${PMM_ADD_SERVICE_PATH}/${type}`;

export const getPageTitle = (type?: AddServiceType) => {
  switch (type) {
    case undefined:
      return Messages.titles.selection;
    case AddServiceType.external:
      return Messages.titles.external;
    case AddServiceType.rds:
    case AddServiceType.azure:
      return Messages.serviceTypes[type];
    default:
      return Messages.titles.configuring(Messages.serviceTypes[type]);
  }
};
