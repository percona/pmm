#!/usr/bin/env python3
import argparse
import requests

# This script migrates Integrated Alerting alert rules to the new Alerting system that was introduced in PMM 2.31
# Migration is partial, it covers only alert rules but not Notification Channels, Silences, etc...

def prepare_labels(rule):
    labels = rule.get("labels", {})
    labels.update({
        "percona_alerting": "1",
        "severity": rule.get("severity", "").lstrip("SEVERITY_").lower(),
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


def convert_rule(rule):
    return {
        "grafana_alert": {
            "title": rule.get("name", "") + "_" + rule.get("rule_id", ""),
            "condition": "A",
            "no_data_state": "OK",
            "exec_err_state": "Alerting",
            "data": [
                {
                    "refId": "A",
                    "datasourceUid": datasourceUID,
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

# Create alert group for migrated rules
groupURL = '{server_url}/graph/api/ruler/grafana/api/v1/rules/{folder}/{group}'.format(**config)
groupReq = requests.get(groupURL, auth=auth, verify=config["insecure"])
alertRulesGroup = groupReq.json()
if "interval" not in alertRulesGroup:
    alertRulesGroup["interval"] = "1m"

# Get Metrics datasource UID
datasourceURL = '{server_url}/graph/api/datasources/1'.format(**config)
datasourceReq = requests.get(datasourceURL, auth=auth, verify=config["insecure"])
datasource = datasourceReq.json()
datasourceUID = datasource["uid"]

# Get existing Integrated Alerting alert rules
iaRulesURL = '{server_url}/v1/management/ia/Rules/List'.format(**config)
iaRulesReq = requests.post(iaRulesURL, auth=auth, verify=config["insecure"])
iaRules = iaRulesReq.json()

if "rules" not in iaRules:
    print("There are no rules to migrate")
    exit(1)

# Convert IA rules and add them to alert group
for rule in iaRules["rules"]:
    alertRulesGroup["rules"].append(convert_rule(rule))

# Update alert group
rulesURL = '{server_url}/graph/api/ruler/grafana/api/v1/rules/{folder}'.format(**config)
resp = requests.post(rulesURL, None, alertRulesGroup, auth=auth, verify=config["insecure"])
print(resp.text)
