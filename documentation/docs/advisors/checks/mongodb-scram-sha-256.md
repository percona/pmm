# MongoDB not using the default SCRAM-SHA-256 authentication 

## Description
This advisor warns if the default `SCRAM-SHA-256` authentication method is not used in MongoDB. `SCRAM-SHA-256` is a salted challenge-response authentication mechanism (SCRAM) that uses your username and password, encrypted with the SHA-256 algorithm, to authenticate your user.

The goal of this check is to ensure security by confirming the identity of the user before connecting. 

For production systems, enable the authentication and specify the authentication method.

In MongoDB versions (4.0 +), journaling is enabled by default and the "authenticationMechanisms" parameter is set to use "SCRAM-SHA-256" by default.
MongoDB v3.0 changed the default authentication mechanism from `MONGODB-CR` to `SCRAM-SHA-1`.

To learn more, see [Authentication Mechanisms](https://docs.mongodb.com/drivers/go/current/fundamentals/auth/) in the MongoDB documentation.



## Rule
MONGODB_GETPARAMETER
```
db.adminCommand( { getParameter : 1,  "authenticationMechanisms" : 1 } )


 authMechanism = "SCRAM-SHA-256" not in parsed.get("authenicationMechanisms")
          if authMechanism:
              results.append({
                  "summary": "MongoDB is not using the default SCRAM-SHA-256 authenticationMechanisms - v4.0 and above",
                  "description": "To follow optimal security practices, see the following documentation",
                  "read_more_url": "https://docs.mongodb.com/drivers/go/current/fundamentals/auth/",
                  "severity": "warning",
              })
          return results
```


## Resolution
`SCRAM-SHA-256` authentication algorithm is the default one in MongoDB v4.0 and above. If it has not been otherwise designated then no change is required. 
Otherwise, set the default authentication method to `SCRAM-SHA-256`.

1. To explicitly create “SCRAM-SHA-256“ credentials, use the SCRAM-SHA-256 `createScramSha256Credential` method: 
```
       String user;     // the user name 
       String source;   // the source where the user is defined 
       char[] password; // the password as a character array 
       // …
       MongoCredential credential = MongoCredential.createScramSha256Credential(user, source, password);
```
2. Use a connection string that explicitly specifies the authMechanism=SCRAM-SHA-256. 

The following example is for a [Java MongoDB driver](https://www.mongodb.com/docs/drivers/java/sync/current/fundamentals/auth/)

__Using the new MongoClient API:__ 

```
MongoClient mongoClient = MongoClients.create("mongodb://user1:pwd1@host1/?authSource=db1&authMechanism=SCRAM-SHA-256");
```

__Always  *Check Latest Driver specific commands and syntax*__

## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }