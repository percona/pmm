# MongoDB not using the default SHA-256 hashing function 

## Description
This check returns a warning if a modern authentication method is not used. 
The goal is to follow updated and optimal security processes. 
For Production systems - Ensure that auth and authentication methods are in place..

MongoDB enables journaling by defaultand  "authenticationMechanisms" are set to use "SCRAM-SHA-256" by default in  MongoDB  versions (4.0 +).
MongoDB v3.0changed the default auth mechanism from MONGODB-CR to SCRAM-SHA-1.
["https://docs.mongodb.com/drivers/go/current/fundamentals/auth/]("https://docs.mongodb.com/drivers/go/current/fundamentals/auth/)



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
Perform the steps mentioned below to specify the SCRAM-SHA-256algorithm

1. This is the default in MogoDB v4.0 and above. If it has not been otherwise designated then no change is required. \
SCRAM-SHA-256 is a salted challenge-response authentication mechanism (SCRAM) that uses your username and password, encrypted with the SHA-256 algorithm, to authenticate your user. 
2. To explicitly create  a “SCRAM-SHA-256 “ credential,  use the SCRAM-SHA-256 createScramSha256Credential method: 
```
       String user;     // the user name 
       String source;   // the source where the user is defined 
       char[] password; // the password as a character array 
       // …
       MongoCredential credential = MongoCredential.createScramSha256Credential(user, source, password);
```
3.  And then use a connection string that explicitly specifies the authMechanism=SCRAM-SHA-256. 

__Using the new MongoClient API:__ \ 
`MongoClient mongoClient = MongoClients.create("mongodb://user1:pwd1@host1/?authSource=db1&authMechanism=SCRAM-SHA-256");`

__Always  *Check Latest Driver specific commands and syntax* This is a Java Driver example__

https://docs.mongodb.com/drivers/go/current/fundamentals/auth/

