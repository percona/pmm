#!/usr/bin/env python3
import argparse
import requests

# Copyright (C) 2017 Percona LLC
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU Affero General Public License for more details.#  //
# You should have received a copy of the GNU Affero General Public License
# along with this program. If not, see <https://www.gnu.org/licenses/>.


# This script migrates Integrated Alerting alert rules to the new Alerting system that was introduced in PMM 2.31
# Migration is partial, it covers only alert rules but not Notification Channels, Silences, etc...


def remove_prefix(string, prefix):
    if string.startswith(prefix):
        return string[len(prefix):]
    return string


def prepare_labels(rule):
    custom_labels = rule.get("custom_labels", {})
    labels = rule.get("labels", {})
    labels.update(custom_labels)
    labels.update({
        "percona_alerting": "1",
        "severity": remove_prefix(rule.get("severity", ""), "SEVERITY_").lower(),
        "template_name": rule.get("template_name", "")
    })
    return labels


def prepare_annotations(rule):
    annotations = rule.get("annotations", {})
    annotations.update({"rule": rule.get("name")})

    return annotations


def prepare_expression(rule):
    expr = rule.get("expr", "")

    if "filters" not in rule:
        return expr

    for f in rule["filters"]:
        key = f.get("key", "")
        value = f.get("value", "")
        expr = f'label_match({expr}, "{key}", "{value}")'

    return expr


def convert_rule(rule, datasource_uid):
    return {
        "grafana_alert": {
            "title": rule.get("name", "") + "_" + rule.get("rule_id", ""),
            "condition": "A",
            "no_data_state": "OK",
            "exec_err_state": "Alerting",
            "data": [
                {
                    "refId": "A",
                    "datasourceUid": datasource_uid,
                    "relativeTimeRange": {"from": 600, "to": 0},
                    "model": {
                        "expr": prepare_expression(rule),
                        "refId": "A",
                        "instant": True,
                    },
                },
            ]
        },
        "for": rule.get("for"),
        "annotations": prepare_annotations(rule),
        "labels": prepare_labels(rule),
    }


def main():
    parser = argparse.ArgumentParser(description="Migration script for Integrated Alerting alert rules")
    parser.add_argument("-u", "--user", required=True, help="PMM user login")
    parser.add_argument("-p", "--password", required=True, help="PMM user password")
    parser.add_argument("-s", "--server-url", default="http://localhost/", help="PMM server URL (default: %(default)s)")
    parser.add_argument("-i", "--insecure", action="store_false", help="skip TLS certificates verification")
    parser.add_argument("-f", "--folder", default="Experimental",
                        help="folder in Grafana where to put migrated alert rules (default: %(default)s)")
    parser.add_argument("-g", "--group", default="migrated",
                        help="alert group name for migrated alert rules (default: %(default)s)")

    config = vars(parser.parse_args())
    auth = (config["user"], config["password"])

    # Get existing Integrated Alerting alert rules
    print("Request existing IA rules.")
    ia_rules_url = '{server_url}/v1/management/ia/Rules/List'.format(**config)
    ia_rules_resp = requests.post(ia_rules_url, auth=auth, verify=config["insecure"])
    ia_rules = ia_rules_resp.json()
    print("Request existing IA rules done.")

    if "rules" not in ia_rules:
        print("There are no rules to migrate, exiting.")
        return

    print("Found rules: {count}.".format(count=len(ia_rules["rules"])))

    # Create alert group for migrated rules
    print("Create alert group for migrated rules.")
    group_url = '{server_url}/graph/api/ruler/grafana/api/v1/rules/{folder}/{group}'.format(**config)
    group_resp = requests.get(group_url, auth=auth, verify=config["insecure"])
    alert_rules_group = group_resp.json()
    print("Create alert group for migrated rules done.")
    if "interval" not in alert_rules_group:
        alert_rules_group["interval"] = "1m"

    # Get Metrics datasource UID
    print("Get datasource UID.")
    datasource_url = '{server_url}/graph/api/datasources/1'.format(**config)
    datasource_resp = requests.get(datasource_url, auth=auth, verify=config["insecure"])
    datasource = datasource_resp.json()
    datasource_uid = datasource["uid"]
    print("Get datasource UID done.")

    # Convert IA rules and add them to alert group
    print("Convert IA rules.")
    for rule in ia_rules["rules"]:
        alert_rules_group["rules"].append(convert_rule(rule, datasource_uid))
    print("Convert IA rules done.")

    # Update alert group
    print("Send request to create migrated alerts.")
    rules_url = '{server_url}/graph/api/ruler/grafana/api/v1/rules/{folder}'.format(**config)
    resp = requests.post(rules_url, None, alert_rules_group, auth=auth, verify=config["insecure"])
    print("Send request to create migrated alerts done.")
    print("Server response:")
    print(resp.text)


if __name__ == "__main__":
    main()
