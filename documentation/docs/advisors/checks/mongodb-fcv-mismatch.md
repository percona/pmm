# MongoDB FeatureCompatibilityVersion

## Description
This check warns if there is a mismatch between the MongoDB version and the value of the internal FCV (Feature Compatibility Version) parameter.

## Resolution

**FeatureCompatibilityVersion (FCV)** enables or disables the features that persist data incompatible with earlier versions of MongoDB.


If there is a mismatch between the MongoDB version and the FCV parameter, make sure to set the FCV according to the MongoDB version. 
See below the values associated with FCV.

|Deployments | FCV |
|--------------------------------------|---|
|For 6.0 deployments upgraded from 5.0 | "5.0" until you setFeatureCompatibilityVersion to "6.0". |
|For 5.0 deployments upgraded from 4.4 | "4.4" until you setFeatureCompatibilityVersion to "5.0". |
|For 4.4 deployments upgraded from 4.2 | "4.2" until you setFeatureCompatibilityVersion to "4.4". |
|For 4.2 deployments upgraded from 4.0 | "4.0" until you setFeatureCompatibilityVersion to "4.2". |


To view the **featureCompatibilityVersion** for a mongod instance, run the **getParameter** command on a mongod instance:

```
db.adminCommand( {
                    getParameter: 1,                    
                    featureCompatibilityVersion:     
                  }
               )
               
```
To set the Feature Compatibility Version, run the command against the admin database. 
For example, to set the FCV on MongoDB version 6.0, run the following command -
```
db.adminCommand( { setFeatureCompatibilityVersion: "6.0" } )
```

For more information on FCV, see the [MongoDB Documentation](https://www.mongodb.com/docs/manual/reference/command/setFeatureCompatibilityVersion/).




## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
