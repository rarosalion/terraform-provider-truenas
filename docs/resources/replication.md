---
page_title: "truenas_replication Resource - terraform-provider-truenas"
subcategory: "Scheduling"
description: |-
  Manages a ZFS replication task on TrueNAS SCALE.
---

# truenas_replication (Resource)

Manages a ZFS replication task on TrueNAS SCALE.

Managed attributes map directly to the TrueNAS SCALE API. Changes are applied
via the JSON-RPC endpoint on the target system; mutations that cannot be
represented in-place force a resource replacement as noted below.

## Example Usage

### Basic

```terraform
resource "truenas_replication" "example" {
  name             = "tank-offsite"
  direction        = "PUSH"
  transport        = "SSH"
  source_datasets  = ["tank/data"]
  target_dataset   = "backup/tank"
  recursive        = true
  enabled          = true
  auto             = true
  retention_policy = "NONE"
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the replication task.
* `direction` - (Required) Replication direction (PUSH or PULL). Valid values: `PUSH`, `PULL`.
* `source_datasets` - (Required) List of source dataset paths.
* `target_dataset` - (Required) Target dataset path.
* `transport` - (Optional) Transport type (SSH, SSH+NETCAT, LOCAL). Valid values: `SSH`, `SSH+NETCAT`, `LOCAL`, `LEGACY`. Default: `LOCAL`.
* `recursive` - (Optional) Whether to recursively replicate child datasets. Default: `false`.
* `auto` - (Optional) Whether to run the replication automatically on schedule. Default: `true`.
* `enabled` - (Optional) Whether the replication task is enabled. Default: `true`.
* `retention_policy` - (Optional) Snapshot retention policy (SOURCE, CUSTOM, NONE). Valid values: `SOURCE`, `CUSTOM`, `NONE`. Default: `SOURCE`.
* `lifetime_value` - (Optional) Lifetime value for CUSTOM retention policy.
* `lifetime_unit` - (Optional) Lifetime unit for CUSTOM retention (HOUR, DAY, WEEK, MONTH, YEAR). Valid values: `HOUR`, `DAY`, `WEEK`, `MONTH`, `YEAR`.
* `ssh_credentials` - (Optional) SSH credentials ID for remote replication.
* `naming_schema` - (Optional) Naming schema for matching snapshots on the source (for pull replication).
* `also_include_naming_schema` - (Optional) Naming schema for snapshots to include (for push replication without periodic snapshot tasks).
* `replicate` - (Optional) Replicate the whole filesystem tree with `zfs send -R`. TrueNAS only allows `retention_policy = "SOURCE"` with it. Computed from TrueNAS when unset.
* `properties` - (Optional) Send dataset properties along with the snapshots.
* `properties_exclude` - (Optional) Dataset properties that are not sent or are discarded on receive. Requires `properties`. Clearing an existing list is not supported.
* `properties_override` - (Optional) Map of dataset properties to set on the target. Requires `properties`. Clearing an existing map is not supported.
* `name_regex` - (Optional) Regular expression selecting the source snapshots to replicate, as an alternative to `naming_schema`. TrueNAS only allows `retention_policy` of `SOURCE` or `NONE` with it.
* `exclude` - (Optional) Child datasets to leave out of a recursive replication. Clearing an existing list is not supported.
* `readonly` - (Optional) How the target's `readonly` property is handled. Valid values: `SET`, `REQUIRE`, `IGNORE`.
* `allow_from_scratch` - (Optional) Destroy the target and do a full send when no common snapshot exists.
* `hold_pending_snapshots` - (Optional) Hold snapshots that have not been replicated yet so retention cannot delete them.
* `retries` - (Optional) How many times a failed replication is retried.
* `large_block` - (Optional) Allow blocks larger than 128 KiB in the stream.
* `compressed` - (Optional) Send blocks in their compressed on-disk form.
* `embed` - (Optional) Send embedded data blocks.
* `speed_limit` - (Optional) Limit the transfer to this many bytes per second.
* `logging_level` - (Optional) Log level of the task. Valid values: `DEBUG`, `INFO`, `WARNING`, `ERROR`.
* `sudo` - (Optional) Use `sudo` for the zfs commands on the remote side.
* `schedule_minute`, `schedule_hour`, `schedule_dom`, `schedule_month`, `schedule_dow` - (Optional) Cron fields of the schedule used when `auto` is true. If any `schedule_*` attribute is set, unset ones default to minute `0` and `*` for the rest.
* `schedule_begin`, `schedule_end` - (Optional) Daily window (`HH:MM`) in which the schedule may fire. Default `00:00` to `23:59`.

All of the optional settings above are also computed: when left unset in the configuration they take the value TrueNAS reports, so importing an existing task and planning shows no changes for them.
* `timeouts` - (Optional) Configuration block for operation timeouts. See [below](#timeouts).

### Timeouts

The `timeouts` block supports:

* `create` - (Default `20m`) Timeout for creating the resource.
* `read` - (Default `5m`) Timeout for reading the resource.
* `update` - (Default `20m`) Timeout for updating the resource.
* `delete` - (Default `20m`) Timeout for deleting the resource.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The unique identifier of the `truenas_replication` resource.

## Import

The `truenas_replication` resource can be imported using its identifier:

```shell
#!/usr/bin/env bash
terraform import truenas_replication.example 1
```
