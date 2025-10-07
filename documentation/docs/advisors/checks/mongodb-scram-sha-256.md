# Advisor check: MongoDB not using the default SCRAM-SHA-256 authentication 

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
`SCRAM-SHA-256` authentication algorithm is the default one in MongoDB v4.0 and above. If you haven't explicitly configured a different method, no action is required. 

To ensure `SCRAM-SHA-256` is used:
{.power-number}

=== "Create explicit SCRAM-SHA-256 credentials"

    Use the `createScramSha256Credential` method in your application code:

    ```
    String user;     // the user name 
    String source;   // the source where the user is defined 
    char[] password; // the password as a character array 
    // …
    MongoCredential credential = MongoCredential.createScramSha256Credential(user, source, password);
    ```
=== "Specify authentication in connection string"
    Include `authMechanism=SCRAM-SHA-256` in your connection string:

    ```
    MongoClient mongoClient = MongoClients.create("mongodb://user1:pwd1@host1/?authSource=db1&authMechanism=SCRAM-SHA-256");
    ```

    The example above uses the [Java MongoDB driver](https://www.mongodb.com/docs/drivers/java/sync/current/fundamentals/auth/). Always check the latest driver-specific commands and syntax for your programming language.
    
## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.
[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
