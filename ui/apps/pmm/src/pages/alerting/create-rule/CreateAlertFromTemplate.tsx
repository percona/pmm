import { FC, useEffect, useRef } from 'react';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import { zodResolver } from '@hookform/resolvers/zod';
import { FormProvider, useForm } from 'react-hook-form';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { enqueueSnackbar } from 'notistack';
import { Page } from 'components/page';
import { useAlertTemplates } from 'hooks/api/useAlertTemplates';
import { useCreateRuleFromTemplate } from 'hooks/api/useAlertTemplates';
import { useFolders } from 'hooks/api/useFolders';
import { Severity } from 'types/alert-templates.types';
import { PMM_ALERTING_TEMPLATES_PATH } from 'lib/constants';
import { Messages } from './CreateAlertFromTemplate.messages';
import {
  DEFAULT_DURATION_SECONDS,
  DEFAULT_INTERVAL_SECONDS,
} from './CreateAlertFromTemplate.constants';
import { CreateRuleFormValues } from './CreateAlertFromTemplate.types';
import { createRuleSchema } from './CreateAlertFromTemplate.schema';
import {
  buildCreateRulePayload,
  getTemplateDefaults,
} from './CreateAlertFromTemplate.utils';
import {
  AdvancedSection,
  DetailsSection,
  FiltersSection,
  FolderGroupSection,
  ParamsSection,
  TemplateSelectSection,
} from './sections';

export const CreateAlertFromTemplate: FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const presetTemplate = searchParams.get('template') ?? '';

  const { data, isLoading } = useAlertTemplates({ reload: true });
  const { data: folders = [], isLoading: loadingFolders } = useFolders();
  const { mutateAsync: createRule, isPending } = useCreateRuleFromTemplate();
  const templates = data?.templates ?? [];

  const methods = useForm<CreateRuleFormValues>({
    resolver: zodResolver(createRuleSchema),
    defaultValues: {
      template: presetTemplate,
      name: '',
      severity: Severity.WARNING,
      duration: String(DEFAULT_DURATION_SECONDS),
      folderUid: '',
      group: '',
      interval: String(DEFAULT_INTERVAL_SECONDS),
      filters: [],
      params: {},
    },
  });

  const { watch, setValue, handleSubmit } = methods;
  const templateName = watch('template');
  const selectedTemplate =
    templates.find((template) => template.name === templateName) ?? null;

  // Seed rule details / params defaults once per template selection.
  const seededFor = useRef<string | null>(null);
  useEffect(() => {
    if (!selectedTemplate || seededFor.current === selectedTemplate.name) {
      return;
    }
    seededFor.current = selectedTemplate.name;
    const defaults = getTemplateDefaults(selectedTemplate);
    (Object.keys(defaults) as (keyof CreateRuleFormValues)[]).forEach((key) => {
      setValue(key, defaults[key] as never, { shouldValidate: false });
    });
  }, [selectedTemplate, setValue]);

  const onSubmit = async (values: CreateRuleFormValues) => {
    if (!selectedTemplate) {
      return;
    }
    await createRule(buildCreateRulePayload(values, selectedTemplate), {
      onSuccess: () => {
        enqueueSnackbar(Messages.success, { variant: 'success' });
        navigate(PMM_ALERTING_TEMPLATES_PATH);
      },
      onError: (error) =>
        enqueueSnackbar(
          error instanceof Error ? error.message : Messages.error,
          { variant: 'error' }
        ),
    });
  };

  return (
    <Page title={Messages.title}>
      <FormProvider {...methods}>
        <Stack component="form" onSubmit={handleSubmit(onSubmit)} gap={4}>
          <TemplateSelectSection templates={templates} />
          <ParamsSection template={selectedTemplate} />
          <DetailsSection />
          <FolderGroupSection
            folders={folders}
            loadingFolders={loadingFolders}
          />
          <FiltersSection />
          <AdvancedSection template={selectedTemplate} />
          <Stack direction="row" gap={2} justifyContent="flex-end">
            <Button
              type="button"
              variant="text"
              data-testid="create-rule-cancel"
              onClick={() => navigate(PMM_ALERTING_TEMPLATES_PATH)}
            >
              {Messages.cancel}
            </Button>
            <Button
              type="submit"
              variant="contained"
              data-testid="create-rule-submit"
              disabled={isPending || isLoading || !selectedTemplate}
            >
              {Messages.submit}
            </Button>
          </Stack>
        </Stack>
      </FormProvider>
    </Page>
  );
};

export default CreateAlertFromTemplate;
