# SSH Key

When you run PMM Server as a virtual machine (AMI or OVF), you can upload your public SSH key to access the server via SSH and manage or troubleshoot it directly.

![PMM Settings SSH Key](../images/PMM_Settings_SSH_Key.jpg)

## Configure SSH access

To configure SSH access:
{.power-number}

1. Enter your **public key** in the **SSH Key** field.
2. Click **Apply SSH Key**.

This adds the key to the virtual machine so you can connect to your PMM Server instance via SSH.

For more information on virtual machine deployments, see [virtual appliance](../install-pmm/install-pmm-server/deployment-options/virtual/index.md).

## Connect via SSH

Once your public key is configured, you can connect to your PMM Server using SSH:

### Using SSH with a private key

Connect to your PMM Server using the `admin` user and your private key:

```bash
ssh -i your-private-key admin@<pmm-server-ip>
```

where:

- `your-private-key` is the path to your private key file (can be `.pem`, `.key`, or no extension)
- `<pmm-server-ip>` is your PMM Server's IP address or hostname
- The username is always `admin` for PMM virtual machine deployments

### Examples

=== "AWS EC2 instance"
    ```bash
    ssh -i ~/keys/my-aws-key.pem admin@ec2-203-0-113-42.compute-1.amazonaws.com
    ```

=== "Local virtual appliance"
    ```bash
    ssh -i ~/.ssh/pmm_key admin@192.168.1.100
    ```

=== "Using default SSH key"
    If your private key is in the default location (`~/.ssh/id_rsa` or `~/.ssh/id_ed25519`), you can omit the `-i` flag:
    ```bash
    ssh admin@<pmm-server-ip>
    ```
