# Changelog

All notable changes to this project will be documented in this file.
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.6.13282186908] - 2025-02-12



## [0.6.12926508568] - 2025-01-23

- Updated the discord announcement script
- Added a script to get the beta changelog
- Updated the way we generate release notes to match the type of release

## [0.5.13] - 2025-01-22

- Please include a summary of the change and which issue is fixed. Please also include relevant motivation and context. List any dependencies that are required for this change.
- Fixed an issue with the installation of brew in both mac intel and arm variants
- Added a new provider for the beta versions

## [0.5.12] - 2025-01-21

- fixed an issue where brew would give an error when trying to install PD into an intel mac

## [0.5.11] - 2025-01-21

- Fixed an issue where if the api stopped responding the provider would crash
- Added better context timeout to the state and remote_vm providers
- Added new DevOps configuration to the system

## [0.5.10] - 2025-01-21

- Added a publish pipeline
- Fixed an issue with the generate-changelog.sh
- Added an announcement automation for discord
- Added a create release pr for the automation
- Added fixes to the new way of building PR's
- Added a better linting system
- updated the makefile
- Fixed some issues with the ensure processes where we could get nil pointers
- Added a new process to delete vms that were created but the creation process failed
- Added the enforce of the flag force_changes on the create where it will delete vms if the pre-exist
- Fixed some issues with the getVms method where the id was not being escaped
- Added a fix for the intel macs where we now add the user to the sudoers list if not present
- Added a fix where if PD failed to install we would get an error with dependencies
- Added the host_url to the output of the remote_vm
- Fixes a nil pointer in the ensure_machine_stopped
- Added a fix where we will only check for IP changes if the machine is running
- Added the ability to set a post_processor_script to always run on update
- Fixed some issues where in some cases the update would but the vm in the wrong state
- Fixed an issue where some errors would bring a nil pointer
- Added an option to wait for network before querying to the datasource vm
- Added a retry mechanism to attempt to get the internal ip on create/update
- Fixed an issue where packer would still be queried during a host update even if it is not necessary
- Fixed an issue where if it failed to create a reverse proxy host it would break and return nil
- Fixed some issues on delete/create remote images
- Added extra logs to check for an error with intel hosts
- Fix mandatory package dependencies
- Fixed an issue where if we enabled the HTTPS for a host and registered it with an orchestrator, it would fail due to the wrong port
- Added the new resource called vm_state to set an existing VM to a state
- Fixed an issue in the telemetry not being sent correctly
- Fixed #53 an issue where the specs where not being applied correctly if the machine was stop at some points
- Fixed an issue where the specs cpu_count and memory_count where not being calculated correctly
- Resolved #54 Added the ability to use the vm datasource with the orchestrator
- Resolved #58 Added the new shared block to allow port forwarding in the the following providers, clone_vm, remote_image, vagrant_box and deploy
- Updated all examples to include the new formats and the new fields
- Added an output to the following providers, clone_vm, remote_image, vagrant_box showing the internal and external ips if available
- Fixed an issue where if the api_port was changed after it was already deployed it would not register with the host
- Resolved #52  Added a new property to set the desired state for any of the VM providers with keep_running
- Moved the resources files to use the new versioning approach for future upgrades in schemas
- Added the ability to set environment variables in the post processor scripts and on destroy scripts
- Fixed an issue where if a catalog was not found the provider would crash
- Improved a error message for when there is no hardware in a orchestrator that can host the remote image
- Bumps [github.com/pkg/sftp](https://github.com/pkg/sftp) from 1.13.6 to 1.13.7.
- Added the ability to set environment variables in the configuration file to fine-tune DevOps service deployment
- Added an option to update the devops service in a deploy resource
- fixed a bug where the deploy resource would update but failed to report correctly
- Fixed an issue with the client apiresponse being empty
- Added better error messages when checking for resources
- Fixed an issue where by mistake the remoteimage was reporting it did not support  orchestrator
- Fixed an issue where remote_vm, vagrant_box and clone_vm would fail if this was pointing to an orchestrator
- Added the ability of only running post scripts if something changed on them
- Added the ability to disbale tls validation for self signed certificates
- Fixed an issue with the https format for the orchestrator


## [0.2.2] - 2023-01-01

- Fixed an issue that was preventing the deployment from linux machines
- Removed some unnecessary files

## [0.2.0] - 2023-01-01

FEATURES:

- Fixed some bugs
- Added the ability to register a deployment with a orchestrator
- Added the ability to run prlctl commands
- Added the ability to run a script on destroy it
