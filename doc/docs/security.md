# Security Features in Percona Monitoring and Management

You can protect PMM from unauthorized access using the following security features:

* SSL encryption secures traffic between PMM Client and PMM Server
* HTTP password protection adds authentication when accessing the PMM Server web interface
* Keep PMM Server isolated from the internet, where possible.

In this chapter

[TOC]

## Enabling SSL Encryption

You can encrypt traffic between PMM Client and PMM Server using SSL certificates.

### Valid certificates

To use a valid SSL certificate, mount the directory with the certificate files to `/srv/nginx/` when running the PMM Server container.

```
$ docker run -d -p 443:443 \
  --volumes-from pmm-data \
  --name pmm-server \
  -v /etc/pmm-certs:/srv/nginx \
  --restart always \
  percona/pmm-server:1
```

The directory (`/etc/pmm-certs` in this example) that you intend to mount must contain the following files:

* `certificate.crt`
* `certificate.key`
* `ca-certs.pem`
* `dhparam.pem`

**NOTE**: To enable SSL encryption, The container publishes port *443* instead of *80*.

Alternatively, you can use **docker cp** to copy the files to an already existing `pmm-server` container.

```
$ docker cp certificate.crt pmm-server:/srv/nginx/certificate.crt
$ docker cp certificate.key pmm-server:/srv/nginx/certificate.key
$ docker cp ca-certs.pem pmm-server:/srv/nginx/ca-certs.pem
$ docker cp dhparam.pem pmm-server:/srv/nginx/dhparam.pem
```

This example assumes that you have changed to the directory that contains the certificate files.

### Self-signed certificates

The PMM Server images (Docker, OVF, and AMI) already include self-signed certificates. To be able to use them in your Docker container, make sure to publish the container’s port *443* to the host’s port *443* when running the **docker run** command.

```
$ docker run -d \
   -p 443:443 \
   --volumes-from pmm-data \
   --name pmm-server \
   --restart always \
   percona/pmm-server:1
```

### Enabling SSL when connecting PMM Client to PMM Server

Then, you need to enable SSL when connecting a PMM Client to a PMM Server.  If you purchased the certificate from a certificate authority (CA):

```
$ pmm-admin config --server 192.168.100.1 --server-ssl
```

If you generated a self-signed certificate:

```
$ pmm-admin config --server 192.168.100.1 --server-insecure-ssl
```

## Enabling Password Protection

You can set the password for accessing the PMM Server web interface by passing the `SERVER_PASSWORD` environment variable when creating and running the PMM Server container.

To set the environment variable, use the `-e` option.

By default, the user name is `pmm`. You can change it by passing the `SERVER_USER` environment variable. Note that the following example uses an insecure port 80 which is typically used for HTTP connections.

Run the following commands as root or by using the **sudo** command.

```
$ docker run -d -p 80:80 \
  --volumes-from pmm-data \
  --name pmm-server \
  -e SERVER_USER=jsmith \
  -e SERVER_PASSWORD=SomeR4ndom-Pa$$w0rd \
  --restart always \
  percona/pmm-server:1
```

PMM Client uses the same credentials to communicate with PMM Server.  If you set the user name and password as described, specify them when connecting a PMM Client to a PMM Server:

```
$ pmm-admin config --server 192.168.100.1 --server-user jsmith --server-password pass1234
```

## Combining Security Features

You can enable both HTTP password protection and SSL encryption by combining the corresponding options.

The following example shows how you might run the PMM Server container:

```
$ docker run -d -p 443:443 \
  --volumes-from pmm-data \
  --name pmm-server \
  -e SERVER_USER=jsmith \
  -e SERVER_PASSWORD=SomeR4ndom-Pa$$w0rd \
  -v /etc/pmm-certs:/srv/nginx \
  --restart always \
  percona/pmm-server:1
```

The following example shows how you might connect to PMM Server:

```
$ pmm-admin config --server 192.168.100.1 --server-user jsmith --server-password pass1234 --server-insecure-ssl
```

To see which security features are enabled, run either **pmm-admin ping**, **pmm-admin config**, **pmm-admin info**, or **pmm-admin list** and look at the server address field. For example:

```
$ pmm-admin ping
OK, PMM server is alive.

PMM Server      | 192.168.100.1 (insecure SSL, password-protected)
Client Name     | centos7.vm
Client Address  | 192.168.200.1
```

## Enable HTTPS secure cookies in Grafana

The following assumes you are using a Docker container for PMM Server.

1. Edit `/etc/grafana/grafana.ini`
2. Enable `cookie_secure` and set the value to `true`
3. Restart Grafana: `supervisorctl restart grafana`
