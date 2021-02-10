# Setting up PMM Server

PMM Server runs as a [Docker image](docker.md), a [virtual appliance](virtual-appliance.md), or on an [AWS instance](aws.md).

## Verifying

In a browser, visit the server's IP address. If you run your server as a
virtual appliance or Amazon machine image, you will need to set up
the user name, password and public key to connect to the
server via ssh. This step is not needed if you run PMM Server using
Docker.

In this example, you would need to direct your browser to
`http://192.168.100.1`. Since you have not added any monitoring services yet,
the site will show only data related to the PMM Server internal services.

## Accessing the Components of the Web Interface

* `http://192.168.100.1` to access Home Dashboard

* `http://192.168.100.1/graph/` to access Metrics Monitor

* `http://192.168.100.1/swagger/` to access [PMM API](/details/api.md).

PMM Server provides user access control. You will need user credentials to access it.

![image](../../_images/PMM_Login.jpg)

- Default user name: `admin`
- Default password: `admin`

You will be prompted at each log in to change the default password until you do so.
