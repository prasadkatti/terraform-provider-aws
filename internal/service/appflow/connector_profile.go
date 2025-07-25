// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package appflow

import (
	"context"
	"log"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appflow"
	"github.com/aws/aws-sdk-go-v2/service/appflow/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/enum"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @SDKResource("aws_appflow_connector_profile", name="Connector Profile")
// @IdentityAttribute("name")
// @ArnFormat("connectorprofile/{name}", attribute="arn")
// @Testing(existsType="github.com/aws/aws-sdk-go-v2/service/appflow/types;types.ConnectorProfile")
// @Testing(importIgnore="connector_profile_config.0.connector_profile_credentials")
// @Testing(idAttrDuplicates="name")
func resourceConnectorProfile() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceConnectorProfileCreate,
		ReadWithoutTimeout:   resourceConnectorProfileRead,
		UpdateWithoutTimeout: resourceConnectorProfileUpdate,
		DeleteWithoutTimeout: resourceConnectorProfileDelete,

		Schema: map[string]*schema.Schema{
			names.AttrARN: {
				Type:     schema.TypeString,
				Computed: true,
			},
			"connector_label": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: validation.All(
					validation.StringMatch(regexache.MustCompile(`[0-9A-Za-z][\w!@#.-]+`), "must contain only alphanumeric, exclamation point (!), at sign (@), number sign (#), period (.), and hyphen (-) characters"),
					validation.StringLenBetween(1, 256),
				),
			},
			"connection_mode": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: enum.Validate[types.ConnectionMode](),
			},
			"connector_profile_config": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"connector_profile_credentials": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"amplitude": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"api_key": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												names.AttrSecretKey: {
													Type:      schema.TypeString,
													Required:  true,
													Sensitive: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"custom_connector": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"api_key": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"api_key": {
																Type:     schema.TypeString,
																Required: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 256),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
															"api_secret_key": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 256),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
														},
													},
												},
												"authentication_type": {
													Type:             schema.TypeString,
													Required:         true,
													ValidateDiagFunc: enum.Validate[types.AuthenticationType](),
												},
												"basic": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															names.AttrPassword: {
																Type:         schema.TypeString,
																Required:     true,
																Sensitive:    true,
																ValidateFunc: validation.StringLenBetween(0, 512),
															},
															names.AttrUsername: {
																Type:         schema.TypeString,
																Required:     true,
																ValidateFunc: validation.StringLenBetween(0, 512),
															},
														},
													},
												},
												"custom": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"credentials_map": {
																Type:      schema.TypeMap,
																Optional:  true,
																Sensitive: true,
																ValidateDiagFunc: validation.AllDiag(
																	validation.MapKeyLenBetween(1, 128),
																	validation.MapKeyMatch(regexache.MustCompile(`[\w]+`), "must contain only alphanumeric and underscore (_) characters"),
																),
																Elem: &schema.Schema{
																	Type: schema.TypeString,
																	ValidateFunc: validation.All(
																		validation.StringLenBetween(0, 2048),
																		validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																	),
																},
															},
															"custom_authentication_type": {
																Type:         schema.TypeString,
																Required:     true,
																ValidateFunc: validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
															},
														},
													},
												},
												"oauth2": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"access_token": {
																Type:      schema.TypeString,
																Optional:  true,
																Sensitive: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 4096),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
															names.AttrClientID: {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 512),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
															names.AttrClientSecret: {
																Type:      schema.TypeString,
																Optional:  true,
																Sensitive: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 512),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
															"oauth_request": {
																Type:     schema.TypeList,
																Optional: true,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"auth_code": {
																			Type:     schema.TypeString,
																			Optional: true,
																			ValidateFunc: validation.All(
																				validation.StringLenBetween(1, 4096),
																				validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																			),
																		},
																		"redirect_uri": {
																			Type:     schema.TypeString,
																			Optional: true,
																			ValidateFunc: validation.All(
																				validation.StringLenBetween(1, 512),
																				validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																			),
																		},
																	},
																},
															},
															"refresh_token": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 4096),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
														},
													},
												},
											},
										},
									},
									"datadog": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"api_key": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												"application_key": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"dynatrace": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"api_token": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"google_analytics": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"access_token": {
													Type:      schema.TypeString,
													Optional:  true,
													Sensitive: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 2048),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												names.AttrClientID: {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												names.AttrClientSecret: {
													Type:      schema.TypeString,
													Required:  true,
													Sensitive: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												"oauth_request": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"auth_code": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 2048),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
															"redirect_uri": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 512),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
														},
													},
												},
												"refresh_token": {
													Type:     schema.TypeString,
													Optional: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 1024),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"honeycode": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"access_token": {
													Type:      schema.TypeString,
													Optional:  true,
													Sensitive: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 2048),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												"oauth_request": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"auth_code": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 2048),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
															"redirect_uri": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 512),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
														},
													},
												},
												"refresh_token": {
													Type:     schema.TypeString,
													Optional: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 1024),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"infor_nexus": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"access_key_id": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												"datakey": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												"secret_access_key": {
													Type:      schema.TypeString,
													Required:  true,
													Sensitive: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												"user_id": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"marketo": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"access_token": {
													Type:      schema.TypeString,
													Optional:  true,
													Sensitive: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 2048),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												names.AttrClientID: {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												names.AttrClientSecret: {
													Type:      schema.TypeString,
													Required:  true,
													Sensitive: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												"oauth_request": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"auth_code": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 512),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
															"redirect_uri": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 512),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
														},
													},
												},
											},
										},
									},
									"redshift": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												names.AttrPassword: {
													Type:         schema.TypeString,
													Required:     true,
													Sensitive:    true,
													ValidateFunc: validation.StringLenBetween(0, 512),
												},
												names.AttrUsername: {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"salesforce": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"access_token": {
													Type:      schema.TypeString,
													Optional:  true,
													Sensitive: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 2048),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												"client_credentials_arn": {
													Type:         schema.TypeString,
													Optional:     true,
													ValidateFunc: verify.ValidARN,
												},
												"jwt_token": {
													Type:         schema.TypeString,
													Optional:     true,
													ValidateFunc: validation.StringLenBetween(1, 8000),
												},
												"oauth2_grant_type": {
													Type:             schema.TypeString,
													Optional:         true,
													ValidateDiagFunc: enum.Validate[types.OAuth2GrantType](),
												},
												"oauth_request": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"auth_code": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 2048),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
															"redirect_uri": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 512),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
														},
													},
												},
												"refresh_token": {
													Type:     schema.TypeString,
													Optional: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 1024),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"sapo_data": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"basic_auth_credentials": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															names.AttrPassword: {
																Type:         schema.TypeString,
																Required:     true,
																Sensitive:    true,
																ValidateFunc: validation.StringLenBetween(0, 512),
															},
															names.AttrUsername: {
																Type:     schema.TypeString,
																Required: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 512),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
														},
													},
												},
												"oauth_credentials": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"access_token": {
																Type:      schema.TypeString,
																Optional:  true,
																Sensitive: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 2048),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
															names.AttrClientID: {
																Type:     schema.TypeString,
																Required: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 512),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
															names.AttrClientSecret: {
																Type:     schema.TypeString,
																Required: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 512),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
															"oauth_request": {
																Type:     schema.TypeList,
																Optional: true,
																MaxItems: 1,
																Elem: &schema.Resource{
																	Schema: map[string]*schema.Schema{
																		"auth_code": {
																			Type:     schema.TypeString,
																			Optional: true,
																			ValidateFunc: validation.All(
																				validation.StringLenBetween(1, 2048),
																				validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																			),
																		},
																		"redirect_uri": {
																			Type:     schema.TypeString,
																			Optional: true,
																			ValidateFunc: validation.All(
																				validation.StringLenBetween(1, 512),
																				validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																			),
																		},
																	},
																},
															},
															"refresh_token": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 1024),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
														},
													},
												},
											},
										},
									},
									"service_now": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												names.AttrPassword: {
													Type:         schema.TypeString,
													Required:     true,
													Sensitive:    true,
													ValidateFunc: validation.StringLenBetween(0, 512),
												},
												names.AttrUsername: {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"singular": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"api_key": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"slack": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"access_token": {
													Type:      schema.TypeString,
													Optional:  true,
													Sensitive: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 2048),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												names.AttrClientID: {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												names.AttrClientSecret: {
													Type:      schema.TypeString,
													Required:  true,
													Sensitive: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												"oauth_request": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"auth_code": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 2048),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
															"redirect_uri": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 512),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
														},
													},
												},
											},
										},
									},
									"snowflake": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												names.AttrPassword: {
													Type:         schema.TypeString,
													Required:     true,
													Sensitive:    true,
													ValidateFunc: validation.StringLenBetween(0, 512),
												},
												names.AttrUsername: {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"trendmicro": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"api_secret_key": {
													Type:      schema.TypeString,
													Required:  true,
													Sensitive: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"veeva": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												names.AttrPassword: {
													Type:         schema.TypeString,
													Required:     true,
													Sensitive:    true,
													ValidateFunc: validation.StringLenBetween(0, 512),
												},
												names.AttrUsername: {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"zendesk": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"access_token": {
													Type:      schema.TypeString,
													Optional:  true,
													Sensitive: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 2048),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												names.AttrClientID: {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												names.AttrClientSecret: {
													Type:      schema.TypeString,
													Required:  true,
													Sensitive: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												"oauth_request": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"auth_code": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 2048),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
															"redirect_uri": {
																Type:     schema.TypeString,
																Optional: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 512),
																	validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																),
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						"connector_profile_properties": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"amplitude": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{},
										},
									},
									"custom_connector": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"oauth2_properties": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"oauth2_grant_type": {
																Type:             schema.TypeString,
																Required:         true,
																ValidateDiagFunc: enum.Validate[types.OAuth2GrantType](),
															},
															"token_url": {
																Type:     schema.TypeString,
																Required: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 256),
																	validation.StringMatch(regexache.MustCompile(`^(https?)://[0-9A-Za-z-+&@#/%?=~_|!:,.;]*[0-9A-Za-z-+&@#/%=~_|]`), "must provide a valid HTTPS url"),
																),
															},
															"token_url_custom_properties": {
																Type:     schema.TypeMap,
																Optional: true,
																ValidateDiagFunc: validation.AllDiag(
																	validation.MapKeyLenBetween(1, 128),
																	validation.MapKeyMatch(regexache.MustCompile(`[\w]+`), "must contain only alphanumeric and underscore (_) characters"),
																),
																Elem: &schema.Schema{
																	Type: schema.TypeString,
																	ValidateFunc: validation.All(
																		validation.StringLenBetween(0, 2048),
																		validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																	),
																},
															},
														},
													},
												},
												"profile_properties": {
													Type:     schema.TypeMap,
													Optional: true,
													ValidateDiagFunc: validation.AllDiag(
														validation.MapKeyLenBetween(1, 128),
														validation.MapKeyMatch(regexache.MustCompile(`[\w]+`), "must contain only alphanumeric and underscore (_) characters"),
													),
													Elem: &schema.Schema{
														Type: schema.TypeString,
														ValidateFunc: validation.All(
															validation.StringLenBetween(0, 2048),
															validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
														),
													},
												},
											},
										},
									},
									"datadog": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"instance_url": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"dynatrace": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"instance_url": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"google_analytics": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{},
										},
									},
									"honeycode": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{},
										},
									},
									"infor_nexus": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"instance_url": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"marketo": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"instance_url": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"redshift": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												names.AttrBucketName: {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(3, 63),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												names.AttrBucketPrefix: {
													Type:         schema.TypeString,
													Optional:     true,
													ValidateFunc: validation.StringLenBetween(0, 512),
												},
												names.AttrClusterIdentifier: {
													Type:     schema.TypeString,
													Optional: true,
												},
												"data_api_role_arn": {
													Type:         schema.TypeString,
													Optional:     true,
													ValidateFunc: verify.ValidARN,
												},
												names.AttrDatabaseName: {
													Type:     schema.TypeString,
													Optional: true,
												},
												"database_url": {
													Type:         schema.TypeString,
													Optional:     true,
													ValidateFunc: validation.StringLenBetween(0, 512),
												},
												names.AttrRoleARN: {
													Type:         schema.TypeString,
													Required:     true,
													ValidateFunc: verify.ValidARN,
												},
											},
										},
									},
									"salesforce": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"instance_url": {
													Type:     schema.TypeString,
													Optional: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												"is_sandbox_environment": {
													Type:     schema.TypeBool,
													Optional: true,
												},
												"use_privatelink_for_metadata_and_authorization": {
													Type:     schema.TypeBool,
													Optional: true,
												},
											},
										},
									},
									"sapo_data": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"application_host_url": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`^(https?)://[0-9A-Za-z-+&@#/%?=~_|!:,.;]*[0-9A-Za-z-+&@#/%=~_|]`), "must provide a valid HTTPS url"),
													),
												},
												"application_service_path": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												"client_number": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(3, 3),
														validation.StringMatch(regexache.MustCompile(`^\d{3}$`), "must consist of exactly three digits"),
													),
												},
												"logon_language": {
													Type:     schema.TypeString,
													Optional: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(0, 2),
														validation.StringMatch(regexache.MustCompile(`^[0-9A-Za-z_]*$`), "must contain only alphanumeric characters and the underscore (_) character"),
													),
												},
												"oauth_properties": {
													Type:     schema.TypeList,
													Optional: true,
													MaxItems: 1,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"auth_code_url": {
																Type:     schema.TypeString,
																Required: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 256),
																	validation.StringMatch(regexache.MustCompile(`^(https?)://[0-9A-Za-z-+&@#/%?=~_|!:,.;]*[0-9A-Za-z-+&@#/%=~_|]`), "must provide a valid HTTPS url"),
																),
															},
															"oauth_scopes": {
																Type:     schema.TypeList,
																Required: true,
																Elem: &schema.Schema{
																	Type: schema.TypeString,
																	ValidateFunc: validation.All(
																		validation.StringLenBetween(1, 128),
																		validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
																	),
																},
															},
															"token_url": {
																Type:     schema.TypeString,
																Required: true,
																ValidateFunc: validation.All(
																	validation.StringLenBetween(1, 256),
																	validation.StringMatch(regexache.MustCompile(`^(https?)://[0-9A-Za-z-+&@#/%?=~_|!:,.;]*[0-9A-Za-z-+&@#/%=~_|]`), "must provide a valid HTTPS url"),
																),
															},
														},
													},
												},
												"port_number": {
													Type:         schema.TypeInt,
													Required:     true,
													ValidateFunc: validation.IntBetween(1, 65535),
												},
												"private_link_service_name": {
													Type:     schema.TypeString,
													Optional: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`^$|com.amazonaws.vpce.[\w/!:@#.\-]+`), "must be a valid AWS VPC endpoint address"),
													),
												},
											},
										},
									},
									"service_now": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"instance_url": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"singular": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{},
										},
									},
									"slack": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"instance_url": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"snowflake": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"account_name": {
													Type:     schema.TypeString,
													Optional: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												names.AttrBucketName: {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(3, 63),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												names.AttrBucketPrefix: {
													Type:         schema.TypeString,
													Optional:     true,
													ValidateFunc: validation.StringLenBetween(0, 512),
												},
												"private_link_service_name": {
													Type:     schema.TypeString,
													Optional: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`^$|com.amazonaws.vpce.[\w/!:@#.\-]+`), "must be a valid AWS VPC endpoint address"),
													),
												},
												names.AttrRegion: {
													Type:     schema.TypeString,
													Optional: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 64),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												names.AttrStage: {
													Type:     schema.TypeString,
													Required: true,
													DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
														return old == new || old == "@"+new
													},
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 512),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
												"warehouse": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(0, 512),
														validation.StringMatch(regexache.MustCompile(`[\s\w/!@#+=.-]*`), "must match [\\s\\w/!@#+=.-]*"),
													),
												},
											},
										},
									},
									"trendmicro": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{},
										},
									},
									"veeva": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"instance_url": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
									"zendesk": {
										Type:     schema.TypeList,
										Optional: true,
										MaxItems: 1,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"instance_url": {
													Type:     schema.TypeString,
													Required: true,
													ValidateFunc: validation.All(
														validation.StringLenBetween(1, 256),
														validation.StringMatch(regexache.MustCompile(`\S+`), "must not contain any whitespace characters"),
													),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"connector_type": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: enum.Validate[types.ConnectorType](),
			},
			"credentials_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"kms_arn": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidARN,
			},
			names.AttrName: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 256),
					validation.StringMatch(regexache.MustCompile(`[\w/!@#+=.-]+`), "must match [\\w/!@#+=.-]+"),
				),
			},
		},
	}
}

func resourceConnectorProfileCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).AppFlowClient(ctx)

	name := d.Get(names.AttrName).(string)
	input := &appflow.CreateConnectorProfileInput{
		ConnectionMode:       types.ConnectionMode(d.Get("connection_mode").(string)),
		ConnectorProfileName: aws.String(name),
		ConnectorType:        types.ConnectorType(d.Get("connector_type").(string)),
	}

	if v, ok := d.Get("connector_label").(string); ok && len(v) > 0 {
		input.ConnectorLabel = aws.String(v)
	}

	if v, ok := d.GetOk("connector_profile_config"); ok && len(v.([]any)) > 0 && v.([]any)[0] != nil {
		input.ConnectorProfileConfig = expandConnectorProfileConfig(v.([]any)[0].(map[string]any))
	}

	if v, ok := d.Get("kms_arn").(string); ok && len(v) > 0 {
		input.KmsArn = aws.String(v)
	}

	_, err := conn.CreateConnectorProfile(ctx, input)

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "creating AppFlow Connector Profile (%s): %s", name, err)
	}

	d.SetId(name)

	return append(diags, resourceConnectorProfileRead(ctx, d, meta)...)
}

func resourceConnectorProfileRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).AppFlowClient(ctx)

	connectorProfile, err := findConnectorProfileByName(ctx, conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] AppFlow Connector Profile (%s) not found, removing from state", d.Id())
		d.SetId("")
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading AppFlow Connector Profile (%s): %s", d.Id(), err)
	}

	// Credentials are not returned by any API operation. Instead, a
	// "credentials_arn" property is returned.
	//
	// It may be possible to implement a function that reads from this
	// credentials resource -- but it is not documented in the API reference.
	// (https://docs.aws.amazon.com/appflow/1.0/APIReference/API_ConnectorProfile.html#appflow-Type-ConnectorProfile-credentialsArn)
	credentials := d.Get("connector_profile_config.0.connector_profile_credentials").([]any)
	d.Set(names.AttrARN, connectorProfile.ConnectorProfileArn)
	d.Set("connection_mode", connectorProfile.ConnectionMode)
	d.Set("connector_label", connectorProfile.ConnectorLabel)
	d.Set("connector_profile_config", flattenConnectorProfileConfig(connectorProfile.ConnectorProfileProperties, credentials))
	d.Set("connector_type", connectorProfile.ConnectorType)
	d.Set("credentials_arn", connectorProfile.CredentialsArn)
	d.Set(names.AttrName, connectorProfile.ConnectorProfileName)

	return diags
}

func resourceConnectorProfileUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).AppFlowClient(ctx)

	name := d.Get(names.AttrName).(string)
	input := &appflow.UpdateConnectorProfileInput{
		ConnectionMode:       types.ConnectionMode(d.Get("connection_mode").(string)),
		ConnectorProfileName: aws.String(name),
	}

	if v, ok := d.GetOk("connector_profile_config"); ok && len(v.([]any)) > 0 && v.([]any)[0] != nil {
		input.ConnectorProfileConfig = expandConnectorProfileConfig(v.([]any)[0].(map[string]any))
	}

	_, err := conn.UpdateConnectorProfile(ctx, input)

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "updating AppFlow Connector Profile (%s): %s", d.Id(), err)
	}

	return append(diags, resourceConnectorProfileRead(ctx, d, meta)...)
}

func resourceConnectorProfileDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var diags diag.Diagnostics

	conn := meta.(*conns.AWSClient).AppFlowClient(ctx)

	log.Printf("[INFO] Deleting AppFlow Connector Profile: %s", d.Id())
	input := appflow.DeleteConnectorProfileInput{
		ConnectorProfileName: aws.String(d.Get(names.AttrName).(string)),
	}
	_, err := conn.DeleteConnectorProfile(ctx, &input)

	if errs.IsA[*types.ResourceNotFoundException](err) {
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "deleting AppFlow Connector Profile (%s): %s", d.Id(), err)
	}

	return diags
}

func findConnectorProfileByName(ctx context.Context, conn *appflow.Client, name string) (*types.ConnectorProfile, error) {
	input := appflow.DescribeConnectorProfilesInput{
		ConnectorProfileNames: []string{name},
	}

	output, err := conn.DescribeConnectorProfiles(ctx, &input)
	if err != nil {
		return nil, err
	}

	if output == nil || len(output.ConnectorProfileDetails) == 0 {
		return nil, tfresource.NewEmptyResultError(input)
	}

	return tfresource.AssertSingleValueResult(output.ConnectorProfileDetails)
}

func expandConnectorProfileConfig(m map[string]any) *types.ConnectorProfileConfig {
	cpc := &types.ConnectorProfileConfig{}

	if v, ok := m["connector_profile_credentials"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.ConnectorProfileCredentials = expandConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["connector_profile_properties"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.ConnectorProfileProperties = expandConnectorProfileProperties(v[0].(map[string]any))
	}

	return cpc
}

func expandConnectorProfileCredentials(m map[string]any) *types.ConnectorProfileCredentials {
	cpc := &types.ConnectorProfileCredentials{}

	if v, ok := m["amplitude"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Amplitude = expandAmplitudeConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["custom_connector"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.CustomConnector = expandCustomConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["datadog"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Datadog = expandDatadogConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["dynatrace"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Dynatrace = expandDynatraceConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["google_analytics"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.GoogleAnalytics = expandGoogleAnalyticsConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["honeycode"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Honeycode = expandHoneycodeConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["infor_nexus"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.InforNexus = expandInforNexusConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["marketo"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Marketo = expandMarketoConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["redshift"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Redshift = expandRedshiftConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["salesforce"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Salesforce = expandSalesforceConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["sapo_data"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.SAPOData = expandSAPODataConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["service_now"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.ServiceNow = expandServiceNowConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["singular"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Singular = expandSingularConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["slack"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Slack = expandSlackConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["snowflake"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Snowflake = expandSnowflakeConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["trendmicro"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Trendmicro = expandTrendmicroConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["veeva"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Veeva = expandVeevaConnectorProfileCredentials(v[0].(map[string]any))
	}
	if v, ok := m["zendesk"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Zendesk = expandZendeskConnectorProfileCredentials(v[0].(map[string]any))
	}

	return cpc
}

func expandAmplitudeConnectorProfileCredentials(m map[string]any) *types.AmplitudeConnectorProfileCredentials {
	credentials := &types.AmplitudeConnectorProfileCredentials{
		ApiKey:    aws.String(m["api_key"].(string)),
		SecretKey: aws.String(m[names.AttrSecretKey].(string)),
	}

	return credentials
}

func expandCustomConnectorProfileCredentials(m map[string]any) *types.CustomConnectorProfileCredentials {
	credentials := &types.CustomConnectorProfileCredentials{
		AuthenticationType: types.AuthenticationType(m["authentication_type"].(string)),
	}

	if v, ok := m["api_key"].([]any); ok && len(v) > 0 && v[0] != nil {
		credentials.ApiKey = expandAPIKeyCredentials(v[0].(map[string]any))
	}
	if v, ok := m["basic"].([]any); ok && len(v) > 0 && v[0] != nil {
		credentials.Basic = expandBasicAuthCredentials(v[0].(map[string]any))
	}
	if v, ok := m["custom"].([]any); ok && len(v) > 0 && v[0] != nil {
		credentials.Custom = expandCustomAuthCredentials(v[0].(map[string]any))
	}
	if v, ok := m["oauth2"].([]any); ok && len(v) > 0 && v[0] != nil {
		credentials.Oauth2 = expandOAuth2Credentials(v[0].(map[string]any))
	}

	return credentials
}

func expandDatadogConnectorProfileCredentials(m map[string]any) *types.DatadogConnectorProfileCredentials {
	credentials := &types.DatadogConnectorProfileCredentials{
		ApiKey:         aws.String(m["api_key"].(string)),
		ApplicationKey: aws.String(m["application_key"].(string)),
	}

	return credentials
}

func expandDynatraceConnectorProfileCredentials(m map[string]any) *types.DynatraceConnectorProfileCredentials {
	credentials := &types.DynatraceConnectorProfileCredentials{
		ApiToken: aws.String(m["api_token"].(string)),
	}

	return credentials
}

func expandGoogleAnalyticsConnectorProfileCredentials(m map[string]any) *types.GoogleAnalyticsConnectorProfileCredentials {
	credentials := &types.GoogleAnalyticsConnectorProfileCredentials{
		ClientId:     aws.String(m[names.AttrClientID].(string)),
		ClientSecret: aws.String(m[names.AttrClientSecret].(string)),
	}

	if v, ok := m["access_token"].(string); ok && v != "" {
		credentials.AccessToken = aws.String(v)
	}
	if v, ok := m["oauth_request"].([]any); ok && len(v) > 0 && v[0] != nil {
		credentials.OAuthRequest = expandOAuthRequest(v[0].(map[string]any))
	}
	if v, ok := m["refresh_token"].(string); ok && v != "" {
		credentials.RefreshToken = aws.String(v)
	}

	return credentials
}

func expandHoneycodeConnectorProfileCredentials(m map[string]any) *types.HoneycodeConnectorProfileCredentials {
	credentials := &types.HoneycodeConnectorProfileCredentials{}

	if v, ok := m["access_token"].(string); ok && v != "" {
		credentials.AccessToken = aws.String(v)
	}
	if v, ok := m["oauth_request"].([]any); ok && len(v) > 0 && v[0] != nil {
		credentials.OAuthRequest = expandOAuthRequest(v[0].(map[string]any))
	}
	if v, ok := m["refresh_token"].(string); ok && v != "" {
		credentials.RefreshToken = aws.String(v)
	}

	return credentials
}

func expandInforNexusConnectorProfileCredentials(m map[string]any) *types.InforNexusConnectorProfileCredentials {
	credentials := &types.InforNexusConnectorProfileCredentials{
		AccessKeyId:     aws.String(m["access_key_id"].(string)),
		Datakey:         aws.String(m["datakey"].(string)),
		SecretAccessKey: aws.String(m["secret_access_key"].(string)),
		UserId:          aws.String(m["user_id"].(string)),
	}

	return credentials
}

func expandMarketoConnectorProfileCredentials(m map[string]any) *types.MarketoConnectorProfileCredentials {
	credentials := &types.MarketoConnectorProfileCredentials{
		ClientId:     aws.String(m[names.AttrClientID].(string)),
		ClientSecret: aws.String(m[names.AttrClientSecret].(string)),
	}

	if v, ok := m["access_token"].(string); ok && v != "" {
		credentials.AccessToken = aws.String(v)
	}
	if v, ok := m["oauth_request"].([]any); ok && len(v) > 0 && v[0] != nil {
		credentials.OAuthRequest = expandOAuthRequest(v[0].(map[string]any))
	}

	return credentials
}

func expandRedshiftConnectorProfileCredentials(m map[string]any) *types.RedshiftConnectorProfileCredentials {
	credentials := &types.RedshiftConnectorProfileCredentials{
		Password: aws.String(m[names.AttrPassword].(string)),
		Username: aws.String(m[names.AttrUsername].(string)),
	}

	return credentials
}

func expandSalesforceConnectorProfileCredentials(m map[string]any) *types.SalesforceConnectorProfileCredentials {
	credentials := &types.SalesforceConnectorProfileCredentials{}

	if v, ok := m["access_token"].(string); ok && v != "" {
		credentials.AccessToken = aws.String(v)
	}
	if v, ok := m["client_credentials_arn"].(string); ok && v != "" {
		credentials.ClientCredentialsArn = aws.String(v)
	}
	if v, ok := m["jwt_token"].(string); ok && v != "" {
		credentials.JwtToken = aws.String(v)
	}
	if v, ok := m["oauth2_grant_type"].(string); ok && v != "" {
		credentials.OAuth2GrantType = types.OAuth2GrantType(v)
	}
	if v, ok := m["oauth_request"].([]any); ok && len(v) > 0 && v[0] != nil {
		credentials.OAuthRequest = expandOAuthRequest(v[0].(map[string]any))
	}
	if v, ok := m["refresh_token"].(string); ok && v != "" {
		credentials.RefreshToken = aws.String(v)
	}

	return credentials
}

func expandSAPODataConnectorProfileCredentials(m map[string]any) *types.SAPODataConnectorProfileCredentials {
	credentials := &types.SAPODataConnectorProfileCredentials{}

	if v, ok := m["basic_auth_credentials"].([]any); ok && len(v) > 0 && v[0] != nil {
		credentials.BasicAuthCredentials = expandBasicAuthCredentials(v[0].(map[string]any))
	}
	if v, ok := m["oauth_credentials"].([]any); ok && len(v) > 0 && v[0] != nil {
		credentials.OAuthCredentials = expandOAuthCredentials(v[0].(map[string]any))
	}

	return credentials
}

func expandServiceNowConnectorProfileCredentials(m map[string]any) *types.ServiceNowConnectorProfileCredentials {
	credentials := &types.ServiceNowConnectorProfileCredentials{
		Password: aws.String(m[names.AttrPassword].(string)),
		Username: aws.String(m[names.AttrUsername].(string)),
	}

	return credentials
}

func expandSingularConnectorProfileCredentials(m map[string]any) *types.SingularConnectorProfileCredentials {
	credentials := &types.SingularConnectorProfileCredentials{
		ApiKey: aws.String(m["api_key"].(string)),
	}

	return credentials
}

func expandSlackConnectorProfileCredentials(m map[string]any) *types.SlackConnectorProfileCredentials {
	credentials := &types.SlackConnectorProfileCredentials{
		AccessToken:  aws.String(m["access_token"].(string)),
		ClientId:     aws.String(m[names.AttrClientID].(string)),
		ClientSecret: aws.String(m[names.AttrClientSecret].(string)),
	}

	if v, ok := m["oauth_request"].([]any); ok && len(v) > 0 && v[0] != nil {
		credentials.OAuthRequest = expandOAuthRequest(v[0].(map[string]any))
	}

	return credentials
}

func expandSnowflakeConnectorProfileCredentials(m map[string]any) *types.SnowflakeConnectorProfileCredentials {
	credentials := &types.SnowflakeConnectorProfileCredentials{
		Password: aws.String(m[names.AttrPassword].(string)),
		Username: aws.String(m[names.AttrUsername].(string)),
	}

	return credentials
}

func expandTrendmicroConnectorProfileCredentials(m map[string]any) *types.TrendmicroConnectorProfileCredentials {
	credentials := &types.TrendmicroConnectorProfileCredentials{
		ApiSecretKey: aws.String(m["api_secret_key"].(string)),
	}

	return credentials
}

func expandVeevaConnectorProfileCredentials(m map[string]any) *types.VeevaConnectorProfileCredentials {
	credentials := &types.VeevaConnectorProfileCredentials{
		Password: aws.String(m[names.AttrPassword].(string)),
		Username: aws.String(m[names.AttrUsername].(string)),
	}

	return credentials
}

func expandZendeskConnectorProfileCredentials(m map[string]any) *types.ZendeskConnectorProfileCredentials {
	credentials := &types.ZendeskConnectorProfileCredentials{
		AccessToken:  aws.String(m["access_token"].(string)),
		ClientId:     aws.String(m[names.AttrClientID].(string)),
		ClientSecret: aws.String(m[names.AttrClientSecret].(string)),
	}

	if v, ok := m["oauth_request"].([]any); ok && len(v) > 0 && v[0] != nil {
		credentials.OAuthRequest = expandOAuthRequest(v[0].(map[string]any))
	}

	return credentials
}

func expandOAuthRequest(m map[string]any) *types.ConnectorOAuthRequest {
	r := &types.ConnectorOAuthRequest{}

	if v, ok := m["auth_code"].(string); ok && v != "" {
		r.AuthCode = aws.String(v)
	}

	if v, ok := m["redirect_uri"].(string); ok && v != "" {
		r.RedirectUri = aws.String(v)
	}

	return r
}

func expandAPIKeyCredentials(m map[string]any) *types.ApiKeyCredentials {
	credentials := &types.ApiKeyCredentials{}

	if v, ok := m["api_key"].(string); ok && v != "" {
		credentials.ApiKey = aws.String(v)
	}

	if v, ok := m["api_secret_key"].(string); ok && v != "" {
		credentials.ApiSecretKey = aws.String(v)
	}

	return credentials
}

func expandBasicAuthCredentials(m map[string]any) *types.BasicAuthCredentials {
	credentials := &types.BasicAuthCredentials{}

	if v, ok := m[names.AttrPassword].(string); ok && v != "" {
		credentials.Password = aws.String(v)
	}

	if v, ok := m[names.AttrUsername].(string); ok && v != "" {
		credentials.Username = aws.String(v)
	}

	return credentials
}

func expandCustomAuthCredentials(m map[string]any) *types.CustomAuthCredentials {
	credentials := &types.CustomAuthCredentials{}

	if v, ok := m["credentials_map"].(map[string]any); ok && len(v) > 0 {
		credentials.CredentialsMap = flex.ExpandStringValueMap(v)
	}

	if v, ok := m["custom_authentication_type"].(string); ok && v != "" {
		credentials.CustomAuthenticationType = aws.String(v)
	}

	return credentials
}

func expandOAuthCredentials(m map[string]any) *types.OAuthCredentials {
	credentials := &types.OAuthCredentials{
		ClientId:     aws.String(m[names.AttrClientID].(string)),
		ClientSecret: aws.String(m[names.AttrClientSecret].(string)),
	}

	if v, ok := m["access_token"].(string); ok && v != "" {
		credentials.AccessToken = aws.String(v)
	}
	if v, ok := m["oauth_request"].([]any); ok && len(v) > 0 && v[0] != nil {
		credentials.OAuthRequest = expandOAuthRequest(v[0].(map[string]any))
	}
	if v, ok := m["refresh_token"].(string); ok && v != "" {
		credentials.RefreshToken = aws.String(v)
	}

	return credentials
}

func expandOAuth2Credentials(m map[string]any) *types.OAuth2Credentials {
	credentials := &types.OAuth2Credentials{}

	if v, ok := m["access_token"].(string); ok && v != "" {
		credentials.AccessToken = aws.String(v)
	}
	if v, ok := m[names.AttrClientID].(string); ok && v != "" {
		credentials.ClientId = aws.String(v)
	}
	if v, ok := m[names.AttrClientSecret].(string); ok && v != "" {
		credentials.ClientSecret = aws.String(v)
	}
	if v, ok := m["oauth_request"].([]any); ok && len(v) > 0 && v[0] != nil {
		credentials.OAuthRequest = expandOAuthRequest(v[0].(map[string]any))
	}
	if v, ok := m["refresh_token"].(string); ok && v != "" {
		credentials.RefreshToken = aws.String(v)
	}

	return credentials
}

func expandConnectorProfileProperties(m map[string]any) *types.ConnectorProfileProperties {
	cpc := &types.ConnectorProfileProperties{}

	if v, ok := m["amplitude"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Amplitude = v[0].(*types.AmplitudeConnectorProfileProperties)
	}
	if v, ok := m["custom_connector"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.CustomConnector = expandCustomConnectorProfileProperties(v[0].(map[string]any))
	}
	if v, ok := m["datadog"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Datadog = expandDatadogConnectorProfileProperties(v[0].(map[string]any))
	}
	if v, ok := m["dynatrace"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Dynatrace = expandDynatraceConnectorProfileProperties(v[0].(map[string]any))
	}
	if v, ok := m["google_analytics"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.GoogleAnalytics = v[0].(*types.GoogleAnalyticsConnectorProfileProperties)
	}
	if v, ok := m["honeycode"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Honeycode = v[0].(*types.HoneycodeConnectorProfileProperties)
	}
	if v, ok := m["infor_nexus"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.InforNexus = expandInforNexusConnectorProfileProperties(v[0].(map[string]any))
	}
	if v, ok := m["marketo"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Marketo = expandMarketoConnectorProfileProperties(v[0].(map[string]any))
	}
	if v, ok := m["redshift"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Redshift = expandRedshiftConnectorProfileProperties(v[0].(map[string]any))
	}
	if v, ok := m["salesforce"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Salesforce = expandSalesforceConnectorProfileProperties(v[0].(map[string]any))
	}
	if v, ok := m["sapo_data"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.SAPOData = expandSAPODataConnectorProfileProperties(v[0].(map[string]any))
	}
	if v, ok := m["service_now"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.ServiceNow = expandServiceNowConnectorProfileProperties(v[0].(map[string]any))
	}
	if v, ok := m["singular"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Singular = v[0].(*types.SingularConnectorProfileProperties)
	}
	if v, ok := m["slack"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Slack = expandSlackConnectorProfileProperties(v[0].(map[string]any))
	}
	if v, ok := m["snowflake"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Snowflake = expandSnowflakeConnectorProfileProperties(v[0].(map[string]any))
	}
	if v, ok := m["trendmicro"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Trendmicro = v[0].(*types.TrendmicroConnectorProfileProperties)
	}
	if v, ok := m["veeva"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Veeva = expandVeevaConnectorProfileProperties(v[0].(map[string]any))
	}
	if v, ok := m["zendesk"].([]any); ok && len(v) > 0 && v[0] != nil {
		cpc.Zendesk = expandZendeskConnectorProfileProperties(v[0].(map[string]any))
	}

	return cpc
}

func expandDatadogConnectorProfileProperties(m map[string]any) *types.DatadogConnectorProfileProperties {
	properties := &types.DatadogConnectorProfileProperties{
		InstanceUrl: aws.String(m["instance_url"].(string)),
	}

	return properties
}

func expandDynatraceConnectorProfileProperties(m map[string]any) *types.DynatraceConnectorProfileProperties {
	properties := &types.DynatraceConnectorProfileProperties{
		InstanceUrl: aws.String(m["instance_url"].(string)),
	}

	return properties
}

func expandInforNexusConnectorProfileProperties(m map[string]any) *types.InforNexusConnectorProfileProperties {
	properties := &types.InforNexusConnectorProfileProperties{
		InstanceUrl: aws.String(m["instance_url"].(string)),
	}

	return properties
}

func expandMarketoConnectorProfileProperties(m map[string]any) *types.MarketoConnectorProfileProperties {
	properties := &types.MarketoConnectorProfileProperties{
		InstanceUrl: aws.String(m["instance_url"].(string)),
	}

	return properties
}

func expandRedshiftConnectorProfileProperties(m map[string]any) *types.RedshiftConnectorProfileProperties {
	properties := &types.RedshiftConnectorProfileProperties{
		BucketName:        aws.String(m[names.AttrBucketName].(string)),
		ClusterIdentifier: aws.String(m[names.AttrClusterIdentifier].(string)),
		RoleArn:           aws.String(m[names.AttrRoleARN].(string)),
		DataApiRoleArn:    aws.String(m["data_api_role_arn"].(string)),
		DatabaseName:      aws.String(m[names.AttrDatabaseName].(string)),
	}

	if v, ok := m[names.AttrBucketPrefix].(string); ok && v != "" {
		properties.BucketPrefix = aws.String(v)
	}

	if v, ok := m["database_url"].(string); ok && v != "" {
		properties.DatabaseUrl = aws.String(v)
	}

	return properties
}

func expandServiceNowConnectorProfileProperties(m map[string]any) *types.ServiceNowConnectorProfileProperties {
	properties := &types.ServiceNowConnectorProfileProperties{
		InstanceUrl: aws.String(m["instance_url"].(string)),
	}

	return properties
}

func expandSalesforceConnectorProfileProperties(m map[string]any) *types.SalesforceConnectorProfileProperties {
	properties := &types.SalesforceConnectorProfileProperties{}

	if v, ok := m["instance_url"].(string); ok && v != "" {
		properties.InstanceUrl = aws.String(v)
	}

	if v, ok := m["is_sandbox_environment"].(bool); ok {
		properties.IsSandboxEnvironment = v
	}

	if v, ok := m["use_privatelink_for_metadata_and_authorization"].(bool); ok {
		properties.UsePrivateLinkForMetadataAndAuthorization = v
	}

	return properties
}

func expandCustomConnectorProfileProperties(m map[string]any) *types.CustomConnectorProfileProperties {
	properties := &types.CustomConnectorProfileProperties{}

	if v, ok := m["oauth2_properties"].([]any); ok && len(v) > 0 && v[0] != nil {
		properties.OAuth2Properties = expandOAuth2Properties(v[0].(map[string]any))
	}
	if v, ok := m["profile_properties"].(map[string]any); ok && len(v) > 0 {
		properties.ProfileProperties = flex.ExpandStringValueMap(v)
	}

	return properties
}

func expandSAPODataConnectorProfileProperties(m map[string]any) *types.SAPODataConnectorProfileProperties {
	properties := &types.SAPODataConnectorProfileProperties{
		ApplicationHostUrl:     aws.String(m["application_host_url"].(string)),
		ApplicationServicePath: aws.String(m["application_service_path"].(string)),
		ClientNumber:           aws.String(m["client_number"].(string)),
		PortNumber:             aws.Int32(int32(m["port_number"].(int))),
	}

	if v, ok := m["logon_language"].(string); ok && v != "" {
		properties.LogonLanguage = aws.String(v)
	}
	if v, ok := m["oauth_properties"].([]any); ok && len(v) > 0 && v[0] != nil {
		properties.OAuthProperties = expandOAuthProperties(v[0].(map[string]any))
	}
	if v, ok := m["private_link_service_name"].(string); ok && v != "" {
		properties.PrivateLinkServiceName = aws.String(v)
	}

	return properties
}

func expandSlackConnectorProfileProperties(m map[string]any) *types.SlackConnectorProfileProperties {
	properties := &types.SlackConnectorProfileProperties{
		InstanceUrl: aws.String(m["instance_url"].(string)),
	}

	return properties
}

func expandSnowflakeConnectorProfileProperties(m map[string]any) *types.SnowflakeConnectorProfileProperties {
	properties := &types.SnowflakeConnectorProfileProperties{
		BucketName: aws.String(m[names.AttrBucketName].(string)),
		Stage:      aws.String(m[names.AttrStage].(string)),
		Warehouse:  aws.String(m["warehouse"].(string)),
	}

	if v, ok := m["account_name"].(string); ok && v != "" {
		properties.AccountName = aws.String(v)
	}

	if v, ok := m[names.AttrBucketPrefix].(string); ok && v != "" {
		properties.BucketPrefix = aws.String(v)
	}

	if v, ok := m["private_link_service_name"].(string); ok && v != "" {
		properties.PrivateLinkServiceName = aws.String(v)
	}

	if v, ok := m[names.AttrRegion].(string); ok && v != "" {
		properties.Region = aws.String(v)
	}

	return properties
}

func expandVeevaConnectorProfileProperties(m map[string]any) *types.VeevaConnectorProfileProperties {
	properties := &types.VeevaConnectorProfileProperties{
		InstanceUrl: aws.String(m["instance_url"].(string)),
	}

	return properties
}

func expandZendeskConnectorProfileProperties(m map[string]any) *types.ZendeskConnectorProfileProperties {
	properties := &types.ZendeskConnectorProfileProperties{
		InstanceUrl: aws.String(m["instance_url"].(string)),
	}

	return properties
}

func expandOAuthProperties(m map[string]any) *types.OAuthProperties {
	properties := &types.OAuthProperties{
		AuthCodeUrl: aws.String(m["auth_code_url"].(string)),
		OAuthScopes: flex.ExpandStringValueList(m["oauth_scopes"].([]any)),
		TokenUrl:    aws.String(m["token_url"].(string)),
	}

	return properties
}

func expandOAuth2Properties(m map[string]any) *types.OAuth2Properties {
	properties := &types.OAuth2Properties{
		OAuth2GrantType: types.OAuth2GrantType(m["oauth2_grant_type"].(string)),
		TokenUrl:        aws.String(m["token_url"].(string)),
	}

	if v, ok := m["token_url_custom_properties"].(map[string]any); ok && len(v) > 0 {
		properties.TokenUrlCustomProperties = flex.ExpandStringValueMap(v)
	}

	return properties
}

func flattenConnectorProfileConfig(cpp *types.ConnectorProfileProperties, cpc []any) []any {
	m := make(map[string]any)

	m["connector_profile_properties"] = flattenConnectorProfileProperties(cpp)
	m["connector_profile_credentials"] = cpc

	return []any{m}
}

func flattenConnectorProfileProperties(cpp *types.ConnectorProfileProperties) []any {
	result := make(map[string]any)
	m := make(map[string]any)

	if cpp.Amplitude != nil {
		result["amplitude"] = []any{m}
	}
	if cpp.CustomConnector != nil {
		result["custom_connector"] = flattenCustomConnectorProfileProperties(cpp.CustomConnector)
	}
	if cpp.Datadog != nil {
		m["instance_url"] = aws.ToString(cpp.Datadog.InstanceUrl)
		result["datadog"] = []any{m}
	}
	if cpp.Dynatrace != nil {
		m["instance_url"] = aws.ToString(cpp.Dynatrace.InstanceUrl)
		result["dynatrace"] = []any{m}
	}
	if cpp.GoogleAnalytics != nil {
		result["google_analytics"] = []any{m}
	}
	if cpp.Honeycode != nil {
		result["honeycode"] = []any{m}
	}
	if cpp.InforNexus != nil {
		m["instance_url"] = aws.ToString(cpp.InforNexus.InstanceUrl)
		result["infor_nexus"] = []any{m}
	}
	if cpp.Marketo != nil {
		m["instance_url"] = aws.ToString(cpp.Marketo.InstanceUrl)
		result["marketo"] = []any{m}
	}
	if cpp.Redshift != nil {
		result["redshift"] = flattenRedshiftConnectorProfileProperties(cpp.Redshift)
	}
	if cpp.SAPOData != nil {
		result["sapo_data"] = flattenSAPODataConnectorProfileProperties(cpp.SAPOData)
	}
	if cpp.Salesforce != nil {
		result["salesforce"] = flattenSalesforceConnectorProfileProperties(cpp.Salesforce)
	}
	if cpp.ServiceNow != nil {
		m["instance_url"] = aws.ToString(cpp.ServiceNow.InstanceUrl)
		result["service_now"] = []any{m}
	}
	if cpp.Singular != nil {
		result["singular"] = []any{m}
	}
	if cpp.Slack != nil {
		m["instance_url"] = aws.ToString(cpp.Slack.InstanceUrl)
		result["slack"] = []any{m}
	}
	if cpp.Snowflake != nil {
		result["snowflake"] = flattenSnowflakeConnectorProfileProperties(cpp.Snowflake)
	}
	if cpp.Trendmicro != nil {
		result["trendmicro"] = []any{m}
	}
	if cpp.Veeva != nil {
		m["instance_url"] = aws.ToString(cpp.Veeva.InstanceUrl)
		result["veeva"] = []any{m}
	}
	if cpp.Zendesk != nil {
		m["instance_url"] = aws.ToString(cpp.Zendesk.InstanceUrl)
		result["zendesk"] = []any{m}
	}

	return []any{result}
}

func flattenRedshiftConnectorProfileProperties(properties *types.RedshiftConnectorProfileProperties) []any {
	m := make(map[string]any)

	m[names.AttrBucketName] = aws.ToString(properties.BucketName)

	if properties.BucketPrefix != nil {
		m[names.AttrBucketPrefix] = aws.ToString(properties.BucketPrefix)
	}

	if properties.DatabaseUrl != nil {
		m["database_url"] = aws.ToString(properties.DatabaseUrl)
	}

	m[names.AttrRoleARN] = aws.ToString(properties.RoleArn)
	m[names.AttrClusterIdentifier] = aws.ToString(properties.ClusterIdentifier)
	m["data_api_role_arn"] = aws.ToString(properties.DataApiRoleArn)
	m[names.AttrDatabaseName] = aws.ToString(properties.DatabaseName)

	return []any{m}
}

func flattenCustomConnectorProfileProperties(properties *types.CustomConnectorProfileProperties) []any {
	m := make(map[string]any)

	if properties.OAuth2Properties != nil {
		m["oauth2_properties"] = flattenOAuth2Properties(properties.OAuth2Properties)
	}

	if properties.ProfileProperties != nil {
		m["profile_properties"] = properties.ProfileProperties
	}

	return []any{m}
}

func flattenSalesforceConnectorProfileProperties(properties *types.SalesforceConnectorProfileProperties) []any {
	m := make(map[string]any)

	if properties.InstanceUrl != nil {
		m["instance_url"] = aws.ToString(properties.InstanceUrl)
	}
	m["is_sandbox_environment"] = properties.IsSandboxEnvironment
	m["use_privatelink_for_metadata_and_authorization"] = properties.UsePrivateLinkForMetadataAndAuthorization

	return []any{m}
}

func flattenSAPODataConnectorProfileProperties(properties *types.SAPODataConnectorProfileProperties) []any {
	m := make(map[string]any)

	m["application_host_url"] = aws.ToString(properties.ApplicationHostUrl)
	m["application_service_path"] = aws.ToString(properties.ApplicationServicePath)
	m["client_number"] = aws.ToString(properties.ClientNumber)
	m["port_number"] = aws.ToInt32(properties.PortNumber)

	if properties.LogonLanguage != nil {
		m["logon_language"] = aws.ToString(properties.LogonLanguage)
	}

	if properties.OAuthProperties != nil {
		m["oauth_properties"] = flattenOAuthProperties(properties.OAuthProperties)
	}

	if properties.PrivateLinkServiceName != nil {
		m["private_link_service_name"] = aws.ToString(properties.PrivateLinkServiceName)
	}

	return []any{m}
}

func flattenSnowflakeConnectorProfileProperties(properties *types.SnowflakeConnectorProfileProperties) []any {
	m := make(map[string]any)
	if properties.AccountName != nil {
		m["account_name"] = aws.ToString(properties.AccountName)
	}

	m[names.AttrBucketName] = aws.ToString(properties.BucketName)

	if properties.BucketPrefix != nil {
		m[names.AttrBucketPrefix] = aws.ToString(properties.BucketPrefix)
	}

	if properties.Region != nil {
		m[names.AttrRegion] = aws.ToString(properties.Region)
	}

	m[names.AttrStage] = aws.ToString(properties.Stage)
	m["warehouse"] = aws.ToString(properties.Warehouse)

	return []any{m}
}

func flattenOAuthProperties(properties *types.OAuthProperties) []any {
	m := make(map[string]any)

	m["auth_code_url"] = aws.ToString(properties.AuthCodeUrl)
	m["oauth_scopes"] = properties.OAuthScopes
	m["token_url"] = aws.ToString(properties.TokenUrl)

	return []any{m}
}

func flattenOAuth2Properties(properties *types.OAuth2Properties) []any {
	m := make(map[string]any)

	m["oauth2_grant_type"] = properties.OAuth2GrantType
	m["token_url"] = aws.ToString(properties.TokenUrl)
	m["token_url_custom_properties"] = properties.TokenUrlCustomProperties

	return []any{m}
}
