# MongoDB logLevel is not default

## Description

This advisor warns if the verbosity level of MongoDB logs is higher than the default value. It reports only a non-zero level that is set explicitly, in `setParameter.logLevel` or `systemLog.verbosity`; a `mongod` that does not set one is not reported.

The verbosity level is controlled by the `logLevel` parameter. Its value is an integer between 0 and 5, where `0` is the default log level (Informational) and `5` is the most verbose level. 

Increasing the verbosity of log messages is useful for debugging purposes for a short period of time.

For more information, see [db.setLogLevel() in the MongoDB documentation](https://docs.mongodb.com/manual/reference/method/db.setLogLevel/).

## Rule
MONGODB_GETCMDLINEOPTS

`db.adminCommand( { getCmdLineOpts: 1 } )`, reading `parsed.setParameter.logLevel` and `parsed.systemLog.verbosity`

## Resolution

The check reads the options `mongod` was started with, so changing the log level at runtime with `db.setLogLevel()` or `setParameter` doesn't clear the finding.

To return to the default log level:
{.power-number}

1. In the configuration file, remove `setParameter.logLevel` and `systemLog.verbosity`, or set them to `0`. Also remove `--setParameter logLevel` and `-v` options from the `mongod` command line.
2. Restart the `mongod` nodes for the change to take effect. For a cluster or replica set, use the rolling restart method.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
