Files in this directory were extracted from the Prometheus project:

* https://github.com/prometheus/common/blob/v0.2.0/config/http_config.go
* https://github.com/prometheus/prometheus/blob/v2.7.1/config/config.go
* https://github.com/prometheus/prometheus/blob/v2.7.1/discovery/config/config.go
* https://github.com/prometheus/prometheus/blob/v2.7.1/discovery/targetgroup/targetgroup.go
* https://github.com/prometheus/prometheus/blob/v2.7.1/pkg/labels/labels.go
* https://github.com/prometheus/prometheus/blob/v2.7.1/pkg/relabel/relabel.go

We use this, as original prometheus config package have a huge amount of dependencies.
There is some problems in vendoring in this dependencies and we don't need it for now.
In the future we will probably remove this package and start to use original one, 
but at this moment we only need StaticConfig and nothing else.
