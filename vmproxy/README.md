# VMProxy

VMProxy is a stateless reverse proxy which proxies requests to VictoriaMetrics and
optionally adds `extra_filters` query based on the provided configuration.

## Configuration
See `vmproxy --help` for more information.

## Extra filters
[Extra filters](https://github.com/VictoriaMetrics/VictoriaMetrics/#prometheus-querying-api-enhancements) is an extension to VictoriaMetrics which allows
filtering access to metrics based on labels.

VMProxy adds extra filters based on configuration provided in a header.  
When running `vmproxy` with `--header-name="X-Proxy-Filter"`, requests to the proxy can send
extra filters configuration in a header called `X-Proxy-Filter`.

The value of the header shall be a base64 encoded JSON array of strings.

**Example:**  
Value `WyJlbnY9UUEiLCAicmVnaW9uPUVVIl0=` decodes to `["env=QA", "region=EU"]`.  
This applies two extra filters:

- `env=QA`
- `region=EU`

Multiple filters are joined with logical OR - such as `env=QA OR region=EU`

If the header is not present, no extra filters are applied.

## Override Host header

Use `--host-header=HOSTNAME` to override the Host header in a http request being proxied. May help with ACL issues on the network balancer.
For example if ACL has a strict check for Host value.  

## Contributing notes

### Building
`make build`

### Testing
Run `make test` to run tests. 
