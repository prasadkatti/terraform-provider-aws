// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package s3

import (
	"context"
	"fmt"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	awstypes "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/hashicorp/aws-sdk-go-base/v2/tfawserr"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	fwflex "github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

var (
	// e.g. example--usw2-az2--x-s3
	directoryBucketNameRegex = regexache.MustCompile(`^(?:[0-9a-z.-]+)--(?:[0-9a-za-z]+(?:-[0-9a-za-z]+)+)--x-s3$`)
)

func isDirectoryBucket(bucket string) bool {
	return bucketNameTypeFor(bucket) == bucketNameTypeDirectoryBucket
}

// @FrameworkResource("aws_s3_directory_bucket", name="Directory Bucket")
func newDirectoryBucketResource(context.Context) (resource.ResourceWithConfigure, error) {
	r := &directoryBucketResource{}

	return r, nil
}

type directoryBucketResource struct {
	framework.ResourceWithModel[directoryBucketResourceModel]
	framework.WithImportByID
}

func (r *directoryBucketResource) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	dataRedundancyType := fwtypes.StringEnumType[awstypes.DataRedundancy]()
	bucketTypeType := fwtypes.StringEnumType[awstypes.BucketType]()
	locationTypeType := fwtypes.StringEnumType[awstypes.LocationType]()

	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			names.AttrARN: framework.ARNAttributeComputedOnly(),
			names.AttrBucket: schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(directoryBucketNameRegex, `must be in the format [bucket_name]--[azid]--x-s3. Use the aws_s3_bucket resource to manage general purpose buckets`),
				},
			},
			"data_redundancy": schema.StringAttribute{
				CustomType: dataRedundancyType,
				Optional:   true,
				Computed:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					directoryBucketDataRedundancyPlanModifier{},
				},
			},
			names.AttrForceDestroy: schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			names.AttrID: framework.IDAttributeDeprecatedWithAlternate(path.Root(names.AttrBucket)),
			names.AttrType: schema.StringAttribute{
				CustomType: bucketTypeType,
				Optional:   true,
				Computed:   true,
				Default:    bucketTypeType.AttributeDefault(awstypes.BucketTypeDirectory),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			names.AttrLocation: schema.ListNestedBlock{
				CustomType: fwtypes.NewListNestedObjectTypeOf[locationInfoModel](ctx),
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						names.AttrName: schema.StringAttribute{
							Required: true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
						names.AttrType: schema.StringAttribute{
							CustomType: locationTypeType,
							Optional:   true,
							Computed:   true,
							Default:    locationTypeType.AttributeDefault(awstypes.LocationTypeAvailabilityZone),
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.RequiresReplace(),
							},
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
					listvalidator.IsRequired(),
				},
			},
		},
	}
}

func (r *directoryBucketResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data directoryBucketResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	locationInfoData, diags := data.Location.ToPtr(ctx)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().S3ExpressClient(ctx)

	input := &s3.CreateBucketInput{
		Bucket: fwflex.StringFromFramework(ctx, data.Bucket),
		CreateBucketConfiguration: &awstypes.CreateBucketConfiguration{
			Bucket: &awstypes.BucketInfo{
				DataRedundancy: data.DataRedundancy.ValueEnum(),
				Type:           awstypes.BucketType(data.Type.ValueString()),
			},
			Location: &awstypes.LocationInfo{
				Name: fwflex.StringFromFramework(ctx, locationInfoData.Name),
				Type: locationInfoData.Type.ValueEnum(),
			},
		},
	}

	_, err := conn.CreateBucket(ctx, input)

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("creating S3 Directory Bucket (%s)", data.Bucket.ValueString()), err.Error())

		return
	}

	// Set values for unknowns.
	data.ARN = types.StringValue(r.arn(ctx, data.Bucket.ValueString()))
	data.ID = data.Bucket

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *directoryBucketResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data directoryBucketResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().S3ExpressClient(ctx)

	data.Bucket = data.ID
	output, err := findBucket(ctx, conn, data.Bucket.ValueString())

	if tfresource.NotFound(err) {
		response.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		response.State.RemoveResource(ctx)

		return
	}

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("reading S3 Directory Bucket (%s)", data.ID.ValueString()), err.Error())

		return
	}

	// Set attributes for import.
	data.ARN = types.StringValue(r.arn(ctx, data.Bucket.ValueString()))
	data.DataRedundancy = fwtypes.StringEnumValue(defaultDirectoryBucketDataRedundancy(output.BucketLocationType))
	data.Location = fwtypes.NewListNestedObjectValueOfPtrMust(ctx, &locationInfoModel{
		Name: fwflex.StringToFramework(ctx, output.BucketLocationName),
		Type: fwtypes.StringEnumValue(output.BucketLocationType),
	})
	data.Type = fwtypes.StringEnumValue(awstypes.BucketTypeDirectory)

	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *directoryBucketResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data directoryBucketResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().S3ExpressClient(ctx)

	_, err := conn.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: fwflex.StringFromFramework(ctx, data.ID),
	})

	if tfawserr.ErrCodeEquals(err, errCodeBucketNotEmpty) {
		if data.ForceDestroy.ValueBool() {
			// Empty the bucket and try again.
			_, err = emptyDirectoryBucket(ctx, conn, data.ID.ValueString())

			if err != nil {
				response.Diagnostics.AddError(fmt.Sprintf("emptying S3 Directory Bucket (%s)", data.ID.ValueString()), err.Error())

				return
			}

			_, err = conn.DeleteBucket(ctx, &s3.DeleteBucketInput{
				Bucket: fwflex.StringFromFramework(ctx, data.ID),
			})
		}
	}

	if tfawserr.ErrCodeEquals(err, errCodeNoSuchBucket) {
		return
	}

	if err != nil {
		response.Diagnostics.AddError(fmt.Sprintf("deleting S3 Directory Bucket (%s)", data.ID.ValueString()), err.Error())

		return
	}
}

// arn returns the ARN of the specified bucket.
func (r *directoryBucketResource) arn(ctx context.Context, bucket string) string {
	return r.Meta().RegionalARN(ctx, "s3express", fmt.Sprintf("bucket/%s", bucket))
}

type directoryBucketResourceModel struct {
	framework.WithRegionModel
	ARN            types.String                                       `tfsdk:"arn"`
	Bucket         types.String                                       `tfsdk:"bucket"`
	DataRedundancy fwtypes.StringEnum[awstypes.DataRedundancy]        `tfsdk:"data_redundancy"`
	ForceDestroy   types.Bool                                         `tfsdk:"force_destroy"`
	Location       fwtypes.ListNestedObjectValueOf[locationInfoModel] `tfsdk:"location"`
	ID             types.String                                       `tfsdk:"id"`
	Type           fwtypes.StringEnum[awstypes.BucketType]            `tfsdk:"type"`
}

type locationInfoModel struct {
	Name types.String                              `tfsdk:"name"`
	Type fwtypes.StringEnum[awstypes.LocationType] `tfsdk:"type"`
}

func defaultDirectoryBucketDataRedundancy(locationType awstypes.LocationType) awstypes.DataRedundancy {
	switch locationType {
	case awstypes.LocationTypeLocalZone:
		return awstypes.DataRedundancySingleLocalZone
	default:
		return awstypes.DataRedundancySingleAvailabilityZone
	}
}

type directoryBucketDataRedundancyPlanModifier struct{}

func (d directoryBucketDataRedundancyPlanModifier) Description(ctx context.Context) string {
	return "Sets default value for data_redundancy based on location type value"
}

func (d directoryBucketDataRedundancyPlanModifier) MarkdownDescription(ctx context.Context) string {
	return d.Description(ctx)
}

func (d directoryBucketDataRedundancyPlanModifier) PlanModifyString(ctx context.Context, request planmodifier.StringRequest, response *planmodifier.StringResponse) {
	// Do nothing if there is a known planned value.
	if !request.PlanValue.IsUnknown() {
		return
	}

	var data directoryBucketResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	locationInfo, diags := data.Location.ToPtr(ctx)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	// Set the default value for data_redundancy based on the location type.
	response.PlanValue = fwflex.StringValueToFramework(ctx, defaultDirectoryBucketDataRedundancy(locationInfo.Type.ValueEnum()))
}
