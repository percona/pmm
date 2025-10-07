# Advisor check: MongoDB BindIP Check

## Description
This check returns a warning if MongoDB network binding setting is not recommended.
This is to avoid possible security breach from listening to unidentified client IP addresses. 

For more information, see the [MongoDB documentation](https://docs.mongodb.com/manual/reference/configuration-options/#mongodb-setting-net.bindIp).

# Rule
``` MONGODB_GETCMDLINEOPTS
db.adminCommand({'getCmdLineOpts':1}).parsed.net.bindIp

net = parsed.get("net", {})
            bindIP = (net.get("bindIp") == "0.0.0.0")
            bindIP = bindIP or ("bindIpAll" in net)
```

## Resolution
Adjust network binding settings:
{.power-number}

1. Adjust IP addresses in *net.bindIp* or *bindIpAll* boolean flag
2. Edit **mongod.conf** and set the following parameter:

```net:
  bindIp: <private IP or hostname of DB server>
  #bindIpAll: false //Any of these parameters might be enabled so adjust accordingly
  ```
3. Roll-restart your mongod nodes to apply the changes.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.
[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }

   
