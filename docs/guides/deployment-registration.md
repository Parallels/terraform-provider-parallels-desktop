# Orchestrator registration state

This behavior requires a provider build containing BF-03. Registration is enabled by
an `orchestrator_registration` block. Omit that block to disable it. There is no
native `register_with_orchestrator` resource attribute; a wrapper module may use that
boolean to conditionally generate the block.

| Situation | `is_registered_in_orchestrator` | `orchestrator_host` | `orchestrator_host_id` |
| --- | --- | --- | --- |
| No registration requested | `false` | `null` | `null` |
| Registration confirmed | `true` | Canonical Mac Host API URL | Returned host ID |
| Removal confirmed or managed ID not found | `false` | `null` | `null` |
| Lookup or removal fails | Error; retain the last confirmed state | Retained | Retained |

`orchestrator_host` is the registered Mac's API URL, not the Orchestrator server URL.
A false value when the block is absent means this resource manages no registration;
it makes no claim about registrations created externally.

Known-disabled plans also show false/null/null. New registration, changed settings,
unknown configuration and a missing remote registration leave outputs unknown until
apply confirms the result. An unchanged, confirmed registration preserves its known
outputs and produces an empty second plan.

Refresh reads the registration without creating or removing records. A definitive
404 clears the computed identity but retains the configured block so the next apply
can register again. Authentication, transport and server errors do not clear identity.
A known ID takes precedence over endpoint matching; descriptions are never identities.
Endpoint discovery is only used when no historical ID is available. Once absence
is confirmed, refresh and removal do not adopt a later external registration at
the same URL; apply explicitly reconciles the configured request.

Removing the block uses the previous Orchestrator credentials and the top-level
managed ID, falling back to the nested ID. Successful deletion or a definitive 404
produces false/null/null. A failed deletion retains the prior block and identity so
it can be retried. Destroy removes registration before removing host software.

## Module configuration and outputs

The [conditional registration example](../../examples/deployment-registration/main.tf)
uses a dynamic block to demonstrate both enabled and disabled configurations. It
requires the fixed provider build; the source-inspected service version in the
example is not a claim of completed live Mac qualification.

Keep nullable IDs nullable in module outputs:

```hcl
output "registration" {
  value = {
    registered = parallels-desktop_deploy.mac.is_registered_in_orchestrator
    host       = parallels-desktop_deploy.mac.orchestrator_host
    host_id    = parallels-desktop_deploy.mac.orchestrator_host_id
  }
}
```

This fixes the previous absent-registration empty strings. Consumers should test
`host_id != null` rather than comparing with `""`. If a downstream interface requires
a string, convert explicitly at that boundary:

```hcl
host_id = parallels-desktop_deploy.mac.orchestrator_host_id == null ? "" : parallels-desktop_deploy.mac.orchestrator_host_id
```

Do not base a resource's `for_each` or `count` on an ID that will only be known after
registration. Use the module's configured enable/disable input instead.

## Verification

Unit tests cover plan transitions, unknown blocks, authoritative IDs, disabled reads,
404 responses, timeouts, authorization/server failures and deletion failures. The
isolated CLI acceptance test uses the production resource and Terraform protocol
with simulated host commands and local HTTP servers:

```shell
BF3_TERRAFORM_ACCEPTANCE=1 go test ./internal/deploy -run '^TestTerraformRegistrationLifecycle$' -count=1 -v
```

It exercises disabled creation, empty second plans, enable/disable transitions,
endpoint changes, external removal, failed deletion recovery and destroy. It never
installs Mac software or changes a real launchd job. Live Apple Silicon deployment
qualification remains separate.
