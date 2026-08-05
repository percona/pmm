import { TemplateCategory, TemplateSource } from 'types/alert-templates.types';
import { Messages } from './AlertTemplates.messages';

export const ALERT_TEMPLATES_TABLE_NAME = 'alert-templates';

export const SOURCE_MAP: Record<TemplateSource, string> = {
  [TemplateSource.BUILT_IN]: Messages.source.builtIn,
  [TemplateSource.SAAS]: Messages.source.saas,
  [TemplateSource.USER_FILE]: Messages.source.userFile,
  [TemplateSource.USER_API]: Messages.source.userApi,
  [TemplateSource.UNSPECIFIED]: Messages.source.unknown,
};

export const TEMPLATE_CATEGORY_MAP: Record<TemplateCategory, string> = {
  [TemplateCategory.UNSPECIFIED]: Messages.category.unspecified,
  [TemplateCategory.PMM]: Messages.category.pmm,
  [TemplateCategory.MONGODB]: Messages.category.mongodb,
  [TemplateCategory.MYSQL]: Messages.category.mysql,
  [TemplateCategory.NODE]: Messages.category.node,
  [TemplateCategory.POSTGRESQL]: Messages.category.posgresql,
  [TemplateCategory.PROXYSQL]: Messages.category.proxysql,
  [TemplateCategory.VALKEY]: Messages.category.valkey,
  [TemplateCategory.HAPROXY]: Messages.category.haproxy,
};
