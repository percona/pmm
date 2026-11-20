import { getErrorMessage } from '../../service-form/ServiceForm.utils';
import { AWS_NO_CREDENTIALS_ERROR } from './RdsDiscovery.constants';
import { Messages } from './RdsDiscovery.messages';

export const getRdsDiscoveryError = (error: unknown) => {
  const message = getErrorMessage(error);
  return message.includes(AWS_NO_CREDENTIALS_ERROR)
    ? Messages.noCredentials
    : message;
};
