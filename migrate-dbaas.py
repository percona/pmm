#!/usr/bin/env python3
import json
import subprocess

def get_clusters(cluster_type):
    output = subprocess.check_output(["kubectl", "get", cluster_type, "-o", "json"])
    return json.loads(output)


def create_cluster(cluster):
    p = subprocess.Popen(["kubectl", "apply", "-f", "-"], stdin=subprocess.PIPE)
    out = p.communicate(json.dumps(cluster).encode('utf-8'))


def convert_pxc(cluster):
    database_cluster = {
        "apiVersion": "dbaas.percona.com/v1",
        "kind": "DatabaseCluster",
        "metadata": {
            "namespace": cluster.get("metadata", {}).get("namespace", ""),
            "name": cluster.get("metadata", {}).get("name", ""),
            #"annotations": cluster.get("metadata", {}).get("annotations", {}),
            "finalizers": cluster.get("metadata", {}).get("finalizers", []),
        },
        "spec": {
            "databaseType": "pxc",
            "databaseConfig": cluster.get("spec", {}).get("pxc", {}).get("configuration"),
            "databaseImage": cluster.get("spec", {}).get("pxc", {}).get("image"),
            "secretsName": cluster.get("spec", {}).get("secretsName"),
            "pause": cluster.get("spec", {}).get("pxc", {}).get("pause", False),
            "clusterSize": cluster.get("spec", {}).get("pxc", {}).get("size"),
            "loadBalancer": {
            },
            "monitoring": {},
            "dbInstance": {}
        }
    }
    if cluster.get("spec", {}).get("haproxy", {}).get("enabled", False):
        lb = cluster.get("spec", {}).get("haproxy", {})
        database_cluster["spec"]["loadBalancer"] = {
            "type": "haproxy",
            "exposeType": lb.get("serviceType"),
            "image": lb.get("image"),
            "size": lb.get("size"),
            "configuration": lb.get("configuration"),
            "annotations": lb.get("annotations") if lb.get("annotations") else None,
            "trafficPolicy": lb.get("externalTrafficPolicy"),
            "resources": lb.get("resources"),
        }
    if cluster.get("spec", {}).get("proxysql", {}).get("enabled", False):
        lb = cluster.get("spec", {}).get("proxysql", {})
        database_cluster["spec"]["loadBalancer"] = {
            "type": "proxysql",
            "exposeType": lb.get("serviceType"),
            "image": lb.get("image"),
            "size": lb.get("size"),
            "configuration": lb.get("configuration"),
            "annotations": lb.get("annotations") if lb.get("annotations") else None,
            "trafficPolicy": lb.get("externalTrafficPolicy"),
            "resources": lb.get("resources"),
        }
    if cluster.get("spec", {}).get("pmm", {}).get("enabled", False):
        mon = cluster.get("spec", {}).get("pmm", {})
        database_cluster["spec"]["monitoring"] = {
            "pmm" : {
                "image": mon.get("image"),
                "serverHost": mon.get("serverHost"),
                "serverUser": mon.get("serverUser"),
                "publicAddress": mon.get("publicAddress"),
                "login": mon.get("login"),
                "password": mon.get("password"),
            },
            "resources": mon.get("resources"),

        }
    volume_spec = cluster.get("spec", {}).get("pxc", {}).get("volumeSpec", {}).get("persistentVolumeClaim", {})
    limits = cluster.get("spec", {}).get("pxc", {}).get("resources", {}).get("limits", {})
    database_cluster["spec"]["dbInstance"] = {
        "cpu": limits.get("cpu"),
        "memory": limits.get("memory"),
        "diskSize": volume_spec.get("resources", {}).get("requests", {}).get("storage"),
    }


    return database_cluster


def convert_psmdb(cluster):
    database_cluster = {
        "apiVersion": "dbaas.percona.com/v1",
        "kind": "DatabaseCluster",
        "metadata": {
            "namespace": cluster.get("metadata", {}).get("namespace", ""),
            "name": cluster.get("metadata", {}).get("name", ""),
            #"annotations": cluster.get("metadata", {}).get("annotations", {}),
            "finalizers": cluster.get("metadata", {}).get("finalizers", []),
        },
        "spec": {
            "databaseType": "psmdb",
            "databaseImage": cluster.get("spec", {}).get("image"),
            "secretsName": cluster.get("spec", {}).get("secrets", {}).get("users"),
            "pause": cluster.get("spec", {}).get("pause", False),
            "loadBalancer": {
            },
            "monitoring": {},
            "dbInstance": {}
        }
    }
    replsets = cluster.get("spec", {}).get("replsets", [])

    if len(replsets) == 0:
        print("Cluster has no replicasets configured. Skipping")
        return

    database_cluster["spec"]["databaseConfig"] = replsets[0].get("configuration")
    database_cluster["spec"]["clusterSize"] = replsets[0].get("size")

    mongos = cluster.get("spec", {}).get("sharding", {}).get("mongos", None)
    if mongos:
        database_cluster["spec"]["loadBalancer"] = {
            "type": "mongos",
            "exposeType": mongos.get("expose", {}).get("exposeType"),
            "image": mongos.get("image"),
            "size": mongos.get("size"),
            "configuration": mongos.get("configuration"),
            "annotations": mongos.get("expose", {}).get("serviceAnnotations"),
            "loadBalancerSourceRanges": mongos.get("expose", {}).get("loadBalancerSourceRanges"),
            "trafficPolicy": mongos.get("externalTrafficPolicy"),
            "resources": mongos.get("resources"),
        }
    if cluster.get("spec", {}).get("pmm", {}).get("enabled", False):
        mon = cluster.get("spec", {}).get("pmm", {})
        database_cluster["spec"]["monitoring"] = {
            "pmm" : {
                "image": mon.get("image"),
                "serverHost": mon.get("serverHost"),
                "serverUser": mon.get("serverUser"),
                "publicAddress": mon.get("publicAddress"),
                "login": mon.get("login"),
                "password": mon.get("password"),
            },
            "resources": mon.get("resources"),

        }
    volume_spec = replsets[0].get("volumeSpec", {}).get("persistentVolumeClaim", {})
    limits = replsets[0].get("resources", {}).get("limits", {})
    database_cluster["spec"]["dbInstance"] = {
        "cpu": limits.get("cpu"),
        "memory": limits.get("memory"),
        "diskSize": volume_spec.get("resources", {}).get("requests", {}).get("storage"),
    }


    return database_cluster


if __name__ == '__main__':
    clusters = []
    for cluster in get_clusters("pxc").get("items", []):
        del cluster["status"]
        clusters.append(convert_pxc(cluster))

    for cluster in get_clusters("psmdb").get("items", []):
        del cluster["status"]
        clusters.append(convert_psmdb(cluster))

    for cluster in clusters:
        create_cluster(cluster)
