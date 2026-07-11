import { TemplateSource } from 'types/alert-templates.types';
import { Messages } from './AlertTemplates.messages';

export const ALERT_TEMPLATES_TABLE_NAME = 'alert-templates';

export const SOURCE_MAP: Record<TemplateSource, string> = {
  [TemplateSource.BUILT_IN]: Messages.source.builtIn,
  [TemplateSource.SAAS]: Messages.source.saas,
  [TemplateSource.USER_FILE]: Messages.source.userFile,
  [TemplateSource.USER_API]: Messages.source.userApi,
  [TemplateSource.UNSPECIFIED]: Messages.source.unknown,
};
