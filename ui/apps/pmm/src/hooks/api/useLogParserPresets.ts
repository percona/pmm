import {
  useMutation,
  useQuery,
  useQueryClient,
  UseMutationOptions,
  UseQueryOptions,
} from '@tanstack/react-query';
import {
  addLogParserPreset,
  changeLogParserPreset,
  listLogParserPresets,
  LogParserPreset,
  removeLogParserPreset,
} from 'api/logParserPresets';

export const LOG_PARSER_PRESETS_KEY = ['logParserPresets'] as const;

export const useLogParserPresets = (
  options?: Partial<UseQueryOptions<LogParserPreset[]>>
) =>
  useQuery({
    queryKey: LOG_PARSER_PRESETS_KEY,
    queryFn: listLogParserPresets,
    ...options,
  });

export const useAddLogParserPreset = (
  options?: Partial<UseMutationOptions<LogParserPreset, Error, Parameters<typeof addLogParserPreset>[0]>>
) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: addLogParserPreset,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: LOG_PARSER_PRESETS_KEY });
      options?.onSuccess?.(...args);
    },
    ...options,
  });
};

export const useChangeLogParserPreset = (
  options?: Partial<
    UseMutationOptions<
      LogParserPreset,
      Error,
      { id: string; description?: string; operatorYaml?: string }
    >
  >
) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...body }) => changeLogParserPreset(id, body),
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: LOG_PARSER_PRESETS_KEY });
      options?.onSuccess?.(...args);
    },
    ...options,
  });
};

export const useRemoveLogParserPreset = (
  options?: Partial<UseMutationOptions<void, Error, string>>
) => {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: removeLogParserPreset,
    onSuccess: (...args) => {
      qc.invalidateQueries({ queryKey: LOG_PARSER_PRESETS_KEY });
      options?.onSuccess?.(...args);
    },
    ...options,
  });
};
