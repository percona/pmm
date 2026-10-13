import type { RdsCredentials, RdsExporter } from 'types/agent.types';
import type { ChangeServiceParams, ManagedService } from 'types/services.types';
import type { EditServiceFormValues } from './EditService.schema';
import { RdsAuthMode } from './EditService.types';

export const toCustomLabelsText = (labels: Record<string, string>) =>
  Object.entries(labels)
    .map(([name, value]) => `${name}:${value}`)
    .join('\n');

// Only label names are restricted, so a value can hold colons too
export const splitCustomLabel = (entry: string): [string, string] => {
  const separator = entry.indexOf(':');

  return separator === -1
    ? [entry, '']
    : [entry.slice(0, separator), entry.slice(separator + 1)];
};

export const parseCustomLabels = (text: string): Record<string, string> =>
  Object.fromEntries(text.split(/\s/).filter(Boolean).map(splitCustomLabel));

export const getRdsAuthMode = (exporter: RdsExporter | null) => {
  if (exporter?.awsRoleArn) {
    return RdsAuthMode.iamRole;
  }

  if (exporter?.awsAccessKey || exporter?.isAwsSecretKeySet) {
    return RdsAuthMode.accessKey;
  }

  return RdsAuthMode.hostCredentials;
};

export const getInitialValues = (
  service: ManagedService,
  exporter: RdsExporter | null
): EditServiceFormValues => ({
  environment: service.environment,
  cluster: service.cluster,
  replicationSet: service.replicationSet,
  customLabels: toCustomLabelsText(service.customLabels),
  rdsAuthMode: getRdsAuthMode(exporter),
  awsAccessKey: exporter?.awsAccessKey ?? '',
  // The API never returns the secret key
  awsSecretKey: '',
  awsRoleArn: exporter?.awsRoleArn ?? '',
});

// The ChangeAgent fields for the chosen mode, or undefined when they are
// untouched. A role ARN is rejected next to an access key, so switching modes
// clears the abandoned fields (as `pmm-admin --aws-role-arn= ...` does).
export const toRdsCredentials = (
  values: EditServiceFormValues,
  exporter: RdsExporter | null
): RdsCredentials | undefined => {
  if (!exporter) {
    return undefined;
  }

  if (values.rdsAuthMode === RdsAuthMode.iamRole) {
    const awsRoleArn = values.awsRoleArn.trim();

    return awsRoleArn === exporter.awsRoleArn
      ? undefined
      : { awsRoleArn, awsAccessKey: '', awsSecretKey: '' };
  }

  if (values.rdsAuthMode === RdsAuthMode.accessKey) {
    const awsAccessKey = values.awsAccessKey.trim();
    const isUnchanged =
      awsAccessKey === exporter.awsAccessKey && !exporter.awsRoleArn;

    if (isUnchanged && !values.awsSecretKey) {
      return undefined;
    }

    // A blank secret key keeps the stored one
    return {
      awsAccessKey,
      awsRoleArn: '',
      ...(values.awsSecretKey && { awsSecretKey: values.awsSecretKey }),
    };
  }

  if (
    !exporter.awsAccessKey &&
    !exporter.awsRoleArn &&
    !exporter.isAwsSecretKeySet
  ) {
    return undefined;
  }

  return { awsAccessKey: '', awsSecretKey: '', awsRoleArn: '' };
};

const areLabelsEqual = (a: Record<string, string>, b: Record<string, string>) =>
  Object.keys(a).length === Object.keys(b).length &&
  Object.entries(a).every(([name, value]) => b[name] === value);

// The ChangeService fields that differ from the service, or undefined when
// none does
export const toServiceChanges = (
  service: ManagedService,
  values: EditServiceFormValues
): ChangeServiceParams | undefined => {
  const customLabels = parseCustomLabels(values.customLabels);
  const changes: Omit<ChangeServiceParams, 'serviceId'> = {
    ...(values.environment !== service.environment && {
      environment: values.environment,
    }),
    ...(values.cluster !== service.cluster && { cluster: values.cluster }),
    ...(values.replicationSet !== service.replicationSet && {
      replicationSet: values.replicationSet,
    }),
    ...(!areLabelsEqual(customLabels, service.customLabels) && {
      customLabels,
    }),
  };

  return Object.keys(changes).length
    ? { serviceId: service.serviceId, ...changes }
    : undefined;
};
