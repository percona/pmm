#! /usr/bin/env python3
# Write md files containing PMM dashboard panel titles and descriptions to current dir

import os, glob, json
# Path to local git clone of https://github.com/percona/grafana-dashboards/
repo_src = '../../grafana-dashboards/dashboards/'

if (not os.path.isdir(repo_src)):
    print(repo_src + " not a directory")
    exit

# Dict of dashboard files
dashboard_files = glob.glob(repo_src + '*.json')
print(dashboard_files)
# For each, open the file, read in fields
for filename in dashboard_files:
    print(filename)
    with open(filename, 'r') as fp:
        # Title and image come from filename
        title = os.path.basename(filename).replace("_", " ").replace(".json", "")
        image = "PMM_" + os.path.basename(filename).replace(".json", "") + ".jpg"
        titlelc = os.path.basename(filename).replace("_","-").replace(".json", "").lower()

        with open("dashboard-" + titlelc + ".md", "w") as md:
            x = json.load(fp)
            md.write("# " + title + "\n\n")
            md.write("![image](../images/" + image + ")\n\n")

            for p in x["panels"]:
                if (p["type"] == "row"):
                    if ("title" in p):
                        md.write("## " + p["title"] + "\n\n")

                    if ("description" in p):
                        md.write(p["description"] + "\n\n")

                    if ("panels" in p):
                        for p2 in p["panels"]:
                            if (p2["type"] in ["graph", "singlestat"]):
                                if ("title" in p2 and "description" in p2):
                                    md.write("### " + p2["title"] + "\n\n")
                                    md.write(p2["description"] + "\n\n")

        md.close
    fp.close
