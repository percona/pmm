# Verifying PMM Server

In your browser, go to the server by its IP address. If you run your server as a
virtual appliance or by using an Amazon machine image, you will need to setup
the user name, password and your public key if you intend to connect to the
server by using ssh. This step is not needed if you run PMM Server using
Docker.

In the given example, you would need to direct your browser to
*http://192.168.100.1*. Since you have not added any monitoring services yet,
the site will show only data related to the PMM Server internal services.

## Accessing the Components of the Web Interface


* `http://192.168.100.1` to access Home Dashboard.


* `http://192.168.100.1/graph/` to access Metrics Monitor.


* `http://192.168.100.1/swagger/` to access PMM API.

PMM Server provides user access control, and therefore you will need
user credentials to access it:



![image](/_images/pmm-login-screen.png)

The default user name is `admin`, and the default password is `admin` also.
You will be proposed to change the default password at login if you didn’t it.

**See also**

Configuring PMM Client with pmm-admin config
