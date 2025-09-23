# MongoDB version check 

## Description
This advisor check rolls through a list of available versions and warns if MongoDB or Percona Server for MongoDB version is not the latest one.

The goal is to follow updated and optimal upgrade plans and paths. This avoids possible bugs and security issues. 

For Production systems, we recommend upgrading to the latest patch release for major or minor stable versions. 


## Rule
```MONGODB_BUILDINFO

version = parse_version(info["version"])
          print("version =", repr(version))

if is_percona:
              latest = LATEST_VERSIONS["percona"][mm]
              if latest > num:


          if True:  # MongoDB
              latest = LATEST_VERSIONS["mongodb"][mm]
              if latest > num:ity": "warning",
              })
          return results
```

## Resolution
Upgrade to the latest patch release for major or minor stable versions as soon as possible.

For example, if you are currently running the major version 4.4.x, upgrade to the latest available patch release for this version.

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
