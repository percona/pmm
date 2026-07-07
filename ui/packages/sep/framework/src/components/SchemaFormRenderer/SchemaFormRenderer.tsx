/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import {
  memo,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  type FormEvent,
} from 'react';
import {
  FormProvider,
  useForm,
  useFormContext,
  type SubmitHandler,
} from 'react-hook-form';
// UNSAFE_DataRouterContext is an unstable react-router API — pinned to react-router-dom ^7.6.0; review on version bumps.
import { UNSAFE_DataRouterContext, useBlocker } from 'react-router-dom';
import Accordion from '@mui/material/Accordion';
import AccordionDetails from '@mui/material/AccordionDetails';
import AccordionSummary from '@mui/material/AccordionSummary';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import Divider from '@mui/material/Divider';
import Typography from '@mui/material/Typography';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import type { FieldValidationError, PluginCapabilities } from '@sep/api';
import {
  AlertOnFailField,
  ALERT_ON_FAIL_FIELD_NAME,
} from '../AlertOnFailField';
import { ConditionalFieldSlot } from './ConditionalFieldSlot';
import { OneOfGroupSlot } from './OneOfGroupSlot';
import { useConditionalSection } from './hooks/useConditionalSection';
import { useCardinalityRules } from './hooks/useCardinalityRules';
import { useFailRules } from './hooks/useFailRules';
import { useUnsavedChangesGuard } from './hooks/useUnsavedChangesGuard';
import { coerceFormValues } from './utils/validationMapper';
import { getAtPath, setAtPath } from './utils/fieldPath';
import {
  collectOneOfGroups,
  flattenSectionFields,
  isOneOfGroup,
} from './utils/flattenSectionFields';
import type { FormSection, PluginField, RenderFieldOverride } from './types';

function fieldDefault(field: PluginField): unknown {
  switch (field.type) {
    case 'bool':
      return field.default ?? false;
    case 'integer':
    case 'float':
      return field.default ?? '';
    case 'multi_choice':
      return field.default ?? [];
    case 'file':
      return undefined;
    case 'service':
    case 'schema':
    case 'table':
    case 'host':
      return field.default ?? '';
    default:
      return field.default ?? '';
  }
}

function flattenFields(sections: FormSection[]): PluginField[] {
  return flattenSectionFields(sections);
}

function buildFormDefaults(
  sections: FormSection[],
  allFields: PluginField[],
  defaultValues: Record<string, unknown> | undefined,
  capabilities: PluginCapabilities | undefined
): Record<string, unknown> {
  const defaults: Record<string, unknown> = {};
  for (const field of allFields) {
    const seed = defaultValues
      ? getAtPath(defaultValues, field.name)
      : undefined;
    setAtPath(defaults, field.name, seed ?? fieldDefault(field));
  }
  for (const group of collectOneOfGroups(sections)) {
    const seed = defaultValues
      ? getAtPath(defaultValues, group.discriminator)
      : undefined;
    setAtPath(
      defaults,
      group.discriminator,
      seed ?? group.default ?? group.branches[0]?.value ?? ''
    );
  }
  if (capabilities?.alert_on_fail) {
    defaults[ALERT_ON_FAIL_FIELD_NAME] =
      defaultValues?.[ALERT_ON_FAIL_FIELD_NAME] ?? false;
  }
  return defaults;
}

interface SectionRendererProps {
  section: FormSection;
  idx: number;
  violations: Array<{ message: string }>;
  renderField?: RenderFieldOverride;
}

const SectionRenderer = memo(function SectionRenderer({
  section,
  idx,
  violations,
  renderField,
}: SectionRendererProps) {
  const { isHidden } = useConditionalSection(section);

  if (isHidden) {
    return null;
  }

  const sectionContent = (
    <>
      {section.description && (
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
          {section.description}
        </Typography>
      )}
      {violations.map((v, i) => (
        <Alert key={i} severity="error" sx={{ mb: 1 }}>
          {v.message}
        </Alert>
      ))}
      {section.fields.map((field) =>
        isOneOfGroup(field) ? (
          <OneOfGroupSlot
            key={field.name}
            group={field}
            renderField={renderField}
          />
        ) : (
          <ConditionalFieldSlot
            key={field.name}
            field={field}
            renderField={renderField}
          />
        )
      )}
    </>
  );

  return (
    <Box component="fieldset" sx={{ border: 0, p: 0, mb: 3 }}>
      {idx > 0 && <Divider sx={{ mb: 2 }} />}
      {section.collapsible ? (
        <Accordion
          defaultExpanded={!section.collapsed_by_default}
          disableGutters
          slotProps={{ transition: { unmountOnExit: true } }}
        >
          <AccordionSummary
            expandIcon={<ExpandMoreIcon />}
            sx={{
              pl: 2,
              pr: 1,
              minHeight: 48,
              '& .MuiAccordionSummary-content': { my: 1 },
            }}
          >
            <Typography
              component="legend"
              variant="subtitle1"
              sx={{ fontWeight: 600 }}
            >
              {section.title}
            </Typography>
          </AccordionSummary>
          <AccordionDetails sx={{ px: 0, pl: 2 }}>
            {sectionContent}
          </AccordionDetails>
        </Accordion>
      ) : (
        <>
          <Typography component="legend" variant="h6" sx={{ mb: 1, px: 0 }}>
            {section.title}
          </Typography>
          {sectionContent}
        </>
      )}
    </Box>
  );
});

export interface SchemaFormRendererProps {
  sections: FormSection[];
  onSubmit: (data: Record<string, unknown>) => void;
  submitLabel?: string;
  loading?: boolean;
  defaultValues?: Record<string, unknown>;
  /** Server-side error to show above the form (e.g. API failure from the caller's mutation). */
  submitError?: string | null;
  /**
   * Backend per-field validation errors (e.g. parsed from a 422) applied to the
   * form this renderer owns via react-hook-form `setError`. Each entry's `path`
   * is a react-hook-form field name; entries with an empty `path` are skipped
   * here (callers surface those through {@link submitError}). Pass a fresh array
   * on each submit failure — changing the array identity re-runs setError, and
   * previously applied errors are cleared before the new set is applied.
   */
  fieldErrors?: FieldValidationError[];
  /** Plugin capabilities. When `alert_on_fail` is true, renders <AlertOnFailField> below the sections. */
  capabilities?: PluginCapabilities;
  /**
   * Optional per-field widget override. Applied after the conditional gate
   * decides visibility / required-ness; receives the gate-resolved field and a
   * `renderDefault()` callback. Overrides must write through react-hook-form.
   * See {@link RenderFieldOverride}.
   */
  renderField?: RenderFieldOverride;
}

/**
 * Blocks in-app navigation while `isGuarded` is true.
 * Rendered only when the component tree is inside a Data Router, so `useBlocker`
 * never runs in contexts where it would throw (legacy MemoryRouter, test renders
 * without createMemoryRouter, etc.).
 */
function UnsavedChangesBlocker({ isGuarded }: { isGuarded: boolean }) {
  const shouldBlock = useCallback(
    ({
      currentLocation,
      nextLocation,
    }: {
      currentLocation: { pathname: string };
      nextLocation: { pathname: string };
    }) => isGuarded && currentLocation.pathname !== nextLocation.pathname,
    [isGuarded]
  );
  const blocker = useBlocker(shouldBlock);

  return (
    <Dialog
      open={blocker.state === 'blocked'}
      onClose={() => blocker.reset?.()}
      aria-labelledby="unsaved-dialog-title"
    >
      <DialogTitle id="unsaved-dialog-title">Unsaved changes</DialogTitle>
      <DialogContent>
        <DialogContentText>
          You have unsaved changes. If you leave this page your changes will be
          lost.
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => blocker.reset?.()}>Stay</Button>
        <Button
          onClick={() => blocker.proceed?.()}
          color="error"
          variant="outlined"
        >
          Discard changes
        </Button>
      </DialogActions>
    </Dialog>
  );
}

/** Inner form body — lives inside FormProvider so it can call useFormContext / useCardinalityRules. */
function SchemaFormBody({
  sections,
  onSubmit,
  submitLabel = 'Run',
  loading = false,
  submitError,
  fieldErrors,
  capabilities,
  renderField,
}: SchemaFormRendererProps) {
  const { handleSubmit, formState, setError, clearErrors, getFieldState } =
    useFormContext<Record<string, unknown>>();

  // Apply backend per-field errors to the form. Clear the paths set by the
  // previous failure first so a resubmit that fixes some fields does not leave
  // stale server errors on the corrected ones. Empty paths are form-level and
  // surfaced only through the submitError banner.
  const appliedServerErrorPaths = useRef<string[]>([]);
  useEffect(() => {
    if (appliedServerErrorPaths.current.length > 0) {
      clearErrors(appliedServerErrorPaths.current);
      appliedServerErrorPaths.current = [];
    }
    if (!fieldErrors?.length) {
      return;
    }
    const applied: string[] = [];
    for (const { path, message } of fieldErrors) {
      if (!path) {
        continue;
      }
      setError(path, { type: 'server', message });
      applied.push(path);
    }
    appliedServerErrorPaths.current = applied;
  }, [fieldErrors, setError, clearErrors]);

  const isGuarded = useUnsavedChangesGuard(submitError);
  const inDataRouter = Boolean(useContext(UNSAFE_DataRouterContext));
  const allFields = useMemo(() => flattenFields(sections), [sections]);
  const beforeSubmitSections = useMemo(
    () => sections.filter((section) => !section.render_after_submit),
    [sections]
  );
  const afterSubmitSections = useMemo(
    () => sections.filter((section) => section.render_after_submit),
    [sections]
  );
  const cardinalityViolations = useCardinalityRules(sections);
  const failViolations = useFailRules(sections);

  // Merge cardinality and fail violations per section into a flat list for SectionRenderer.
  const violationsBySection = useMemo(() => {
    const map = new Map<FormSection, Array<{ message: string }>>();
    sections.forEach((section, i) => {
      map.set(section, [
        ...(cardinalityViolations[i] ?? []),
        ...(failViolations[i] ?? []),
      ]);
    });
    return map;
  }, [sections, cardinalityViolations, failViolations]);

  const hasSectionViolations = useMemo(
    () => [...violationsBySection.values()].some((vs) => vs.length > 0),
    [violationsBySection]
  );
  const hasInlineErrors = Object.keys(formState.errors).length > 0;

  const handleFormSubmit: SubmitHandler<Record<string, unknown>> = (values) => {
    if (hasSectionViolations) {
      return;
    }
    onSubmit(coerceFormValues(values, allFields));
  };

  // Clear the server errors from the previous failure before react-hook-form's
  // validation gate runs. A 422 whose loc lands on a field with no mounted
  // input (unknown field, hidden conditional section, or array/nested path)
  // gets a setError entry that no input can ever clear; handleSubmit refuses to
  // call onValid while any error remains, which would wedge resubmission.
  // Those errors stay visible in the persistent banner regardless. Defined
  // inline (not memoized) so it always wraps the latest handleFormSubmit, which
  // closes over the current cardinality / fail_when violation state.
  const handleSubmitEvent = (event: FormEvent<HTMLFormElement>) => {
    if (appliedServerErrorPaths.current.length > 0) {
      clearErrors(appliedServerErrorPaths.current);
      appliedServerErrorPaths.current = [];
    }
    void handleSubmit(handleFormSubmit, () => {
      // Resubmit was blocked by a client-side validation error on some field, so
      // handleFormSubmit never fired: the parent won't re-send fieldErrors and the
      // effect won't re-run. Re-apply the server errors we cleared above (skipping
      // any field that now has its own client-side error) so their inline highlight
      // stays in sync with the still-visible persistent banner. Cleared again at the
      // top of the next submit, so this never re-introduces the resubmission wedge.
      if (!fieldErrors?.length) {
        return;
      }
      const reapplied: string[] = [];
      for (const { path, message } of fieldErrors) {
        if (path && !getFieldState(path).error) {
          setError(path, { type: 'server', message });
          reapplied.push(path);
        }
      }
      appliedServerErrorPaths.current = reapplied;
    })(event);
  };

  return (
    <>
      {inDataRouter && <UnsavedChangesBlocker isGuarded={isGuarded} />}
      <Box
        component="form"
        onSubmit={handleSubmitEvent}
        noValidate
        sx={{ maxWidth: 800 }}
      >
        {submitError && (
          <Alert severity="error" sx={{ mb: 2, whiteSpace: 'pre-line' }}>
            {submitError}
          </Alert>
        )}
        {hasInlineErrors && formState.isSubmitted && !submitError && (
          <Alert severity="error" sx={{ mb: 2 }}>
            Fix the highlighted fields before submitting.
          </Alert>
        )}

        {beforeSubmitSections.map((section, idx) => (
          <SectionRenderer
            key={`${section.title}-${idx}`}
            section={section}
            idx={idx}
            violations={violationsBySection.get(section) ?? []}
            renderField={renderField}
          />
        ))}

        {capabilities?.alert_on_fail && (
          <Box sx={{ mb: 2 }}>
            <AlertOnFailField />
          </Box>
        )}

        <Button
          type="submit"
          variant="contained"
          size="large"
          loading={loading}
          loadingPosition="start"
          sx={{ mt: 1 }}
        >
          {submitLabel}
        </Button>

        {afterSubmitSections.length > 0 ? (
          <Box sx={{ mt: 3 }}>
            {afterSubmitSections.map((section, idx) => (
              <SectionRenderer
                key={`${section.title}-after-${idx}`}
                section={section}
                idx={idx}
                violations={violationsBySection.get(section) ?? []}
                renderField={renderField}
              />
            ))}
          </Box>
        ) : null}
      </Box>
    </>
  );
}

export function SchemaFormRenderer(props: SchemaFormRendererProps) {
  const { sections, defaultValues, capabilities } = props;
  const allFields = useMemo(() => flattenFields(sections), [sections]);

  const formDefaults = useMemo(
    () => buildFormDefaults(sections, allFields, defaultValues, capabilities),
    [sections, allFields, defaultValues, capabilities]
  );

  const methods = useForm<Record<string, unknown>>({
    defaultValues: formDefaults,
  });

  return (
    <FormProvider {...methods}>
      <SchemaFormBody {...props} />
    </FormProvider>
  );
}
