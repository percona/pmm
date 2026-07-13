import { stripQanServiceId } from 'utils/qanServiceId';
import type { QanLabelsMap, QanQueryExample } from 'types/qan.types';

const ALL_VALUES = new Set(['$__all', 'All', '']);

/** Bare UUID or `/service_id/{uuid}` style id from QAN labels. */
export function serviceUuidFromLabels(labels: QanLabelsMap): string {
  const raw = labels.service_id?.find((v) => v && !ALL_VALUES.has(v));
  if (!raw) return '';
  return stripQanServiceId(raw);
}

export function serviceIdFromExample(examples?: QanQueryExample[]): string {
  const ex = examples?.find((e) => e.serviceId && (e.example || e.exampleType));
  return ex?.serviceId ? stripQanServiceId(ex.serviceId) : '';
}

export function resolveServiceUuid(
  labels: QanLabelsMap,
  managedServices: { serviceId: string; serviceName: string }[] | undefined,
  exampleServiceId?: string
): string {
  const fromLabel = serviceUuidFromLabels(labels);
  if (fromLabel) return fromLabel;

  const fromExample = exampleServiceId ? stripQanServiceId(exampleServiceId) : '';
  if (fromExample) return fromExample;

  const name = labels.service_name?.find((v) => v && !ALL_VALUES.has(v));
  if (name && managedServices?.length) {
    const match = managedServices.find((s) => s.serviceName === name);
    if (match) return match.serviceId;
  }

  return '';
}
