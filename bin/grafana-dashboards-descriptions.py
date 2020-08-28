#! /usr/bin/env python3
# Write rst and md files containing PMM dashboard panel titles and descriptions

import os
import glob
import json
# Path to local git clone of https://github.com/percona/grafana-dashboards/
repo_src = '~/Documents/GitHub/grafana-dashboards/dashboards/*.json'
dashboard_files = glob.glob(repo_src)

for filename in dashboard_files:
    with open(filename, 'r') as fp:
        title = os.path.basename(filename).replace("_", " ").replace(".json", "")
        image = "PMM_" + os.path.basename(filename).replace(".json", "") + ".jpg"
        titlelc = os.path.basename(filename).replace("_","-").replace(".json", "").lower()
        with open("dashboard-" + titlelc + ".md", "w") as md, open("dashboard-" + titlelc + ".rst", "w") as rst:
            x = json.load(fp)

            md.write("# " + title + "\n\n")
            md.write("![image](../_images/" + image + "\n\n")

            rst.write("#" * len(title) + "\n")
            rst.write(title + "\n")
            rst.write("#" * len(title) + "\n\n")
            rst.write(".. image:: /_images/" + image + "\n\n")

            for p in x["panels"]:
                if (p["type"] == "graph"):
                    if ("title" in p and "description" in p):

                        md.write("## " + p["title"] + "\n\n")
                        md.write(p["description"] + "\n\n")

                        rst.write("*" * len(p["title"]) + "\n")
                        rst.write(p["title"] + "\n")
                        rst.write("*" * len(p["title"]) + "\n\n")
                        rst.write(p["description"] + "\n\n")
        md.close
    fp.close
