import { type FC, useCallback, useState } from 'react';
import { Navigate, useParams } from 'react-router-dom';
import { Page } from 'components/page';
import { useSettings } from 'contexts/settings';
import { PMM_ADD_SERVICE_PATH } from 'lib/constants';
import { OrgRole } from 'types/user.types';
import { BackLink } from './components/back-link';
import { SERVICE_FORMS } from './AddServicePage.constants';
import { BackToListContext } from './AddServicePage.context';
import { Messages } from './AddServicePage.messages';
import { AddServiceType } from './AddServicePage.types';
import { getPageTitle, isAddServiceType } from './AddServicePage.utils';
import { SelectServiceType } from './select-service-type';

export const AddServicePage: FC = () => {
  const { type } = useParams<{ type?: string }>();
  const { settings, isLoading } = useSettings();
  const [backToList, setBackToList] = useState<{ onBack: () => void }>();
  const setOnBack = useCallback(
    (onBack?: () => void) => setBackToList(onBack && { onBack }),
    []
  );
  const serviceType = isAddServiceType(type) ? type : undefined;
  const isAzureEnabled = !!settings?.azurediscoverEnabled;
  const isUnavailable =
    (type !== undefined && !serviceType) ||
    // Settings stay null until they load, even when nothing reports loading yet.
    (serviceType === AddServiceType.azure &&
      !isLoading &&
      !!settings &&
      !isAzureEnabled);

  if (isUnavailable) {
    return <Navigate to={PMM_ADD_SERVICE_PATH} replace />;
  }

  const ServiceTypeForm = serviceType && SERVICE_FORMS[serviceType];
  const backLink =
    serviceType &&
    (backToList ? (
      <BackLink label={Messages.back.instances} onClick={backToList.onBack} />
    ) : (
      <BackLink label={Messages.back.selection} to={PMM_ADD_SERVICE_PATH} />
    ));

  return (
    <Page
      title={getPageTitle(serviceType)}
      topBar={backLink}
      roles={[OrgRole.Admin]}
    >
      {!serviceType && <SelectServiceType showAzure={isAzureEnabled} />}
      {ServiceTypeForm && (
        <BackToListContext.Provider value={setOnBack}>
          <ServiceTypeForm />
        </BackToListContext.Provider>
      )}
    </Page>
  );
};
