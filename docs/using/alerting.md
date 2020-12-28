# Integrated alerting

---

[TOC]

---

!!! alert alert-warning "Warning"
    Integrated alerting is a technical preview and is subject to change.

## Alerts

An alert has a name, a summary, a description, and a definition as a set of rules.

- Name: A unique name for the alert
- Summary: A short summary
- Description: A long description

The definition includes:

- Frequency: How often to check whether the rule matches any events
- Service level agreement/objective (SLA/SLO): an expression of the expected availability for the application (e.g. 99.99% uptime)

---

To open the *Integrated Alerting* page:

- From the left menu, select <i class="uil uil-bell"></i> *Alerting*, <i class="uil uil-list-ul"></i> *Integrated Alerting*

This page has four tabs.

1. *Alerts*: Lists any alerts.

    ![](../_images/PMM_Integrated_Alerting_Alerts.jpg)

2. *Alert Rules*: Lists rule definitions.

    ![](../_images/PMM_Integrated_Alerting_Alert_Rules.jpg)

3. *Alert Rule Templates*: Lists Alert rule templates.

    ![](../_images/PMM_Integrated_Alerting_Alert_Rule_Templates.jpg)

4. *Notification Channels*: Lists notification channels.

    ![](../_images/PMM_Integrated_Alerting_Notification_Channels.jpg)


## Add an Alert Rule

1. On the *Integrated Alerting* page, go to the *Alert Rules* tab.

2. Click <i class="uil uil-plus-square"></i> *Add*.

3. Fill in the details

    ![](../_images/PMM_Integrated_Alerting_Alert_Rules_Add_Form.jpg)

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

2. Click <i class="uil uil-plus-square"></i> *Add*.

3. Enter a template in the *Alert Rule Template* text box.

    ![](../_images/PMM_Integrated_Alerting_Alert_Rule_Templates_Add_Form.jpg)

4. Click *Add* to add the alert rule template, or *Cancel* to abort the operation.

## Add a Notification Channel

1. On the *Integrated Alerting* page, go to the *Notification Channels* tab.

2. Click <i class="uil uil-plus-square"></i> *Add*.

3. Fill in the details:

    ![](../_images/PMM_Integrated_Alerting_Notification_Channels_Add_Form.jpg)

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
