# Adding External Services

## Adding general external services

You can collect metrics from an external (custom) exporter on a node when:


* there is already a PMM Agent instance running and,


* this node has been configured using the `pmm-admin config` command.

### Usage

```
pmm-admin add external [--service-name=<service-name>] [--listen-port=<listen-port>] [--metrics-path=<metrics-path>] [--scheme=<scheme>]
```
