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
  useState,
  type FormEvent,
} from 'react';
import {
  FormProvider,
  get,
  useForm,
  useFormContext,
  type FieldErrors,
  type SubmitHandler,
} from 'react-hook-form';
import { FormFieldsProvider } from './formFieldsContext';
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
import { useFieldPayloadCleanup } from './hooks/useConditionalField';
import { OneOfGroupSlot } from './OneOfGroupSlot';
import {
  useConditionalSections,
  useUnregisterHiddenSections,
} from './hooks/useConditionalSection';
import { useCardinalityRules } from './hooks/useCardinalityRules';
import { useFailRules } from './hooks/useFailRules';
import { useUnsavedChangesGuard } from './hooks/useUnsavedChangesGuard';
import { coerceFormValues } from './utils/validationMapper';
import { fieldDefault } from './utils/fieldDefault';
import { getAtPath, setAtPath } from './utils/fieldPath';
import {
  collectOneOfGroups,
  flattenSectionFields,
  isOneOfGroup,
} from './utils/flattenSectionFields';
import type { FormSection, PluginField, RenderFieldOverride } from './types';

function flattenFields(sections: FormSection[]): PluginField[] {
  return flattenSectionFields(sections);
}

function scrollToFirstErrorField(
  errors: FieldErrors<Record<string, unknown>>,
  allFields: PluginField[],
  formEl: HTMLFormElement | null
): void {
  if (!formEl) {
    return;
  }
  const firstInvalid = allFields.find((field) => get(errors, field.name));
  if (!firstInvalid) {
    return;
  }
  const target = formEl.querySelector(
    `[data-field-name="${firstInvalid.name}"]`
  );
  if (target && typeof target.scrollIntoView === 'function') {
    target.scrollIntoView({ behavior: 'smooth', block: 'center' });
  }
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

/** One section paired with its position in the schema's own section list. */
interface SectionEntry {
  section: FormSection;
  index: number;
}

/**
 * Split sections into the ones a form opens on and the expert ones withheld
 * behind the reveal control, preserving each group's own order.
 *
 * Advanced sections are collected wherever they appear rather than having to be
 * adjacent, so a schema controls membership with one flag and nothing depends
 * on where the fields happen to be declared on the model.
 */
function partitionAdvanced(entries: SectionEntry[]): {
  standard: SectionEntry[];
  advanced: SectionEntry[];
} {
  const standard: SectionEntry[] = [];
  const advanced: SectionEntry[] = [];
  for (const entry of entries) {
    (entry.section.advanced ? advanced : standard).push(entry);
  }
  return { standard, advanced };
}

/**
 * Whether a section holds a value that differs from what the schema would have
 * pre-filled, judged against the values the form opened with.
 *
 * Read once, from the caller's `defaultValues`, rather than watched: this
 * decides whether an *unopened* advanced section has something in it worth
 * revealing, and a value the user has just typed means they opened it already.
 */
function sectionHasNonDefaultValue(
  section: FormSection,
  defaultValues: Record<string, unknown> | undefined
): boolean {
  if (!defaultValues) {
    return false;
  }
  return flattenSectionFields([section]).some((field) => {
    const seeded = getAtPath(defaultValues, field.name);
    if (seeded === undefined) {
      return false;
    }
    return !Object.is(seeded, fieldDefault(field));
  });
}

/**
 * Whether any field in a section currently carries a validation error.
 *
 * In an unrevealed section that means a backend error applied through
 * `setError`: an unmounted field is never registered, so react-hook-form's own
 * validation has nothing to run against it.
 */
function sectionHasError(
  section: FormSection,
  errors: FieldErrors<Record<string, unknown>>
): boolean {
  return flattenSectionFields([section]).some((field) =>
    Boolean(get(errors, field.name))
  );
}

interface SectionRendererProps {
  section: FormSection;
  idx: number;
  isHidden: boolean;
  violations: Array<{ message: string }>;
  renderField?: RenderFieldOverride;
  /**
   * Open a collapsible section that would otherwise start collapsed, because
   * it holds something the reader needs to see. Raising this later — when a
   * submit puts an error inside a collapsed section — opens it then; the
   * reader can still collapse it again afterwards.
   */
  forceExpanded?: boolean;
}

const SectionRenderer = memo(function SectionRenderer({
  section,
  idx,
  isHidden,
  violations,
  renderField,
  forceExpanded = false,
}: SectionRendererProps) {
  const [expanded, setExpanded] = useState(
    !section.collapsed_by_default || forceExpanded
  );

  useEffect(() => {
    if (forceExpanded) {
      setExpanded(true);
    }
  }, [forceExpanded]);

  // Mark the odd optional field out in a section that is otherwise required —
  // the absence of an asterisk is easy to miss when every neighbour has one.
  //
  // Deliberately narrow: one unfilled field among required ones. Two or more
  // and the section is a mix, where the asterisks already read as the
  // exception; and a field carrying a default is not blank-optional but
  // pre-filled, so "(optional)" would suggest a choice that has been made for
  // the reader already.
  const optionalNames = useMemo(() => {
    const leaves = section.fields.filter((f) => !isOneOfGroup(f));
    const required = leaves.filter((f) => f.required);
    const optional = leaves.filter((f) => !f.required);
    if (required.length < 2 || optional.length !== 1) {
      return new Set<string>();
    }
    const [only] = optional;
    const seeded = only.default !== undefined && only.default !== null;
    return seeded ? new Set<string>() : new Set([only.name]);
  }, [section.fields]);

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
            markOptional={optionalNames.has(field.name)}
          />
        )
      )}
    </>
  );

  // A collapsed shell already draws its own rule, and stacking several of them
  // is the bulk of an expert-heavy form's resting height — so no divider above
  // one, and a tighter gap between them.
  const showDivider = idx > 0 && !section.collapsible;

  return (
    <Box
      component="fieldset"
      sx={{ border: 0, p: 0, mb: section.collapsible ? 1 : 3 }}
    >
      {showDivider && <Divider sx={{ mb: 2 }} />}
      {section.collapsible ? (
        <Accordion
          expanded={expanded}
          onChange={(_, isExpanded) => setExpanded(isExpanded)}
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
          <AccordionDetails sx={{ pl: 2, pr: 2 }}>
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

/**
 * The control that reveals a form's advanced sections.
 *
 * A link-style button rather than another accordion: the sections it reveals
 * are themselves collapsible, and wrapping them in a further shell puts the
 * common case two disclosures deep. Once revealed it stays out of the way —
 * there is no "hide again", because collapsing a section the reader has
 * already put a value in is the behaviour this whole control exists to avoid.
 */
function AdvancedReveal({ onReveal }: { onReveal: () => void }) {
  return (
    <Box sx={{ mb: 3 }}>
      <Button
        type="button"
        variant="text"
        size="small"
        onClick={onReveal}
        startIcon={<ExpandMoreIcon />}
        data-testid="show-advanced-options"
        sx={{ px: 0.5 }}
      >
        Show advanced options
      </Button>
    </Box>
  );
}

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
        <Button onClick={() => blocker.proceed?.()} variant="contained">
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
  defaultValues,
  submitError,
  fieldErrors,
  capabilities,
  renderField,
}: SchemaFormRendererProps) {
  const { handleSubmit, formState, setError, clearErrors, getFieldState } =
    useFormContext<Record<string, unknown>>();
  const formRef = useRef<HTMLFormElement>(null);

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
  // Evaluated once for every section, so a group shell can know whether all of
  // its members are gated out before it renders.
  const hiddenSections = useConditionalSections(sections);
  // Both owned here, not by the section or field components: anything inside a
  // collapsed shell never mounts, so it cannot drop its own values when it is
  // gated out, and `buildFormDefaults` has already seeded them.
  useUnregisterHiddenSections(sections, hiddenSections);
  useFieldPayloadCleanup(allFields);
  const { standardEntries, advancedEntries, afterSubmitEntries } =
    useMemo(() => {
      const entries = sections.map((section, index) => ({ section, index }));
      const { standard, advanced } = partitionAdvanced(
        entries.filter(({ section }) => !section.render_after_submit)
      );
      return {
        standardEntries: standard,
        advancedEntries: advanced,
        afterSubmitEntries: entries.filter(
          ({ section }) => section.render_after_submit
        ),
      };
    }, [sections]);

  // An advanced section arriving with a value already in it — an edit form, or
  // a task cloned from another — must not be hidden behind the reveal.
  const seededAdvanced = useMemo(
    () =>
      new Set(
        advancedEntries
          .filter(({ section }) =>
            sectionHasNonDefaultValue(section, defaultValues)
          )
          .map(({ index }) => index)
      ),
    [advancedEntries, defaultValues]
  );
  const erroredAdvanced = useMemo(
    () =>
      new Set(
        advancedEntries
          .filter(({ section }) => sectionHasError(section, formState.errors))
          .map(({ index }) => index)
      ),
    [advancedEntries, formState.errors]
  );
  const [advancedRevealed, setAdvancedRevealed] = useState(false);
  // Reveal, never re-hide: a submit that puts an error in an advanced section
  // has to show it, and pulling the section back once the reader fixes it
  // would move the ground under them.
  const showAdvanced =
    advancedRevealed || seededAdvanced.size > 0 || erroredAdvanced.size > 0;
  const visibleAdvanced = advancedEntries.filter(
    ({ index }) => !(hiddenSections[index] ?? false)
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

  const renderSection = (
    entry: SectionEntry,
    idx: number,
    keyPrefix: string
  ) => (
    <SectionRenderer
      key={`${keyPrefix}-${entry.section.title}-${entry.index}`}
      section={entry.section}
      idx={idx}
      isHidden={hiddenSections[entry.index] ?? false}
      violations={violationsBySection.get(entry.section) ?? []}
      renderField={renderField}
      forceExpanded={
        seededAdvanced.has(entry.index) || erroredAdvanced.has(entry.index)
      }
    />
  );

  const handleFormSubmit: SubmitHandler<Record<string, unknown>> = (values) => {
    onSubmit(coerceFormValues(values, allFields));
  };

  // Clear the server errors from the previous failure before react-hook-form's
  // validation gate runs. A 422 whose loc lands on a field with no mounted
  // input (unknown field, hidden conditional section, or array/nested path)
  // gets a setError entry that no input can ever clear; handleSubmit refuses to
  // call onValid while any error remains, which would wedge resubmission.
  // Those errors stay visible in the persistent banner regardless. Defined
  // inline (not memoized) so it always sees the current cardinality /
  // fail_when violation state and the latest handleFormSubmit.
  const handleSubmitEvent = (event: FormEvent<HTMLFormElement>) => {
    if (hasSectionViolations) {
      // Section-level rules render their own inline Alerts. Stop before
      // react-hook-form runs, so it never flags the submit as successful —
      // that would disarm `useUnsavedChangesGuard` (isDirty && !isSubmitSuccessful)
      // for good on this path, since no submitError ever arrives to re-arm it.
      event.preventDefault();
      return;
    }
    if (appliedServerErrorPaths.current.length > 0) {
      clearErrors(appliedServerErrorPaths.current);
      appliedServerErrorPaths.current = [];
    }
    void handleSubmit(handleFormSubmit, (errors) => {
      scrollToFirstErrorField(errors, allFields, formRef.current);
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
    <FormFieldsProvider value={allFields}>
      {inDataRouter && <UnsavedChangesBlocker isGuarded={isGuarded} />}
      <Box
        component="form"
        ref={formRef}
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

        {standardEntries.map((entry, idx) =>
          renderSection(entry, idx, 'standard')
        )}

        {visibleAdvanced.length > 0 &&
          (showAdvanced ? (
            visibleAdvanced.map((entry, idx) =>
              renderSection(entry, standardEntries.length + idx, 'advanced')
            )
          ) : (
            <AdvancedReveal onReveal={() => setAdvancedRevealed(true)} />
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

        {afterSubmitEntries.length > 0 ? (
          <Box sx={{ mt: 3 }}>
            {afterSubmitEntries.map((entry, idx) =>
              renderSection(entry, idx, 'after')
            )}
          </Box>
        ) : null}
      </Box>
    </FormFieldsProvider>
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
