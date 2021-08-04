# Integrated Alerting

!!! caution alert alert-warning "Caution"
    Integrated Alerting is a [technical preview](../details/glossary.md#technical-preview) and is subject to change.

*Integrated Alerting* lets you know when certain system events occur.

- Alerts are generated when their criteria (*alert rules*) are met; an *alert* is the result of an *alert rule* expression evaluating to *true*.
- Alert rules are based on *alert rule templates*. We provide a default set of templates. You can also create your own.

!!! summary alert alert-info "Summary"
    - [Activate Integrated Alerting](#activate-integrated-alerting)
    - [Set up a communication channel](#set-up-a-communication-channel)
    - [Add a notification channel](#add-a-notification-channel)
    - [Add an alert rule](#add-an-alert-rule) (based on a built-in alert rule template)
    - (Optional) [Create your own alert rule template](#add-an-alert-rule-template)

This short video (3m 36s) shows how to activate and configure Integrated Alerting.

<video width="100%" controls>
  <source src="../_images/Integrated-Alerting.mp4" type="video/mp4">
  Your browser does not support playing this video.
</video>

## Before you start

Before you can get alerts, you must activate the feature, and set up a *communication channel* (define by how alerts should arrive, as emails or slack messages).

### Activate Integrated Alerting

1. Select <i class="uil uil-cog"></i> *Configuration* → <i class="uil uil-setting"></i> *Settings* → *Advanced Settings*.

1. Under *Technical preview features*, turn on *Integrated Alerting*.

1. Click *Apply changes*. A new *Communication* tab will appear.

### Set up a communication channel

1. When the *Communication* tab appears, select it.

1. Select the tab for an alert method, *Email* or *Slack*.

    1. For *Email*, enter values to define the SMTP email server

        - *Server Address*: The default SMTP smarthost used for sending emails, including port number.
        - *Hello*: The default hostname to identify to the SMTP server.
        - *From*: The sender's email address.
        - *Auth type*: Authentication type. Choose from:
            - *None*
            - *Plain*
            - *Login*
            - *CRAM-MD5*
        - *Username*: Username for SMTP Auth using CRAM-MD5, LOGIN and PLAIN.
        - *Password*: Password for SMTP Auth using CRAM-MD5, LOGIN and PLAIN.

    1. For *Slack*, enter a value for *URL*, the Slack webhook URL to use.

1. Click *Apply changes*.

1. From the left menu, select <i class="uil uil-bell"></i> *Alerting* → <i class="uil uil-list-ul"></i> *Integrated Alerting*. The default tab of the *Integrated Alerting* page lists alerts, if any are set up.

    ![!](../_images/PMM_Integrated_Alerting_Alerts.jpg)

!!! note alert alert-primary ""
    - The *Alerting* menu also lists <i class="uil uil-list-ul"></i> *Alert Rules* and <i class="uil uil-comment-alt-share"></i> *Notification Channels*. These are for Grafana's alerting functionality.
    - PMM's *Integrated Alerting* is a customized and separate instance of the Prometheus Alertmanager, and distinct from Grafana's alerting functionality.

## Add a Notification Channel

!!! note alert alert-primary ""
    A *notification channel* is a specific instance of a *communication channel*. For example, for email, the communication channel defines a server, while the notification channel specifies recipients (one or more email addresses) who receive alerts sent via the email server.

1. Select <i class="uil uil-bell"></i> *Alerting* → <i class="uil uil-list-ul"></i> *Integrated Alerting*.

1. Select the *Notification Channels* tab.

    ![!](../_images/PMM_Integrated_Alerting_Notification_Channels.jpg)

1. Click <i class="uil uil-plus-square"></i> *Add*.

1. Fill in the details:

    ![!](../_images/PMM_Integrated_Alerting_Notification_Channels_Add_Form.jpg)

    - Name:
    - Type:
        - Email:
            - Addresses:
        - Pager Duty:
            - Routing key:
            - Service key:
        - Slack:
            - Channel:

1. Click *Add* to add the notification channel, or *Cancel* to abort the operation.

## Add an Alert Rule

1. Select the *Alert Rules* tab.

    ![!](../_images/PMM_Integrated_Alerting_Alert_Rules.jpg)

1. Click <i class="uil uil-plus-square"></i> *Add*.

1. Fill in the details

    ![!](../_images/PMM_Integrated_Alerting_Alert_Rules_Add_Form.jpg)

    - Template:
    - Name:
    - Threshold:
    - Duration(s):
    - Severity:
    - Filters:
    - Channels:
    - Activate:

1. Click *Add* to add the alert rule, or *Cancel* to abort the operation.

## Add an Alert Rule Template

If the provided alert rule templates don't do what you want, you can create your own.

1. Select the *Alert Rule Templates* tab.

    ![!](../_images/PMM_Integrated_Alerting_Alert_Rule_Templates.jpg)

1. Click <i class="uil uil-plus-square"></i> *Add*.

1. Enter a template in the *Alert Rule Template* text box.

    ```yaml
    {% raw %}
    ---
    templates:
        - name: mysql_too_many_connections
          version: 1
          summary: MySQL connections in use
          tiers: [anonymous, registered]
          expr: |-
            max_over_time(mysql_global_status_threads_connected[5m]) / ignoring (job)
            mysql_global_variables_max_connections
            * 100
            > [[ .threshold ]]
          params:
            - name: threshold
              summary: A percentage from configured maximum
              unit: '%'
              type: float
              range: [0, 100]
              value: 80
          for: 5m
          severity: warning
          labels:
            foo: bar
          annotations:
            description: |-
                More than [[ .threshold ]]% of MySQL connections are in use on {{ $labels.instance }}
                VALUE = {{ $value }}
                LABELS: {{ $labels }}
            summary: MySQL too many connections (instance {{ $labels.instance }})
    {% endraw %}
    ```

    ![!](../_images/PMM_Integrated_Alerting_Alert_Rule_Templates_Add_Form.jpg)

    !!! note alert alert-primary "Alert Rule Template parameters"
        The parameters used in the template follow a format and might include different fields depending on their `type`:

        - `name` (required): the name of the parameter. Spaces and special characters not allowed.
        - `summary` (required): a short description of what this parameter represents.
        - `type` (required): PMM currently supports the `float` type. (More will be available in the future, such as `string` or `bool`.)
        - `unit` (optional): PMM currently supports either `s` (seconds) or `%` (percentage).
        - `value` (optional): the parameter value itself.
        - `range` (optional): only for `float` parameters, defining the boundaries for the value.

        **Restrictions**

        - Value strings must not include any of these special characters: `<` `>` `!` `@` `#` `$` `%` `^` `&` `*` `(` `)` `_` `/` `\` `'` `+` `-` `=` ` ` (space)
        - Any variables must be predefined.

1. Click *Add* to add the alert rule template, or *Cancel* to abort the operation.
