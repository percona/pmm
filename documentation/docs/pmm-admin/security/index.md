# About security in PMM


By Default, PMM ships with a self-signed certificate to enable usage out of the box.  While this does enable users to have encrypted connections between clients (database clients and web/API clients) and the PMM Server, it shouldn't be considered a properly secured connection.  Taking the following precautions will ensure that you are truly secure:

- [SSL encryption with trusted certificates](#ssl-encryption) to secure traffic between clients and server;

- [Grafana HTTPS secure cookies](#grafana-https-secure-cookies)
