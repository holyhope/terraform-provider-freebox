package internal

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/nikolalohinski/free-go/client"
	freeboxTypes "github.com/nikolalohinski/free-go/types"
)

var (
	_ resource.Resource                = &virtualMachinePowerResource{}
	_ resource.ResourceWithImportState = &virtualMachinePowerResource{}
)

const defaultPowerTimeoutKill = "30s"

func NewVirtualMachinePowerResource() resource.Resource {
	return &virtualMachinePowerResource{}
}

type virtualMachinePowerResource struct {
	client client.Client
}

type virtualMachinePowerModel struct {
	ID         types.String `tfsdk:"id"`
	VMID       types.Int64  `tfsdk:"vm_id"`
	PowerState types.String `tfsdk:"power_state"`
	Timeouts   types.Object `tfsdk:"timeouts"`
}

type virtualMachinePowerTimeoutsModel struct {
	Kill timetypes.GoDuration `tfsdk:"kill"`
}

func (r *virtualMachinePowerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_virtual_machine_power"
}

func (r *virtualMachinePowerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the power state of a Freebox virtual machine independently from its configuration. Use this resource on `freebox_virtual_machine` to start a VM only after other resources (such as `freebox_dhcp_static_lease`) have been applied in the same Terraform run.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifier of the power management resource (same as `vm_id`)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vm_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Identifier of the virtual machine",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"power_state": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Desired power state of the virtual machine",
				Validators: []validator.String{
					stringvalidator.OneOf(
						freeboxTypes.RunningStatus,
						freeboxTypes.StoppedStatus,
					),
				},
			},
			"timeouts": schema.SingleNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Timeouts for power operations",
				Default: objectdefault.StaticValue(
					types.ObjectValueMust(
						map[string]attr.Type{"kill": timetypes.GoDurationType{}},
						map[string]attr.Value{
							"kill": timetypes.NewGoDurationValueFromStringMust(defaultPowerTimeoutKill),
						},
					),
				),
				Attributes: map[string]schema.Attribute{
					"kill": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						CustomType:          timetypes.GoDurationType{},
						Default:             stringdefault.StaticString(defaultPowerTimeoutKill),
						MarkdownDescription: "Duration to wait for a graceful shutdown before force killing the virtual machine (default: `\"" + defaultPowerTimeoutKill + "\"`)",
					},
				},
			},
		},
	}
}

func (r *virtualMachinePowerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *virtualMachinePowerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model virtualMachinePowerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if diags := r.applyPowerState(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	model.ID = basetypes.NewStringValue(strconv.FormatInt(model.VMID.ValueInt64(), 10))
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *virtualMachinePowerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model virtualMachinePowerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	virtualMachine, err := r.client.GetVirtualMachine(ctx, model.VMID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get virtual machine", err.Error())
		return
	}

	model.ID = basetypes.NewStringValue(strconv.FormatInt(model.VMID.ValueInt64(), 10))
	switch virtualMachine.Status {
	case freeboxTypes.RunningStatus, freeboxTypes.StartingStatus:
		model.PowerState = basetypes.NewStringValue(freeboxTypes.RunningStatus)
	case freeboxTypes.StoppedStatus, freeboxTypes.StoppingStatus:
		model.PowerState = basetypes.NewStringValue(freeboxTypes.StoppedStatus)
	default:
		model.PowerState = basetypes.NewStringValue(virtualMachine.Status)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *virtualMachinePowerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model virtualMachinePowerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if diags := r.applyPowerState(ctx, &model); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	model.ID = basetypes.NewStringValue(strconv.FormatInt(model.VMID.ValueInt64(), 10))
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *virtualMachinePowerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model virtualMachinePowerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if model.PowerState.ValueString() != freeboxTypes.StoppedStatus {
		model.PowerState = basetypes.NewStringValue(freeboxTypes.StoppedStatus)
		if diags := r.applyPowerState(ctx, &model); diags.HasError() {
			resp.Diagnostics.Append(diags...)
		}
	}
}

func (r *virtualMachinePowerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	vmID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected the import identifier to be an int64 but got: %s", req.ID),
		)
		return
	}

	virtualMachine, err := r.client.GetVirtualMachine(ctx, vmID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get virtual machine", err.Error())
		return
	}

	powerState := freeboxTypes.StoppedStatus
	if virtualMachine.Status == freeboxTypes.RunningStatus || virtualMachine.Status == freeboxTypes.StartingStatus {
		powerState = freeboxTypes.RunningStatus
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vm_id"), vmID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("power_state"), powerState)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("timeouts"), &virtualMachinePowerTimeoutsModel{
		Kill: timetypes.NewGoDurationValueFromStringMust(defaultPowerTimeoutKill),
	})...)
}

func (r *virtualMachinePowerResource) applyPowerState(ctx context.Context, model *virtualMachinePowerModel) diag.Diagnostics {
	var diags diag.Diagnostics

	var timeouts virtualMachinePowerTimeoutsModel
	diags.Append(model.Timeouts.As(ctx, &timeouts, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return diags
	}

	virtualMachine, err := r.client.GetVirtualMachine(ctx, model.VMID.ValueInt64())
	if err != nil {
		diags.AddError("Failed to get virtual machine", err.Error())
		return diags
	}

	desired := model.PowerState.ValueString()
	switch desired {
	case freeboxTypes.RunningStatus:
		if virtualMachine.Status == freeboxTypes.RunningStatus || virtualMachine.Status == freeboxTypes.StartingStatus {
			return diags
		}
		if err := r.client.StartVirtualMachine(ctx, model.VMID.ValueInt64()); err != nil {
			diags.AddError("Failed to start virtual machine", err.Error())
		}
	case freeboxTypes.StoppedStatus:
		if virtualMachine.Status == freeboxTypes.StoppedStatus || virtualMachine.Status == freeboxTypes.StoppingStatus {
			return diags
		}
		if err := r.client.StopVirtualMachine(ctx, model.VMID.ValueInt64()); err != nil {
			diags.AddError("Failed to stop virtual machine", err.Error())
		}
	default:
		diags.AddError("Invalid power state", fmt.Sprintf("Unsupported power state %q", desired))
	}

	return diags
}
