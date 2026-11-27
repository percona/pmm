import { z } from 'zod';
import { requiredText } from '../../../service-form/ServiceForm.schema';

export const azureCredentialsSchema = z.object({
  azureClientId: requiredText,
  azureClientSecret: requiredText,
  azureTenantId: requiredText,
  azureSubscriptionId: requiredText,
});
