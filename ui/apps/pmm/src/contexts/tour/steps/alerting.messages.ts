// Tip paragraphs keyed by the id of the Alerts menu item they describe.
// todo: need Catalinas input here
export const Messages: Record<string, string[]> = {
  'alerts-rules': [
    'Alert rules specify the conditions for firing alerts.',
    'Start from your available templates to pre-fill some of these conditions, and create your rule by adding more conditions and changing the template values.',
    "If you need to create an alert rule from scratch, use Grafana's complex alerting configuration instead.",
  ],
  'alerts-status': [
    'View the current state of alerts in your environment and their associated nodes',
  ],
  'alerts-templates': [
    'Templates help you create complex alert rules effortlessly.',
    'Percona offers a set of default templates for common events and expressions, but you can also create custom templates to fit your unique requirements.',
  ],
  'alerts-silences': [
    'Create silences when you want to stop notifications from one or more alerting rules.',
    'Silences will prevent notifications from getting created, but they will not prevent alerting rules from being evaluated or recorded in the Fired alerts tab. A silence only lasts for a specified window of time.',
  ],
  'alerts-groups': [
    'Alert groups show grouped alerts.',
    "Group common alerts into a single alert group to ensure that PMM doesn't fire duplicate alerts.",
  ],
  'alerts-settings': [
    'Use this to configure Alertmanagers in raw JSON format.',
  ],
  'alerts-contact-points': [
    'Define how your contacts are notified when an alert fires.',
    'Grafana supports a multitude of ChatOps tools to ensure that your team is notified via the most relevant channel.',
  ],
  'alerts-policies': [
    'Set where, when, and how the alerts get routed.',
    'A notification policy has a contact point assigned to it that consists of one or more notifiers.',
  ],
};
