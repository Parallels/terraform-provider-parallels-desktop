# SSH requirements and dependency installation

Remote `parallels-desktop_deploy` operations need a complete SSH connection from the Terraform runner to the Mac, including authentication and noninteractive command execution. Opening TCP port 22 alone does not verify these requirements. `ssh_connection.host_port` defaults to 22 when omitted.

Use a deployment account with a password or an unencrypted private key and the privileges needed to install the requested software. The provider uses Go's SSH client: it does not automatically read `~/.ssh/config`, use an SSH agent, jump hosts, or OpenSSH connection multiplexing. It does not request a terminal or support interactive authentication prompts.

## Privileges and downloads

Provision noninteractive sudo rights before applying. The provider checks `sudo -n true` before dependency installation and uses `sudo -n` for its direct privileged service commands. A successful preflight does not grant rights to every subsequent command: the account must also be authorized for the actual installation/service operations. A cached sudo credential may expire during installation; unattended deployments should use an administrator-managed policy appropriate to their environment.

The provider no longer appends broad passwordless rules to `/etc/sudoers`, sends the SSH password to `sudo`, or recursively changes ownership of `/usr/local/share`. Homebrew installation runs with `NONINTERACTIVE=1`. Existing package-directory permissions must already allow the deployment account to use Homebrew.

The target Mac needs DNS and HTTPS access to Homebrew's installer, package sources and downloads, and the DevOps Service installer/release assets. The SSH file-transfer helper additionally needs the SFTP subsystem when used. Host API and Orchestrator connectivity are separate from SSH connectivity.

## Connections, retries and cancellation

Each resource lifecycle operation owns one SSH transport and runs sequential sessions on it. Separate resources can still run concurrently. The connection is closed when that operation finishes, fails, or is cancelled.

Connection establishment uses a 15-second TCP dial timeout, a 30-second handshake timeout and a 90-second connection budget, with at most five attempts. Transient connection failures use exponential backoff of 1, 2, 4 and 8 seconds plus up to 250 milliseconds of jitter, within that budget. Terraform cancellation or an earlier operation deadline takes precedence. Invalid credentials, invalid private keys and host-key verification failures are not retried. This change preserves the existing SSH host-key policy; it does not add known-hosts configuration.

A broken connection can be replaced before submitting the next command. Once a command may have reached the Mac, it is never automatically replayed. An error reporting uncertain execution means the remote command may have partially completed; inspect the host before applying again. Cancelling or closing an SSH connection does not guarantee that every remote child process has stopped.

Executable discovery uses one probe, checking the remote PATH and standard locations including `/opt/homebrew/bin`, `/usr/local/bin` and the remote account's `~/bin`. Successful discoveries are cached for the operation. A failed SSH/probe operation is reported as an error, never treated as proof that software is missing. Only newly installed dependencies are recorded for dependency rollback; pre-existing dependencies are retained.

## Investigating a handshake reset

Compare a complete authenticated SSH connection from the same Terraform runner with the provider's error phase, target, attempts and elapsed time. Remember that OpenSSH may use client settings the provider does not use. Correlate the failure timestamp with the Mac's SSH/security logs and network-device logs to identify which component sent the reset.

Check effective server limits before changing them. `MaxStartups` concerns concurrent unauthenticated connections; `MaxSessions` concerns channels within a connection. Available per-source settings depend on the Mac's installed OpenSSH version. See the [OpenSSH server configuration reference](https://man.openbsd.org/sshd_config).

Compare a failing run with `terraform apply -parallelism=1` as a diagnostic, not a required permanent setting. Multiple different target Macs do not share a single target's SSH connection limit, although network security devices can apply shared limits. Redact passwords, private keys and service credentials before sharing logs.
