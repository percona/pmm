import type { FC } from 'react';
import { FieldGrid } from '../field-grid';
import { TextField } from '../fields';
import { FormSection } from '../form-section';
import { Messages } from './Labels.messages';
import type { LabelsProps } from './Labels.types';

const { labels } = Messages;

export const Labels: FC<LabelsProps> = ({ withCluster = true }) => (
  <FormSection title={Messages.title} description={Messages.description}>
    <FieldGrid>
      <TextField name="environment" label={labels.environment} />
      {withCluster && (
        <>
          <TextField name="cluster" label={labels.cluster} />
          <TextField name="replicationSet" label={labels.replicationSet} />
        </>
      )}
      <TextField
        name="region"
        label={labels.region}
        tooltip={Messages.tooltips.region}
      />
      <TextField name="az" label={labels.az} tooltip={Messages.tooltips.az} />
    </FieldGrid>
    <TextField
      name="customLabels"
      kind="textarea"
      label={labels.customLabels}
      placeholder={Messages.placeholders.customLabels}
      helperText={Messages.customLabelsDescription}
    />
  </FormSection>
);
