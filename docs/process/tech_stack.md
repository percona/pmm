# Tech stack

Currently, our development team has fewer people than components/repositories. It is important for us to use shared libraries and tools to make our life easier. It's also fine to bring in new ones if there is a reason, but that reason should be more appealing than just "let's try this new cool package" or "that's an overengineering". Also, if we decide to make a change in this list, it's better to change it in all components within a reasonable timeframe.

- Read more
  - [Best practices](./best_practices.md)
  - [Code style](./best_practices.md#code-style)

## Our technology stack

- [protobuf v3](https://developers.google.com/protocol-buffers/) gives us [strongly-typed](https://developers.google.com/protocol-buffers/docs/proto3) serialization format with good [forward- and backward-compatibility](https://developers.google.com/protocol-buffers/docs/gotutorial#extending-a-protocol-buffer), [canonical mapping to and from JSON](https://developers.google.com/protocol-buffers/docs/proto3#json), and a large ecosystem of libraries and tools. We don't have to write code to work with it because there are code generators for a lot of languages.
- [gRPC](https://grpc.io/) extends protobuf with RPC mechanism. Both single requests/responses and bi-directional streams are supported. Error handling is built-in. Again, there are code generators for both client- and server-side code, so we don't have to write it by ourselves.
- [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway) takes gRPC specification and generates code for HTTP JSON API server-side wrapper for it. It also generates [Swagger](https://swagger.io/) specification from protocol specification and annotations, with documentation built up from comments. In turn, it is used to generate client-side code for environments where gRPC is not yet supported natively (e.g. web browser). No manual writing of serialization and communication code, and documentation with examples and interactive tools – gRPC specification becomes the single source of truth.
- [logrus](https://github.com/sirupsen/logrus) or stdlib `log` package should be used for logging. Always log to unbuffered stderr, let process supervisor do the rest.
- [prometheus client](https://github.com/prometheus/client_golang) is used for exposing internal metrics of application and gRPC library.
- [testify](https://github.com/stretchr/testify) or stdlib `testing` package should be used for writing tests. Testify should be used only for `assert` and `require` packages – suites here have some problems with logging and parallel tests. Common setups and teardowns should be implemented with `testing` [subtests](https://golang.org/pkg/testing/#hdr-Subtests_and_Sub_benchmarks).
- [golangci-lint](https://github.com/golangci/golangci-lint) is used for static code checks.
- [gocoverutil](https://github.com/AlekSi/gocoverutil) gather code coverage metrics.
- [Docker Compose](https://docs.docker.com/compose/) is used for a local development environment and in CI.
- [Kong](https://github.com/alecthomas/kong) for PMM CLI and [kingpin.v2](http://gopkg.in/alecthomas/kingpin.v2) for exporters and some other code. Use [Kong](https://github.com/alecthomas/kong) if you want to contribute a brand new CLI or need to make significant changes to the old `kingpin.v2`-based CLI.
- [go modules](https://go.dev/ref/mod#introduction) for vendoring.

## Open questions

- Do we need something else for tracing?
- Do we need something for integration tests? Something like https://github.com/go-gophers/gophers?
- Configuration library? Files, flags, environment variables?
- Build system:
  - Use promu so we don't have to copy&paste Makefiles everywhere? It also has a nice cross-build functionality, it can build containers, it has license checking tools.
  - Consider [Go Releaser](https://goreleaser.com)?
