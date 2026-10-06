import type { FC } from 'react';
import { Navigate, useParams } from 'react-router-dom';
import { Page } from 'components/page';
import { useSettings } from 'contexts/settings';
import {
  PMM_ADD_SERVICE_PATH,
  PMM_INVENTORY_SERVICES_URL,
} from 'lib/constants';
import { OrgRole } from 'types/user.types';
import { AddServiceActions } from './add-service-actions';
import { AddServiceType } from './AddServicePage.types';
import { getPageTitle, isAddServiceType } from './AddServicePage.utils';
import { SelectServiceType } from './select-service-type';

export const AddServicePage: FC = () => {
  const { type } = useParams<{ type?: string }>();
  const { settings, isLoading } = useSettings();
  const serviceType = isAddServiceType(type) ? type : undefined;
  const isAzureEnabled = !!settings?.azurediscoverEnabled;
  const isUnavailable =
    (type !== undefined && !serviceType) ||
    (serviceType === AddServiceType.azure && !isLoading && !isAzureEnabled);

  if (isUnavailable) {
    return <Navigate to={PMM_ADD_SERVICE_PATH} replace />;
  }

  return (
    <Page title={getPageTitle(serviceType)} roles={[OrgRole.Admin]}>
      {!serviceType && <SelectServiceType showAzure={isAzureEnabled} />}
      <AddServiceActions
        cancelTo={
          serviceType ? PMM_ADD_SERVICE_PATH : PMM_INVENTORY_SERVICES_URL
        }
      />
    </Page>
  );
};
