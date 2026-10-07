export const Messages = {
  title: (nodeName: string) => `Alert thresholds: ${nodeName}`,
  loading: 'Loading thresholds…',
  empty: 'No alert rules support threshold overrides for this node.',
  error:
    "Couldn't load thresholds for this node. Close the dialog and try again.",
  actions: {
    cancel: 'Cancel',
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
    tooltips: {
      override:
        "Overrides the alert threshold for this node only. Other nodes keep the rule's default.",
    },
  },
  success: {
    updated: 'Alert thresholds updated',
  },
};
