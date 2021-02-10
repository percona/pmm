# Troubleshoot

---

[TOC]

---

## Integrated Alerting

### No {{icon.bell}} Integrated Alerting icon

You are not logged in as a privileged user. You need either Admin or Editor roles to work with Integrated Alerting.

### {{icon.bell}} Integrated Alerting icon but no submenu

Integrated Alerting isn't activated.

1. Go to PMM --> PMM Settings --> Advanced Settings
2. Enable Integrated Alerting

### Unreachable external IP addresses

> When I get an email or page from my system the IP is not reachable from outside my organization how do I fix this?

You can configure your PMM Server’s Public Address by navigating to PMM --> PMM Settings --> Advanced Settings, and supply an address to use in your alert notifications.

### What is 'Alertmanager integration'?

> There’s already an Alertmanager integration tab without me turning it on, I know because I was using your existing Alertmanager integration.

This will continue to work but will be renamed *External Alertmanager*.

### Notification channels not working

> I tried to setup a Slack/Email channel but nothing happened

Before you can use a notification channel you must provide your connection details.

1. Go to PMM --> PMM Settings--> Communication
2. Define your SMTP server or Slack incoming webhook URL

For PagerDuty you can configure in the notification channel tab of Integrated Alerting by supplying your server/routing key.

### What's the difference: Username/Password vs Identity/Secret

> In configuring my email server I’m being asked for a Username and Password as well as Identity and Secret. What is the difference between these and which do I use or do I need both?

It depends on what kind of authentication your system uses:

- LOGIN: Use Username/Password
- PLAIN: Use either Username or Identity and Password
- CRAM-MD5: Use Username and Secret

### Alert Rule Templates is disabled

Built-In alerts are not editable.

However, you can copy them and edit the copies. (PMM >=2.14.0).

If you create a custom alert rule template you will have access to edit.

### Creating rules

> I’m ready to create my first rule!  I’ve chosen a template and given it a name...what is the format of the fields?

- Threshold - float value, it has different meanings depending on what template is used

- Duration - The duration the condition must be satisfied in seconds

- Filters - A Key, Evaluator, and Value. E.g. `service_name=ps5.7`

	- Key must be an exact match. You can find a complete list of keys by using the {{icon.compass }}*Explore* main menu item in PMM

	- Evaluator can be any of: `=`, `=~`

	- Value is an exact match or when used with a ‘fuzzy’ evaluator (=~) can be a regular expression. E.g. `service_name=~ps.*`

### Variables in Templates

> The concept of “template” implies things like variable substitutions...where can I use these? Where can I find a complete list of them?

Here is a guide to creating templates for Alertmanager: <https://prometheus.io/docs/prometheus/latest/configuration/template_examples/>
