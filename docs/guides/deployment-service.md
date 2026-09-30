# Configuring and registering a Mac host

These instructions require a provider build containing BF-02. A released,
lab-qualified provider/service pair has not yet been established. The compatibility
adapter was checked against service source v1.0.4 and v1.1.0; live Apple Silicon
qualification remains required.

## Configuration

Use `api_config` to configure the Host API. Defaults are HTTP port `8080`, TLS port
`8443`, prefix `/api`, and modules `api,host`. TLS requires a certificate and private
key. `enabled_modules` replaces deprecated `mode`. Legacy `api`, `catalog`, and
`orchestrator` modes include `host`. Explicit module sets are authoritative;
registration requires `api` and `host`.

The provider writes `/etc/prl-devops-service/prldevops_config.yaml` with permissions
`0600`, replacing its environment map while preserving unrelated YAML sections.
Configuration updates retain the binary and database, and regenerate the launchd
definition to clear stale argument or environment overrides. Installer omissions use an
atomic configuration overlay before checking authenticated readiness.

Typed attributes and `environment_variables` must agree when setting the same key.
Environment-only values are supported. Explicit false and zero are retained;
removing a setting removes its managed entry, allowing the service default to apply.
`API_PORT` and `ROOT_PASSWORD` remain valid names. The provider does not rewrite
optional configuration fields with defaults.

| Attribute | Runtime key |
| --- | --- |
| `enable_logging` | `LOG_TO_FILE` |
| `log_path` | `LOG_FILE_PATH` |
| `enable_port_forwarding` | Inverse of `DISABLE_REVERSE_PROXY` |
| `catalog_cache_allow_cache_above_keep_free_disk_space` | `CATALOG_CACHE_ALLOW_CACHE_ABOVE_FREE_DISK_SPACE` |
| `catalog_cache_disable_stream` | `DISABLE_CATALOG_PROVIDER_STREAMING` |

Logging aliases `PRL_DEVOPS_LOG_TO_FILE` and `PRL_DEVOPS_LOG_FILE_PATH` remain
accepted. Use the corrected keys for new configurations.

## Credentials and registration

SSH access, the Host API root password, and Orchestrator API authentication are
independent. The Host password comes from `root_password`, an equivalent
`ROOT_PASSWORD` environment entry, or the provider license as a compatibility
fallback. An explicit empty password is rejected for registration. Prefer an
explicit sensitive password variable.

The provider waits up to two minutes for authenticated Host API readiness before
registration, with five-second request deadlines. Startup network failures and
server errors are retried; rejected credentials, missing routes and invalid
certificates fail promptly. No initial manual password reset is part of the workflow.

Registration uses the resolved scheme, port and prefix, then reads back the ID,
endpoint and healthy status. The Terraform runner must reach the Host and
Orchestrator APIs. The Orchestrator must independently reach the advertised Host API.
Use either an Orchestrator API key or a complete username/password pair.

Existing registrations are matched by ID or endpoint, never by description. Port,
prefix, password and nonempty description updates use PUT. Released service APIs
cannot update tags or clear a description in place; remove the registration block
and apply before adding it again with those changes. Changing Orchestrators also
requires explicit removal first. A lost POST response is reconciled using reads,
without repeating the POST.

To disable registration, omit the entire `orchestrator_registration` block. A module
can make it conditional with a dynamic block. `register_with_orchestrator` is a module
input, not a native resource attribute.

## Versions and recovery

Pin `devops_version` for reproducibility. New installations resolve `latest` or
`use_latest_beta` to a concrete release once. The installer script is pinned to
reviewed revision `797a00b5b89aed885f870fda142c22d31c4f9a9a`. The adapter accepts
module-based versions from 1.0.4 through 1.x; this range is not a claim of live
qualification for every release. An installed binary differing from an explicit pin
fails with a diagnostic; upgrade it deliberately before applying. `latest` retains
an existing compatible binary. An explicit pin and `use_latest_beta` conflict.

A failed configuration transaction attempts to restore the previous file and job.
Readiness or registration failures preserve the installed service, database and
confirmed registration identity in partial state. `keep_after_error` controls
preservation of newly installed dependencies on earlier deployment errors; later
failures always preserve the configured host.

Terraform may mark a failed create as tainted. Inspect saved state and the actual
host before accepting a replacement plan; normal destroy still removes managed
software. After verifying that the host should be retained, clear its taint before
retrying. See [registration state](deployment-registration.md) for refresh and disabled-output behavior.

The service companion patch adds environment passthrough, missing TLS and resource
fields, atomic private configuration writes, and direct CLI certificate and caching
key fixes. The provider retains its overlay and does not assume that patch is released.
