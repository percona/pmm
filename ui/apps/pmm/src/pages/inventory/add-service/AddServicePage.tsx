import type { FC } from 'react';
import { Navigate, useParams } from 'react-router-dom';
import { Page } from 'components/page';
import { useSettings } from 'contexts/settings';
import { PMM_ADD_SERVICE_PATH } from 'lib/constants';
import { OrgRole } from 'types/user.types';
import { SERVICE_FORMS } from './AddServicePage.constants';
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

  const ServiceTypeForm = serviceType && SERVICE_FORMS[serviceType];

  return (
    <Page title={getPageTitle(serviceType)} roles={[OrgRole.Admin]}>
      {!serviceType && <SelectServiceType showAzure={isAzureEnabled} />}
      {ServiceTypeForm && <ServiceTypeForm />}
    </Page>
  );
};
