import { FilterType, Severity } from 'types/alert-templates.types';

export interface RuleFilterFormValue {
  type: FilterType;
  label: string;
  regexp: string;
}

export type RuleParamFormValue = number | boolean | string;

export interface CreateRuleFormValues {
  template: string;
  name: string;
  severity: Severity;
  // Kept as strings because the number inputs bind string values; coerced in
  // buildCreateRulePayload.
  duration: string;
  folderUid: string;
  // Set when folderUid is the "create new folder" sentinel; the folder is
  // created on submit.
  newFolderTitle: string;
  group: string;
  interval: string;
  filters: RuleFilterFormValue[];
  params: Record<string, RuleParamFormValue>;
}
