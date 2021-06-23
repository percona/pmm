# Integrated Alerting

!!! caution alert alert-warning "Caution"
    Integrated Alerting is a [technical preview](../details/glossary.md#technical-preview) and is subject to change.

*Integrated Alerting* lets you know when certain system events occur.

**To activate *Integrated Alerting***, select *{{icon.cog}} Configuration-->{{icon.setting}} Settings-->Advanced Settings*, turn on *Integrated Alerting* and click *Apply changes*.

This short (3:36) video shows how to activate and configure Integrated Alerting.

<video width="100%" controls>
  <source src="../_images/Integrated-Alerting.mp4" type="video/mp4">
  Your browser does not support playing this video.
</video>


<!-- incomplete flow diagram
```plantuml
@startuml
!include docs/_images/plantuml_styles.puml
|//PMM Settings//|
start
:Activate //Integrated Alerting//;
if (//Communication// settings) then
    :Email;
else
    :Slack;
endif
|//Integrated Alerting//|
:Add //Notification Channel//;
if (Alert Rule Template?) then
    :Select;
else
    |Server|
    -[dashed]->
    :Create;
    note right
        Edit on filesystem
    end note
    -[dashed]->
endif
|//Integrated Alerting//|
:Add //Alert Rule//;
stop
@enduml
```
-->
## Definitions

- Alerts are generated when their criteria (*alert rules*) are met; an *alert* is the result of an *alert rule* expression evaluating to *true*.
- Alert rules are based on *alert rule templates*. We provide a default set of templates. You can also create your own.

!!! note alert alert-primary ""
    PMM's *Integrated Alerting* is a customized and separate instance of the Prometheus Alertmanager, and distinct from Grafana's alerting functionality.

## Before you start

- Set up a communication channel:

    - When the *Communication* tab appears, select it.
    - Enter details for *Email* or *Slack*. ([Read more.](../how-to/configure.md#communication))

## Open the *Integrated Alerting* page

- From the left menu, select *{{icon.bell}} Alerting-->{{icon.listul}} Integrated Alerting*

!!! note alert alert-primary ""
    The *Alerting* menu also lists {{icon.listul}} *Alert Rules* and {{icon.commentshare}} *Notification Channels*. These are for Grafana's alerting functionality.

This page has four tabs.

1. *Alerts*: Shows alerts (if any).

    ![!](../_images/PMM_Integrated_Alerting_Alerts.jpg)

2. *Alert Rules*: Shows rule definitions.

    ![!](../_images/PMM_Integrated_Alerting_Alert_Rules.jpg)

3. *Alert Rule Templates*: Lists rule templates.

    ![!](../_images/PMM_Integrated_Alerting_Alert_Rule_Templates.jpg)

4. *Notification Channels*: Lists notification channels.

    ![!](../_images/PMM_Integrated_Alerting_Notification_Channels.jpg)

## Add a Notification Channel

1. On the *Integrated Alerting* page, go to the *Notification Channels* tab.

2. Click {{icon.plussquare}} *Add*.

3. Fill in the details:

    ![!](../_images/PMM_Integrated_Alerting_Notification_Channels_Add_Form.jpg)

    - Name
    - Type
        - Email:
            - Addresses
        - Pager Duty
            - Routing key
            - Service key
        - Slack
            - Channel

4. Click *Add* to add the notification channel, or *Cancel* to abort the operation.

## Add an Alert Rule

1. On the *Integrated Alerting* page, go to the *Alert Rules* tab.

2. Click {{icon.plussquare}} *Add*.

3. Fill in the details

    ![!](../_images/PMM_Integrated_Alerting_Alert_Rules_Add_Form.jpg)

    - Template
    - Name
    - Threshold
    - Duration(s)
    - Severity
    - Filters
    - Channels
    - Activate

4. Click *Add* to add the alert rule, or *Cancel* to abort the operation.

## Add an Alert Rule Template

1. On the *Integrated Alerting* page, go to the *Alert Rule Templates* tab.

2. Click {{icon.plussquare}} *Add*.

3. Enter a template in the *Alert Rule Template* text box.

    ```
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
    ```
    {% endraw %}

    ![!](../_images/PMM_Integrated_Alerting_Alert_Rule_Templates_Add_Form.jpg)

    !!! note alert alert-primary ""
        **Alert Rule Template parameters**

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

4. Click *Add* to add the alert rule template, or *Cancel* to abort the operation.
