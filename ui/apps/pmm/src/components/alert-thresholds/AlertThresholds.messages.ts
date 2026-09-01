export const Messages = {
  title: (nodeName: string) => `Alert thresholds: ${nodeName}`,
  loading: 'Loading thresholds…',
  empty: 'No overridable thresholds for this node.',
  actions: {
    cancel: 'Cancel and close',
    submit: 'Submit changes',
    reset: 'Reset to default',
  },
  table: {
    columns: {
      ruleTitle: 'Alert rule',
      parameter: 'Parameter',
      default: 'Default',
      override: 'Override',
      unit: 'Unit',
    },
  },
  success: {
    updated: 'Alert thresholds updated',
  },
};
