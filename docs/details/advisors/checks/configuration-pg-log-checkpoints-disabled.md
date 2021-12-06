# PostgreSQL log_checkpoints Is Not Enabled

## Description

It is recommended to enable the logging of checkpoint information, as that provides a lot of useful information with almost no drawbacks.

Information about checkpoints in the logs is extremely useful and provides a detailed history of changes in write load on the PostgreSQL instance, as well as gives insight into the IO performance. It is a cheap way of augmenting the regular monitoring based on views, and the only downside is that the volume of logs will be increased slightly.

## Resolution

Set log_checkpoints server configuration option to the value of on. That can be done online, and the change will reflect immediately. Next checkpoint information will be present in the PostgreSQL logs.