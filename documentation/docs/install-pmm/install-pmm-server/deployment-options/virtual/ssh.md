# SSH key

When you run PMM Server as an AWS AMI instance, you can upload your public SSH key to enable SSH access for direct management and troubleshooting.

!!! caution alert alert-warning "Important"
    The **SSH key** setting is available only on AWS AMI deployments. On all other deployment methods, the **SSH key** tab does not appear in **Configuration > Settings**.

![PMM Settings SSH Key](../../../../images/PMM_Settings_SSH_Key.jpg)

## Configure SSH access

PMM Server stores a single SSH key. Applying changes on this tab overwrites the whole `authorized_keys` file, so the key you submit replaces the previously configured one, which can no longer be used to log in.

To configure SSH access:
{.power-number}

1. Go to **Configuration > Settings > SSH key**.
2. Enter your public key in the **SSH key** field.
3. Click **Apply changes**.

!!! caution alert alert-warning "Important"
    PMM rejects a malformed key before writing anything, but it accepts any valid key — even one whose private key you do not hold. Applying a key does not disconnect current sessions, so keep one open until you confirm the new key works.

## Connect via SSH

Once your public key is configured, connect using the `admin` user:

```bash
ssh -i your-private-key admin@<pmm-server-ip>
```

=== "AWS EC2 instance"
    ```bash
    ssh -i ~/keys/my-aws-key.pem admin@ec2-203-0-113-42.compute-1.amazonaws.com
    ```

=== "Default key location"
    If your private key is in the default location (`~/.ssh/id_rsa` or `~/.ssh/id_ed25519`), you can omit the `-i` flag:
    ```bash
    ssh admin@<pmm-server-ip>
    ```
