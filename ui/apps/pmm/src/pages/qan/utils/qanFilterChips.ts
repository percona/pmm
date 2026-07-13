import { ALL_VARIABLE_TEXT, ALL_VARIABLE_VALUE } from './qanTools';
import { asStringList } from './qanNormalize';
import type { QanLabelsMap } from 'types/qan.types';

const hasAllValueOrText = (element: string) =>
  element !== ALL_VARIABLE_VALUE && element !== ALL_VARIABLE_TEXT;

export interface QanFilterChip {
  key: string;
  value: string;
  label: string;
}

export function getActiveFilterChips(labels: QanLabelsMap): QanFilterChip[] {
  const chips: QanFilterChip[] = [];
  Object.entries(labels).forEach(([key, values]) => {
    if (key === 'interval' || key === 'by') return;
    asStringList(values).filter(hasAllValueOrText).forEach((value) => {
      chips.push({
        key,
        value,
        label: `${key === 'service_id' ? 'service' : key.replace(/_/g, ' ')}: ${value}`,
      });
    });
  });
  return chips;
}

export function clearAllFilters(labels: QanLabelsMap): QanLabelsMap {
  const next: QanLabelsMap = {};
  Object.keys(labels).forEach((key) => {
    next[key] = [ALL_VARIABLE_VALUE];
  });
  return next;
}

export function removeFilterChip(
  labels: QanLabelsMap,
  key: string,
  value: string
): QanLabelsMap {
  const next: QanLabelsMap = { ...labels };
  const current = asStringList(next[key]);
  const filtered = current.filter((v) => v !== value);
  next[key] = filtered.length ? filtered : [ALL_VARIABLE_VALUE];
  return next;
}
