package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var (
	_ resource.Resource                = &ReplicationResource{}
	_ resource.ResourceWithImportState = &ReplicationResource{}
	_ resource.ResourceWithModifyPlan  = &ReplicationResource{}
)

// ReplicationResource manages a ZFS replication task.
type ReplicationResource struct {
	client *wsclient.Client
}

// ReplicationResourceModel describes the resource data model.
type ReplicationResourceModel struct {
	ID                      types.String   `tfsdk:"id"`
	Name                    types.String   `tfsdk:"name"`
	Direction               types.String   `tfsdk:"direction"`
	Transport               types.String   `tfsdk:"transport"`
	SourceDatasets          types.List     `tfsdk:"source_datasets"`
	TargetDataset           types.String   `tfsdk:"target_dataset"`
	Recursive               types.Bool     `tfsdk:"recursive"`
	Auto                    types.Bool     `tfsdk:"auto"`
	Enabled                 types.Bool     `tfsdk:"enabled"`
	RetentionPolicy         types.String   `tfsdk:"retention_policy"`
	LifetimeValue           types.Int64    `tfsdk:"lifetime_value"`
	LifetimeUnit            types.String   `tfsdk:"lifetime_unit"`
	SSHCredentials          types.Int64    `tfsdk:"ssh_credentials"`
	NamingSchema            types.List     `tfsdk:"naming_schema"`
	AlsoIncludeNamingSchema types.List     `tfsdk:"also_include_naming_schema"`
	Replicate               types.Bool     `tfsdk:"replicate"`
	Properties              types.Bool     `tfsdk:"properties"`
	PropertiesExclude       types.List     `tfsdk:"properties_exclude"`
	PropertiesOverride      types.Map      `tfsdk:"properties_override"`
	NameRegex               types.String   `tfsdk:"name_regex"`
	Exclude                 types.List     `tfsdk:"exclude"`
	Readonly                types.String   `tfsdk:"readonly"`
	AllowFromScratch        types.Bool     `tfsdk:"allow_from_scratch"`
	HoldPendingSnapshots    types.Bool     `tfsdk:"hold_pending_snapshots"`
	Retries                 types.Int64    `tfsdk:"retries"`
	LargeBlock              types.Bool     `tfsdk:"large_block"`
	Compressed              types.Bool     `tfsdk:"compressed"`
	Embed                   types.Bool     `tfsdk:"embed"`
	SpeedLimit              types.Int64    `tfsdk:"speed_limit"`
	LoggingLevel            types.String   `tfsdk:"logging_level"`
	Sudo                    types.Bool     `tfsdk:"sudo"`
	ScheduleMinute          types.String   `tfsdk:"schedule_minute"`
	ScheduleHour            types.String   `tfsdk:"schedule_hour"`
	ScheduleDom             types.String   `tfsdk:"schedule_dom"`
	ScheduleMonth           types.String   `tfsdk:"schedule_month"`
	ScheduleDow             types.String   `tfsdk:"schedule_dow"`
	ScheduleBegin           types.String   `tfsdk:"schedule_begin"`
	ScheduleEnd             types.String   `tfsdk:"schedule_end"`
	Timeouts                timeouts.Value `tfsdk:"timeouts"`
}

func NewReplicationResource() resource.Resource {
	return &ReplicationResource{}
}

func (r *ReplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_replication"
}

func (r *ReplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a ZFS replication task on TrueNAS SCALE. " +
		"The initial snapshot sync can be slow; raise `create` and `update` in the `timeouts` block if they time out.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the replication task.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the replication task.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"direction": schema.StringAttribute{
				Description: "Replication direction (PUSH or PULL).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("PUSH", "PULL"),
				},
			},
			"transport": schema.StringAttribute{
				Description: "Transport type (SSH, SSH+NETCAT, LOCAL).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("LOCAL"),
				Validators: []validator.String{
					stringvalidator.OneOf("SSH", "SSH+NETCAT", "LOCAL", "LEGACY"),
				},
			},
			"source_datasets": schema.ListAttribute{
				Description: "List of source dataset paths.",
				Required:    true,
				ElementType: types.StringType,
				// A replication task with no sources replicates nothing.
				// The API accepts an empty list on update, so without this
				// the task silently stops doing anything and the apply
				// still reports success.
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"target_dataset": schema.StringAttribute{
				Description: "Target dataset path.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"recursive": schema.BoolAttribute{
				Description: "Whether to recursively replicate child datasets.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"auto": schema.BoolAttribute{
				Description: "Whether to run the replication automatically on schedule.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the replication task is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"retention_policy": schema.StringAttribute{
				Description: "Snapshot retention policy (SOURCE, CUSTOM, NONE).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("SOURCE"),
				Validators: []validator.String{
					stringvalidator.OneOf("SOURCE", "CUSTOM", "NONE"),
				},
			},
			"lifetime_value": schema.Int64Attribute{
				Description: "Lifetime value for CUSTOM retention policy (1-9999).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 9999),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"lifetime_unit": schema.StringAttribute{
				Description: "Lifetime unit for CUSTOM retention (HOUR, DAY, WEEK, MONTH, YEAR).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("HOUR", "DAY", "WEEK", "MONTH", "YEAR"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ssh_credentials": schema.Int64Attribute{
				Description: "SSH credentials ID for remote replication.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"naming_schema": schema.ListAttribute{
				Description: "Naming schema for matching snapshots on the source (for pull replication).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"also_include_naming_schema": schema.ListAttribute{
				Description: "Naming schema for snapshots to include (for push replication without periodic snapshot tasks).",
				Optional:    true,
				ElementType: types.StringType,
			},

			"replicate": schema.BoolAttribute{
				Description: "Replicate the whole filesystem tree with `zfs send -R`, including child datasets, snapshots and properties. " +
					"TrueNAS only allows `retention_policy` of SOURCE together with this. Leave unset to keep the TrueNAS default (or the current value).",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"properties": schema.BoolAttribute{
				Description:   "Send dataset properties along with the snapshots.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"properties_exclude": schema.ListAttribute{
				Description:   "Dataset properties that are not sent or are discarded on receive. Requires `properties`. Clearing an existing list is not supported.",
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"properties_override": schema.MapAttribute{
				Description:   "Dataset properties to set to a fixed value on the target. Requires `properties`. Clearing an existing map is not supported.",
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Map{mapplanmodifier.UseStateForUnknown()},
			},
			"name_regex": schema.StringAttribute{
				Description: "Regular expression selecting which source snapshots are replicated, as an alternative to `naming_schema`. " +
					"TrueNAS only allows `retention_policy` of SOURCE or NONE with it.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"exclude": schema.ListAttribute{
				Description:   "Child datasets of the sources to leave out of a recursive replication. Clearing an existing list is not supported.",
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"readonly": schema.StringAttribute{
				Description: "How the target dataset's `readonly` property is handled: SET makes it read-only, REQUIRE fails if it is not already, IGNORE leaves it alone.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("SET", "REQUIRE", "IGNORE"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"allow_from_scratch": schema.BoolAttribute{
				Description:   "Destroy the target and do a full send when no common snapshot exists, instead of failing.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"hold_pending_snapshots": schema.BoolAttribute{
				Description:   "Hold snapshots that have not been replicated yet so retention cannot delete them.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"retries": schema.Int64Attribute{
				Description: "How many times a failed replication is retried.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"large_block": schema.BoolAttribute{
				Description:   "Allow blocks larger than 128 KiB in the stream (`zfs send -L`).",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"compressed": schema.BoolAttribute{
				Description:   "Send blocks in their compressed on-disk form (`zfs send -c`).",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"embed": schema.BoolAttribute{
				Description:   "Send embedded data blocks (`zfs send -e`).",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"speed_limit": schema.Int64Attribute{
				Description: "Limit the transfer to this many bytes per second.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"logging_level": schema.StringAttribute{
				Description: "Log level of the replication task (DEBUG, INFO, WARNING, ERROR). Unset uses the system default.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("DEBUG", "INFO", "WARNING", "ERROR"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sudo": schema.BoolAttribute{
				Description:   "Use `sudo` for the zfs commands on the remote side (SSH transports).",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"schedule_minute": schema.StringAttribute{
				Description:   "Cron minute field of the replication schedule. If any `schedule_*` attribute is set, unset ones default to minute `0`, hour `*`, day-of-month, month and day-of-week `*`, window `00:00` to `23:59`. Only used when `auto` is true.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"schedule_hour": schema.StringAttribute{
				Description:   "Cron hour field of the replication schedule.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"schedule_dom": schema.StringAttribute{
				Description:   "Cron day-of-month field of the replication schedule.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"schedule_month": schema.StringAttribute{
				Description:   "Cron month field of the replication schedule.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"schedule_dow": schema.StringAttribute{
				Description:   "Cron day-of-week field of the replication schedule.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"schedule_begin": schema.StringAttribute{
				Description:   "Start of the daily window (HH:MM) in which the schedule may fire.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"schedule_end": schema.StringAttribute{
				Description:   "End of the daily window (HH:MM) in which the schedule may fire.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *ReplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*wsclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *wsclient.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *ReplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Replication start")

	var plan ReplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithCreateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	var sourceDatasets []string
	resp.Diagnostics.Append(plan.SourceDatasets.ElementsAs(ctx, &sourceDatasets, false)...)
	createReq := &truenas.ReplicationCreateRequest{
		Name:            plan.Name.ValueString(),
		Direction:       plan.Direction.ValueString(),
		Transport:       plan.Transport.ValueString(),
		SourceDatasets:  sourceDatasets,
		TargetDataset:   plan.TargetDataset.ValueString(),
		Recursive:       plan.Recursive.ValueBool(),
		AutoBool:        plan.Auto.ValueBool(),
		Enabled:         plan.Enabled.ValueBool(),
		RetentionPolicy: plan.RetentionPolicy.ValueString(),
	}

	if !plan.LifetimeValue.IsNull() {
		createReq.LifetimeValue = int(plan.LifetimeValue.ValueInt64())
	}
	if !plan.LifetimeUnit.IsNull() {
		createReq.LifetimeUnit = plan.LifetimeUnit.ValueString()
	}
	if !plan.SSHCredentials.IsNull() && !plan.SSHCredentials.IsUnknown() {
		createReq.SSHCredentials = int(plan.SSHCredentials.ValueInt64())
	}
	if !plan.NamingSchema.IsNull() && !plan.NamingSchema.IsUnknown() {
		var ns []string
		resp.Diagnostics.Append(plan.NamingSchema.ElementsAs(ctx, &ns, false)...)
		createReq.NamingSchema = ns
	}
	if !plan.AlsoIncludeNamingSchema.IsNull() && !plan.AlsoIncludeNamingSchema.IsUnknown() {
		var ains []string
		resp.Diagnostics.Append(plan.AlsoIncludeNamingSchema.ElementsAs(ctx, &ains, false)...)
		createReq.AlsoIncludeNamingSchema = ains
	}

	opts := replicationOptionsFromPlan(ctx, &plan, &resp.Diagnostics)
	createReq.Replicate = opts.replicate
	createReq.Properties = opts.properties
	createReq.PropertiesExclude = opts.propertiesExclude
	createReq.PropertiesOverride = opts.propertiesOverride
	createReq.NameRegex = opts.nameRegex
	createReq.Exclude = opts.exclude
	createReq.Readonly = opts.readonly
	createReq.AllowFromScratch = opts.allowFromScratch
	createReq.HoldPendingSnapshots = opts.holdPendingSnapshots
	createReq.Retries = opts.retries
	createReq.LargeBlock = opts.largeBlock
	createReq.Compressed = opts.compressed
	createReq.Embed = opts.embed
	createReq.SpeedLimit = opts.speedLimit
	createReq.LoggingLevel = opts.loggingLevel
	createReq.Sudo = opts.sudo
	createReq.Schedule = opts.schedule
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating replication task", map[string]interface{}{
		"name": plan.Name.ValueString(),
	})

	repl, err := r.client.CreateReplication(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Replication",
			fmt.Sprintf("Could not create replication task %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, repl, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create Replication success")
}

func (r *ReplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Replication start")

	var state ReplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithReadTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse replication ID: %s", err))
		return
	}

	repl, err := r.client.GetReplication(ctx, id)
	if err != nil {
		if wsclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Replication",
			fmt.Sprintf("Could not read replication %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, repl, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read Replication success")
}

func (r *ReplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Replication start")

	var plan ReplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithUpdateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	var state ReplicationResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse replication ID: %s", err))
		return
	}

	var sourceDatasets []string
	resp.Diagnostics.Append(plan.SourceDatasets.ElementsAs(ctx, &sourceDatasets, false)...)
	recursive := plan.Recursive.ValueBool()
	auto := plan.Auto.ValueBool()
	enabled := plan.Enabled.ValueBool()

	updateReq := &truenas.ReplicationUpdateRequest{
		Name:            plan.Name.ValueString(),
		Direction:       plan.Direction.ValueString(),
		Transport:       plan.Transport.ValueString(),
		SourceDatasets:  sourceDatasets,
		TargetDataset:   plan.TargetDataset.ValueString(),
		Recursive:       &recursive,
		AutoBool:        &auto,
		Enabled:         &enabled,
		RetentionPolicy: plan.RetentionPolicy.ValueString(),
	}

	if !plan.LifetimeValue.IsNull() {
		updateReq.LifetimeValue = int(plan.LifetimeValue.ValueInt64())
	}
	if !plan.LifetimeUnit.IsNull() {
		updateReq.LifetimeUnit = plan.LifetimeUnit.ValueString()
	}
	if !plan.SSHCredentials.IsNull() {
		updateReq.SSHCredentials = int(plan.SSHCredentials.ValueInt64())
	}

	opts := replicationOptionsFromPlan(ctx, &plan, &resp.Diagnostics)
	updateReq.Replicate = opts.replicate
	updateReq.Properties = opts.properties
	updateReq.PropertiesExclude = opts.propertiesExclude
	updateReq.PropertiesOverride = opts.propertiesOverride
	updateReq.NameRegex = opts.nameRegex
	updateReq.Exclude = opts.exclude
	updateReq.Readonly = opts.readonly
	updateReq.AllowFromScratch = opts.allowFromScratch
	updateReq.HoldPendingSnapshots = opts.holdPendingSnapshots
	updateReq.Retries = opts.retries
	updateReq.LargeBlock = opts.largeBlock
	updateReq.Compressed = opts.compressed
	updateReq.Embed = opts.embed
	updateReq.SpeedLimit = opts.speedLimit
	updateReq.LoggingLevel = opts.loggingLevel
	updateReq.Sudo = opts.sudo
	updateReq.Schedule = opts.schedule
	if resp.Diagnostics.HasError() {
		return
	}

	repl, err := r.client.UpdateReplication(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Replication",
			fmt.Sprintf("Could not update replication %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, repl, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update Replication success")
}

func (r *ReplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Replication start")

	var state ReplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithDeleteTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse replication ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting replication task", map[string]interface{}{"id": id})

	err = r.client.DeleteReplication(ctx, id)
	if err != nil {
		if wsclient.IsNotFound(err) {
			tflog.Warn(ctx, "Replication task already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Replication",
			fmt.Sprintf("Could not delete replication %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete Replication success")
}

// ModifyPlan enforces replication cross-attribute constraints:
//
//   - Any transport other than LOCAL requires `ssh_credentials` to be set.
//     SSH and SSH+NETCAT both need an SSH keypair; LOCAL replication does not.
//   - `retention_policy=CUSTOM` requires `lifetime_value` and `lifetime_unit`
//     to be set, otherwise snapshots never expire.
func (r *ReplicationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_replication")
	if req.Plan.Raw.IsNull() {
		return
	}

	var config ReplicationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	transport := "LOCAL"
	if !config.Transport.IsNull() && !config.Transport.IsUnknown() {
		transport = config.Transport.ValueString()
	}
	sshSet := !config.SSHCredentials.IsNull() && !config.SSHCredentials.IsUnknown() && config.SSHCredentials.ValueInt64() > 0

	if transport != "LOCAL" && transport != "" && !sshSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("ssh_credentials"),
			"Missing ssh_credentials",
			fmt.Sprintf("transport=%s requires ssh_credentials to reference a valid "+
				"keychain credential. Only transport=LOCAL can run without SSH.", transport),
		)
	}

	retention := ""
	if !config.RetentionPolicy.IsNull() && !config.RetentionPolicy.IsUnknown() {
		retention = config.RetentionPolicy.ValueString()
	}
	lifetimeSet := !config.LifetimeValue.IsNull() && !config.LifetimeValue.IsUnknown() && config.LifetimeValue.ValueInt64() > 0
	lifetimeUnitSet := !config.LifetimeUnit.IsNull() && !config.LifetimeUnit.IsUnknown() && config.LifetimeUnit.ValueString() != ""
	if retention == "CUSTOM" && (!lifetimeSet || !lifetimeUnitSet) {
		resp.Diagnostics.AddAttributeError(
			path.Root("lifetime_value"),
			"Missing lifetime for CUSTOM retention",
			"retention_policy=CUSTOM requires both lifetime_value (>0) and lifetime_unit to be set.",
		)
	}
}

func (r *ReplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Replication ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ReplicationResource) mapResponseToModel(ctx context.Context, repl *truenas.Replication, model *ReplicationResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(repl.ID))
	model.Name = types.StringValue(repl.Name)
	model.Direction = types.StringValue(repl.Direction)
	model.Transport = types.StringValue(repl.Transport)
	model.TargetDataset = types.StringValue(repl.TargetDataset)
	model.Recursive = types.BoolValue(repl.Recursive)
	model.Auto = types.BoolValue(repl.AutoBool)
	model.Enabled = types.BoolValue(repl.Enabled)
	model.RetentionPolicy = types.StringValue(repl.RetentionPolicy)

	if repl.LifetimeValue > 0 {
		model.LifetimeValue = types.Int64Value(int64(repl.LifetimeValue))
	} else {
		model.LifetimeValue = types.Int64Null()
	}
	if repl.LifetimeUnit != "" {
		model.LifetimeUnit = types.StringValue(repl.LifetimeUnit)
	} else {
		model.LifetimeUnit = types.StringNull()
	}
	if repl.SSHCredentials > 0 {
		model.SSHCredentials = types.Int64Value(int64(repl.SSHCredentials))
	} else {
		model.SSHCredentials = types.Int64Null()
	}

	sourceValues, diags := types.ListValueFrom(ctx, types.StringType, repl.SourceDatasets)
	if !diags.HasError() {
		model.SourceDatasets = sourceValues
	}

	if len(repl.NamingSchema) > 0 {
		nsValues, d := types.ListValueFrom(ctx, types.StringType, repl.NamingSchema)
		if !d.HasError() {
			model.NamingSchema = nsValues
		}
	} else {
		model.NamingSchema = types.ListNull(types.StringType)
	}

	if len(repl.AlsoIncludeNamingSchema) > 0 {
		ainsValues, d := types.ListValueFrom(ctx, types.StringType, repl.AlsoIncludeNamingSchema)
		if !d.HasError() {
			model.AlsoIncludeNamingSchema = ainsValues
		}
	} else {
		model.AlsoIncludeNamingSchema = types.ListNull(types.StringType)
	}

	model.Replicate = types.BoolValue(repl.Replicate)
	model.Properties = types.BoolValue(repl.Properties)
	model.AllowFromScratch = types.BoolValue(repl.AllowFromScratch)
	model.HoldPendingSnapshots = types.BoolValue(repl.HoldPendingSnapshots)
	model.LargeBlock = types.BoolValue(repl.LargeBlock)
	model.Compressed = types.BoolValue(repl.Compressed)
	model.Embed = types.BoolValue(repl.Embed)
	model.Sudo = types.BoolValue(repl.Sudo)

	// These attributes are Optional+Computed, so they must always be known
	// after apply. An empty collection is used rather than null.
	propsExclude, d := types.ListValueFrom(ctx, types.StringType, nonNilStrings(repl.PropertiesExclude))
	if !d.HasError() {
		model.PropertiesExclude = propsExclude
	}
	exclude, d := types.ListValueFrom(ctx, types.StringType, nonNilStrings(repl.Exclude))
	if !d.HasError() {
		model.Exclude = exclude
	}
	override := make(map[string]string, len(repl.PropertiesOverride))
	for k, v := range repl.PropertiesOverride {
		override[k] = fmt.Sprint(v)
	}
	overrideValue, d := types.MapValueFrom(ctx, types.StringType, override)
	if !d.HasError() {
		model.PropertiesOverride = overrideValue
	}

	model.NameRegex = optionalString(repl.NameRegex)
	model.Readonly = optionalString(repl.Readonly)
	if repl.Retries > 0 {
		model.Retries = types.Int64Value(int64(repl.Retries))
	} else {
		model.Retries = types.Int64Null()
	}
	if repl.SpeedLimit != nil {
		model.SpeedLimit = types.Int64Value(int64(*repl.SpeedLimit))
	} else {
		model.SpeedLimit = types.Int64Null()
	}
	if repl.LoggingLevel != nil {
		model.LoggingLevel = optionalString(*repl.LoggingLevel)
	} else {
		model.LoggingLevel = types.StringNull()
	}

	if sched := repl.Schedule; sched != nil {
		model.ScheduleMinute = types.StringValue(sched.Minute)
		model.ScheduleHour = types.StringValue(sched.Hour)
		model.ScheduleDom = types.StringValue(sched.Dom)
		model.ScheduleMonth = types.StringValue(sched.Month)
		model.ScheduleDow = types.StringValue(sched.Dow)
		model.ScheduleBegin = optionalString(sched.Begin)
		model.ScheduleEnd = optionalString(sched.End)
	} else {
		model.ScheduleMinute = types.StringNull()
		model.ScheduleHour = types.StringNull()
		model.ScheduleDom = types.StringNull()
		model.ScheduleMonth = types.StringNull()
		model.ScheduleDow = types.StringNull()
		model.ScheduleBegin = types.StringNull()
		model.ScheduleEnd = types.StringNull()
	}
}

func optionalString(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

// replicationOptions holds the optional replication settings taken from a
// plan. A nil pointer or empty value means "not configured", so it is left
// out of the request and TrueNAS keeps its default (or the current value).
type replicationOptions struct {
	replicate            *bool
	properties           *bool
	propertiesExclude    []string
	propertiesOverride   map[string]interface{}
	nameRegex            string
	exclude              []string
	readonly             string
	allowFromScratch     *bool
	holdPendingSnapshots *bool
	retries              *int
	largeBlock           *bool
	compressed           *bool
	embed                *bool
	speedLimit           *int
	loggingLevel         string
	sudo                 *bool
	schedule             *truenas.ReplicationSchedule
}

func known(v interface {
	IsNull() bool
	IsUnknown() bool
}) bool {
	return !v.IsNull() && !v.IsUnknown()
}

func boolOption(v types.Bool) *bool {
	if !known(v) {
		return nil
	}
	b := v.ValueBool()
	return &b
}

func replicationOptionsFromPlan(ctx context.Context, plan *ReplicationResourceModel, diags *diag.Diagnostics) replicationOptions {
	opts := replicationOptions{
		replicate:            boolOption(plan.Replicate),
		properties:           boolOption(plan.Properties),
		allowFromScratch:     boolOption(plan.AllowFromScratch),
		holdPendingSnapshots: boolOption(plan.HoldPendingSnapshots),
		largeBlock:           boolOption(plan.LargeBlock),
		compressed:           boolOption(plan.Compressed),
		embed:                boolOption(plan.Embed),
		sudo:                 boolOption(plan.Sudo),
	}
	if known(plan.PropertiesExclude) {
		diags.Append(plan.PropertiesExclude.ElementsAs(ctx, &opts.propertiesExclude, false)...)
	}
	if known(plan.Exclude) {
		diags.Append(plan.Exclude.ElementsAs(ctx, &opts.exclude, false)...)
	}
	if known(plan.PropertiesOverride) {
		var override map[string]string
		diags.Append(plan.PropertiesOverride.ElementsAs(ctx, &override, false)...)
		if len(override) > 0 {
			opts.propertiesOverride = make(map[string]interface{}, len(override))
			for k, v := range override {
				opts.propertiesOverride[k] = v
			}
		}
	}
	if known(plan.NameRegex) {
		opts.nameRegex = plan.NameRegex.ValueString()
	}
	if known(plan.Readonly) {
		opts.readonly = plan.Readonly.ValueString()
	}
	if known(plan.Retries) {
		n := int(plan.Retries.ValueInt64())
		opts.retries = &n
	}
	if known(plan.SpeedLimit) {
		n := int(plan.SpeedLimit.ValueInt64())
		opts.speedLimit = &n
	}
	if known(plan.LoggingLevel) {
		opts.loggingLevel = plan.LoggingLevel.ValueString()
	}

	// Build the schedule only if at least one schedule_* attribute is set.
	sched := truenas.ReplicationSchedule{
		Minute: "0", Hour: "*", Dom: "*", Month: "*", Dow: "*",
		Begin: "00:00", End: "23:59",
	}
	any := false
	for _, f := range []struct {
		v   types.String
		dst *string
	}{
		{plan.ScheduleMinute, &sched.Minute},
		{plan.ScheduleHour, &sched.Hour},
		{plan.ScheduleDom, &sched.Dom},
		{plan.ScheduleMonth, &sched.Month},
		{plan.ScheduleDow, &sched.Dow},
		{plan.ScheduleBegin, &sched.Begin},
		{plan.ScheduleEnd, &sched.End},
	} {
		if known(f.v) {
			*f.dst = f.v.ValueString()
			any = true
		}
	}
	if any {
		opts.schedule = &sched
	}
	return opts
}
