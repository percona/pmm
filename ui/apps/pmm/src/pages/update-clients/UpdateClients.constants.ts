import { TextSelectOption } from 'components/text-select/TextSelect.types';
import { Messages } from './UpdateClients.messages';
import { VersionsFilter } from './UpdateClients.types';

export const FILTER_OPTIONS: TextSelectOption<VersionsFilter>[] = [
  {
    label: Messages.filter.all,
    value: VersionsFilter.All,
  },
  {
    label: Messages.filter.update,
    value: VersionsFilter.Required,
  },
  {
    label: Messages.filter.critical,
    value: VersionsFilter.Critical,
  },
];
