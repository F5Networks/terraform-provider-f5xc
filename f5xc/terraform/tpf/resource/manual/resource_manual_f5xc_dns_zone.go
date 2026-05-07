//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"gopkg.volterra.us/stdlib/client/vesapi"
	"gopkg.volterra.us/stdlib/server"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/custom_types/timetypes"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/utils"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/validators/atmostoneofvalidator"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/validators/bytesvalidator"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	ves_io_schema "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema"
	ves_io_schema_dns_zone "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/dns_zone"
	ves_io_schema_views "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views"
	ves_io_schema_views_origin_pool "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views/origin_pool"
)

// ResourceF5XCDnsZone defines the resource implementation.
type ResourceF5XCDnsZone struct {
	client *client.APIClient
}

type ObjectRefTypeModel struct {
	Name      types.String `tfsdk:"name"`
	Tenant    types.String `tfsdk:"tenant"`
	Namespace types.String `tfsdk:"namespace"`
}

type GlobalSpecTypeDnsType struct {
	Secondary *GlobalSpecTypeSecondary `tfsdk:"secondary"`
	Primary   *GlobalSpecTypePrimary   `tfsdk:"primary"`
}

type NsRecordModel struct {
	Name   types.String `tfsdk:"name"`
	Values types.List   `tfsdk:"values"`
}

type DefaultRrSetGroupModel struct {
	Ttl         types.Int64  `tfsdk:"ttl"`
	NsRecord    types.Object `tfsdk:"ns_record"`
	Description types.String `tfsdk:"description"`
}

type SecretType struct {
	SecretInfoChoice *SecretTypeSecretInfoOneof `tfsdk:"secret_info_choice"`
}

type SecretTypeSecretInfoOneof struct {
	BlindfoldSecretInfo *SecretTypeBlindfoldSecretInfo `tfsdk:"blindfold_secret_info"`
	ClearSecretInfo     *SecretTypeClearSecretInfo     `tfsdk:"clear_secret_info"`
}

type SecretTypeBlindfoldSecretInfo struct {
	StoreProvider      types.String `tfsdk:"store_provider"`
	Location           types.String `tfsdk:"location"`
	DecryptionProvider types.String `tfsdk:"decryption_provider"`
}

type SecretTypeClearSecretInfo struct {
	Provider types.String `tfsdk:"provider"`
	Url      types.String `tfsdk:"url"`
}

type GlobalSpecTypeSecondary struct {
	TsigKeyName       types.String      `tfsdk:"tsig_key_name"`
	TsigKeyAlgorithm  types.List        `tfsdk:"tsig_key_algorithm"`
	TsigKeyValue      *SecretType       `tfsdk:"tsig_key_value"`
	LastAxfrTimestamp timetypes.RFC3339 `tfsdk:"last_axfr_timestamp"`
	PrimaryServers    types.List        `tfsdk:"primary_servers"`
}

type GlobalSpecTypePrimary struct {
	Admin                     types.String                               `tfsdk:"admin"`
	DefaultRrSetGroup         types.List                                 `tfsdk:"default_rr_set_group"`
	DnssecMode                *DnsZoneDNSSECMode                         `tfsdk:"dnssec_mode"`
	AllowHttpLbManagedRecords types.Bool                                 `tfsdk:"allow_http_lb_managed_records"`
	SoaRecordParametersChoice *PrimaryDNSConfigSoaRecordParametersChoice `tfsdk:"soa_record_parameters_choice"`
	RrSetGroup                []DnsZoneRRSetGroup                        `tfsdk:"rr_set_group"`
	Serial                    types.Int32                                `tfsdk:"serial"`
}

type DnsZoneRRSet struct {
	Ttl                 types.Int32         `tfsdk:"ttl"`
	TypeRecordSetChoice *RRSetTypeRecordSet `tfsdk:"type_record_set_choice"`
	Description         types.String        `tfsdk:"description"`
}

type RRSetTypeRecordSet struct {
	MxRecord    *RRSetMxRecord    `tfsdk:"mx_record"`
	NaptrRecord *RRSetNaptrRecord `tfsdk:"naptr_record"`
	SrvRecord   *RRSetSrvRecord   `tfsdk:"srv_record"`
	PtrRecord   *RRSetPtrRecord   `tfsdk:"ptr_record"`
	ARecord     *RRSetARecord     `tfsdk:"a_record"`
	AaaaRecord  *RRSetAaaaRecord  `tfsdk:"aaaa_record"`
	Eui48Record *RRSetEui48Record `tfsdk:"eui48_record"`
	AliasRecord *RRSetAliasRecord `tfsdk:"alias_record"`
	AfsdbRecord *RRSetAfsdbRecord `tfsdk:"afsdb_record"`
	DsRecord    *RRSetDsRecord    `tfsdk:"ds_record"`
	CnameRecord *RRSetCnameRecord `tfsdk:"cname_record"`
	TlsaRecord  *RRSetTlsaRecord  `tfsdk:"tlsa_record"`
	Eui64Record *RRSetEui64Record `tfsdk:"eui64_record"`
	CaaRecord   *RRSetCaaRecord   `tfsdk:"caa_record"`
	SshfpRecord *RRSetSshfpRecord `tfsdk:"sshfp_record"`
	CertRecord  *RRSetCertRecord  `tfsdk:"cert_record"`
	TxtRecord   *RRSetTxtRecord   `tfsdk:"txt_record"`
	CdsRecord   *RRSetCdsRecord   `tfsdk:"cds_record"`
	NsRecord    *RRSetNsRecord    `tfsdk:"ns_record"`
	LocRecord   *RRSetLocRecord   `tfsdk:"loc_record"`
	LbRecord    *RRSetLbRecord    `tfsdk:"lb_record"`
}

type RRSetMxRecord struct {
	Values []DnsZoneMailExchanger `tfsdk:"values"`
	Name   types.String           `tfsdk:"name"`
}

type DnsZoneMailExchanger struct {
	Domain   types.String `tfsdk:"domain"`
	Priority types.Int32  `tfsdk:"priority"`
}

type RRSetNaptrRecord struct {
	Name   types.String        `tfsdk:"name"`
	Values []DnsZoneNAPTRValue `tfsdk:"values"`
}

type DnsZoneNAPTRValue struct {
	Replacement types.String `tfsdk:"replacement"`
	Order       types.Int32  `tfsdk:"order"`
	Preference  types.Int32  `tfsdk:"preference"`
	Flags       types.String `tfsdk:"flags"`
	Service     types.String `tfsdk:"service"`
	Regexp      types.String `tfsdk:"regexp"`
}

type RRSetSrvRecord struct {
	Name   types.String        `tfsdk:"name"`
	Values []DnsZoneSRVService `tfsdk:"values"`
}

type DnsZoneSRVService struct {
	Weight   types.Int32  `tfsdk:"weight"`
	Port     types.Int32  `tfsdk:"port"`
	Target   types.String `tfsdk:"target"`
	Priority types.Int32  `tfsdk:"priority"`
}

type RRSetPtrRecord struct {
	Name   types.String `tfsdk:"name"`
	Values types.List   `tfsdk:"values"`
}

type RRSetARecord struct {
	Name   types.String `tfsdk:"name"`
	Values types.List   `tfsdk:"values"`
}

type RRSetAaaaRecord struct {
	Name   types.String `tfsdk:"name"`
	Values types.List   `tfsdk:"values"`
}

type RRSetEui48Record struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type RRSetAliasRecord struct {
	Value types.String `tfsdk:"value"`
}

type RRSetAfsdbRecord struct {
	Name   types.String              `tfsdk:"name"`
	Values []DnsZoneAFSDBRecordValue `tfsdk:"values"`
}

type DnsZoneAFSDBRecordValue struct {
	Hostname types.String `tfsdk:"hostname"`
	Subtype  types.List   `tfsdk:"subtype"`
}

type RRSetDsRecord struct {
	Name   types.String           `tfsdk:"name"`
	Values []DnsZoneDSRecordValue `tfsdk:"values"`
}

type DnsZoneDSRecordValue struct {
	KeyTag         types.Int32                `tfsdk:"key_tag"`
	DsKeyAlgorithm types.List                 `tfsdk:"ds_key_algorithm"`
	DigestChoice   *DSRecordValueDigestChoice `tfsdk:"digest_choice"`
}

type DSRecordValueDigestChoice struct {
	Sha256Digest *DSRecordValueSha256Digest `tfsdk:"sha256_digest"`
	Sha384Digest *DSRecordValueSha384Digest `tfsdk:"sha384_digest"`
	Sha1Digest   *DSRecordValueSha1Digest   `tfsdk:"sha1_digest"`
}

type DSRecordValueSha256Digest struct {
	Digest types.String `tfsdk:"digest"`
}

type DSRecordValueSha384Digest struct {
	Digest types.String `tfsdk:"digest"`
}

type DSRecordValueSha1Digest struct {
	Digest types.String `tfsdk:"digest"`
}

type RRSetCnameRecord struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type RRSetTlsaRecord struct {
	Name   types.String             `tfsdk:"name"`
	Values []DnsZoneTLSARecordValue `tfsdk:"values"`
}

type DnsZoneTLSARecordValue struct {
	CertificateUsage           types.List   `tfsdk:"certificate_usage"`
	Selector                   types.List   `tfsdk:"selector"`
	MatchingType               types.List   `tfsdk:"matching_type"`
	CertificateAssociationData types.String `tfsdk:"certificate_association_data"`
}

type RRSetEui64Record struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type RRSetCaaRecord struct {
	Name   types.String                                 `tfsdk:"name"`
	Values []DnsZoneCertificationAuthorityAuthorization `tfsdk:"values"`
}

type DnsZoneCertificationAuthorityAuthorization struct {
	Value types.String `tfsdk:"value"`
	Flags types.Int32  `tfsdk:"flags"`
	Tag   types.String `tfsdk:"tag"`
}

type RRSetSshfpRecord struct {
	Name   types.String              `tfsdk:"name"`
	Values []DnsZoneSSHFPRecordValue `tfsdk:"values"`
}

type DnsZoneSSHFPRecordValue struct {
	Algorithm             types.List                       `tfsdk:"algorithm"`
	FingerprintTypeChoice *SSHFPRecordValueFingerprintType `tfsdk:"fingerprint_type_choice"`
}

type SSHFPRecordValueFingerprintType struct {
	Sha1Fingerprint   *SSHFPRecordValueSha1Fingerprint   `tfsdk:"sha1_fingerprint"`
	Sha256Fingerprint *SSHFPRecordValueSha256Fingerprint `tfsdk:"sha256_fingerprint"`
}

type SSHFPRecordValueSha1Fingerprint struct {
	Fingerprint types.String `tfsdk:"fingerprint"`
}

type SSHFPRecordValueSha256Fingerprint struct {
	Fingerprint types.String `tfsdk:"fingerprint"`
}

type RRSetCertRecord struct {
	Name   types.String             `tfsdk:"name"`
	Values []DnsZoneCERTRecordValue `tfsdk:"values"`
}

type DnsZoneCERTRecordValue struct {
	CertType    types.List   `tfsdk:"cert_type"`
	CertKeyTag  types.Int32  `tfsdk:"cert_key_tag"`
	Algorithm   types.List   `tfsdk:"algorithm"`
	Certificate types.String `tfsdk:"certificate"`
}

type RRSetTxtRecord struct {
	Name   types.String `tfsdk:"name"`
	Values types.List   `tfsdk:"values"`
}

type RRSetCdsRecord struct {
	Name   types.String           `tfsdk:"name"`
	Values []DnsZoneDSRecordValue `tfsdk:"values"`
}

type RRSetNsRecord struct {
	Name   types.String `tfsdk:"name"`
	Values types.List   `tfsdk:"values"`
}

type RRSetLocRecord struct {
	Name   types.String      `tfsdk:"name"`
	Values []DnsZoneLOCValue `tfsdk:"values"`
}

type DnsZoneLOCValue struct {
	Altitude            types.Float32 `tfsdk:"altitude"`
	LocationDiameter    types.Float32 `tfsdk:"location_diameter"`
	VerticalPrecision   types.Float32 `tfsdk:"vertical_precision"`
	LatitudeMinute      types.Int32   `tfsdk:"latitude_minute"`
	LatitudeSecond      types.Float32 `tfsdk:"latitude_second"`
	LongitudeDegree     types.Int32   `tfsdk:"longitude_degree"`
	LongitudeMinute     types.Int32   `tfsdk:"longitude_minute"`
	LongitudeSecond     types.Float32 `tfsdk:"longitude_second"`
	HorizontalPrecision types.Float32 `tfsdk:"horizontal_precision"`
	LatitudeDegree      types.Int32   `tfsdk:"latitude_degree"`
	LatitudeHemisphere  types.List    `tfsdk:"latitude_hemisphere"`
	LongitudeHemisphere types.List    `tfsdk:"longitude_hemisphere"`
}

type RRSetLbRecord struct {
	Name  types.String        `tfsdk:"name"`
	Value *ObjectRefTypeModel `tfsdk:"value"`
}

type DnsZoneDNSSECMode struct {
	ModeChoice *DNSSECModeMode `tfsdk:"mode_choice"`
}

type DNSSECModeMode struct {
	Enable types.Bool `tfsdk:"enable"`
}

type PrimaryDNSConfigSoaRecordParametersChoice struct {
	SoaParameters        *PrimaryDNSConfigSoaParameters `tfsdk:"soa_parameters"`
	DefaultSoaParameters types.Bool                     `tfsdk:"default_soa_parameters"`
}

type PrimaryDNSConfigSoaParameters struct {
	Refresh     types.Int32 `tfsdk:"refresh"`
	Retry       types.Int32 `tfsdk:"retry"`
	Expire      types.Int32 `tfsdk:"expire"`
	NegativeTtl types.Int32 `tfsdk:"negative_ttl"`
	Ttl         types.Int32 `tfsdk:"ttl"`
}

type MessageMetaType struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type DnsZoneRRSetGroup struct {
	Metadata MessageMetaType `tfsdk:"metadata"`
	RrSet    []DnsZoneRRSet  `tfsdk:"rr_set"`
}

// ResourceF5XCDnsZoneModel describes the resource data model.
type ResourceF5XCDnsZoneModel struct {
	Name              types.String           `tfsdk:"name"`
	Namespace         types.String           `tfsdk:"namespace"`
	Uid               types.String           `tfsdk:"uid"`
	Labels            types.Map              `tfsdk:"labels"`
	Annotations       types.Map              `tfsdk:"annotations"`
	Description       types.String           `tfsdk:"description"`
	ServiceInstanceId types.String           `tfsdk:"service_instance_id"`
	NumOfDnsRecords   types.Int32            `tfsdk:"num_of_dns_records"`
	DnsTypeChoice     *GlobalSpecTypeDnsType `tfsdk:"dns_type_choice"`
	SubscriptionId    types.String           `tfsdk:"subscription_id"`
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceF5XCDnsZone{}
var _ resource.ResourceWithImportState = &ResourceF5XCDnsZone{}

var defaultRrSetGroupObjectType attr.Type = types.ObjectType{
	AttrTypes: map[string]attr.Type{ // Map keys are attr.Type
		"ttl":         types.Int64Type,
		"description": types.StringType,
		"ns_record": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name": types.StringType,
				"values": attr.Type(basetypes.ListType{
					ElemType: types.StringType,
				}),
			},
		},
	},
}

// Define the global variable using the concrete type
var nsRecordObjectType types.ObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name": types.StringType,
		"values": attr.Type(basetypes.ListType{
			ElemType: types.StringType, // Your version's required syntax
		}),
	},
}

func NewResourceF5XCDnsZone() resource.Resource {
	return &ResourceF5XCDnsZone{}
}

func (r *ResourceF5XCDnsZone) Metadata(ctx context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_dns_zone"
}

func (r *ResourceF5XCDnsZone) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "DnsZone resource",
		Description:         "DnsZone resource",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "name This is the name of configuration object. It has to be unique within the namespace. It can only be specified during create API and cannot be changed during replace API. The value of name has to follow DNS-1035 format.",
				Required:    true,
				Optional:    false,
				Computed:    false,
			},
			"namespace": schema.StringAttribute{
				Description: "namespace This defines the workspace within which each the configuration object is to be created. Must be a DNS_LABEL format. For a namespace object itself, namespace value will be ''",
				Required:    true,
				Optional:    false,
				Computed:    false,
			},
			"uid": schema.StringAttribute{
				Description: "uid uid is the unique in time and space value for this object. Object create will fail if provided by the client and the value exists in the system. Typically generated by the server on successful creation of an object and is not allowed to change once populated. Shadowed by SystemObjectMeta's uid field.",
				Required:    false,
				Optional:    false,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"labels": schema.MapAttribute{
				Description: "labels Map of string keys and values that can be used to organize and categorize (scope and select) objects as chosen by the user. Values specified here will be used by selector expression",
				Required:    false,
				Optional:    true,
				Computed:    false,
				ElementType: types.StringType,
			},
			"annotations": schema.MapAttribute{
				Description: "annotations Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects.",
				Required:    false,
				Optional:    true,
				Computed:    false,
				ElementType: types.StringType,
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.LengthBetween(1, 64)),
					mapvalidator.ValueStringsAre(stringvalidator.LengthBetween(1, 1024)),
				},
			},
			"description": schema.StringAttribute{
				Description: "description Human readable description for the object",
				Required:    false,
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					bytesvalidator.BytesAtMost(1200),
				},
			},
			"service_instance_id": schema.StringAttribute{
				Description: "Service Instance ID Service Instance ID is an internal identifier for backend dns services",
				Required:    false,
				Optional:    true,
				Computed:    false,
			},
			"num_of_dns_records": schema.Int32Attribute{
				Description: "Number of DNS records Number of DNS records displays number of dns records for each zone",
				Required:    false,
				Optional:    true,
				Computed:    false,
			},
			"dns_type_choice": schema.SingleNestedAttribute{
				Description: "Choice to DNS zone type Select type of zone whether its primary or secondary",
				Required:    true,
				Optional:    false,
				Computed:    false,
				Validators: []validator.Object{
					atmostoneofvalidator.AtMostOneOf(
						"primary",
						"secondary",
					),
				},
				Attributes: map[string]schema.Attribute{
					"secondary": schema.SingleNestedAttribute{
						Description: "Secondary DNS Configuration Secondary DNS",
						Required:    false,
						Optional:    true,
						Computed:    false,
						Attributes: map[string]schema.Attribute{
							"tsig_key_name": schema.StringAttribute{
								Description: "TSIG Key Name TSIG key name as used in TSIG protocol extension",
								Required:    false,
								Optional:    true,
								Computed:    false,
							},
							"tsig_key_algorithm": schema.ListAttribute{
								Description: "TSIG Key Algorithm TSIG key value must be compatible with the specified algorithm",
								ElementType: types.StringType,
								Required:    false,
								Optional:    true,
								Computed:    true,
								Default:     listdefault.StaticValue(utils.GetListStringDefaultValue("HMAC_MD5")),
								Validators: []validator.List{
									listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"UNDEFINED", "HMAC_MD5", "HMAC_SHA1", "HMAC_SHA224", "HMAC_SHA256", "HMAC_SHA384", "HMAC_SHA512"}...)),
									listvalidator.SizeAtMost(1),
								},
							},
							"tsig_key_value": schema.SingleNestedAttribute{
								Description: "TSIG Key value",
								Required:    true,
								Optional:    false,
								Computed:    false,
								Attributes: map[string]schema.Attribute{
									"secret_info_choice": schema.SingleNestedAttribute{
										Description: "SecretInfoOneof Secret can be one of the following types - * Blindfold Secret - Secret is managed by F5XC Secret Management System * Clear Secret - Secret is not encrypted",
										Required:    true,
										Optional:    false,
										Computed:    false,
										Validators: []validator.Object{
											atmostoneofvalidator.AtMostOneOf(
												"blindfold_secret_info",
												"clear_secret_info",
											),
										},
										Attributes: map[string]schema.Attribute{
											"blindfold_secret_info": schema.SingleNestedAttribute{
												Description: "Blindfold Secret Blindfold Secret is used for the secrets managed by F5XC Secret Management Service",
												Required:    false,
												Optional:    true,
												Computed:    false,
												Attributes: map[string]schema.Attribute{
													"decryption_provider": schema.StringAttribute{
														Description: "Decryption Provider Name of the Secret Management Access object that contains information about the backend Secret Management service.",
														Required:    false,
														Optional:    true,
														Computed:    false,
													},
													"store_provider": schema.StringAttribute{
														Description: "Store Provider Name of the Secret Management Access object that contains information about the store to get encrypted bytes This field needs to be provided only if the url scheme is not string:///",
														Required:    false,
														Optional:    true,
														Computed:    false,
													},
													"location": schema.StringAttribute{
														Description: "Location Location is the uri_ref. It could be in url format for string:/// Or it could be a path if the store provider is an http/https location",
														Required:    true,
														Optional:    false,
														Computed:    false,
													},
												},
											},
											"clear_secret_info": schema.SingleNestedAttribute{
												Description: "Clear Secret Clear Secret is used for the secrets that are not encrypted",
												Required:    false,
												Optional:    true,
												Computed:    false,
												Attributes: map[string]schema.Attribute{
													"provider_dns_zone": schema.StringAttribute{
														Description: "Provider Name of the Secret Management Access object that contains information about the store to get encrypted bytes This field needs to be provided only if the url scheme is not string:///",
														Required:    false,
														Optional:    true,
														Computed:    false,
													},
													"url": schema.StringAttribute{
														Description: "URL URL of the secret. Currently supported URL schemes is string:///. For string:/// scheme, Secret needs to be encoded Base64 format. When asked for this secret, caller will get Secret bytes after Base64 decoding.",
														Required:    true,
														Optional:    false,
														Computed:    false,
														Validators: []validator.String{
															bytesvalidator.BytesAtMost(131072),
														},
													},
												},
											},
										},
									},
								},
							},
							"last_axfr_timestamp": schema.StringAttribute{
								Description: "Last AXFR timestamp Last successful zone transfer timestamp.",
								CustomType:  timetypes.RFC3339Type{},
								Required:    false,
								Optional:    false,
								Computed:    true,
							},
							"primary_servers": schema.ListAttribute{
								Description: "List of zone primary servers",
								ElementType: types.StringType,
								Required:    true,
								Optional:    false,
								Computed:    false,
								Validators: []validator.List{
									listvalidator.UniqueValues(),
								},
							},
						},
					},
					"primary": schema.SingleNestedAttribute{
						Description: "Primary DNS Configuration Primary DNS",
						Required:    false,
						Optional:    true,
						Computed:    false,
						Attributes: map[string]schema.Attribute{
							"admin": schema.StringAttribute{
								Description: "Administrator",
								Required:    false,
								Optional:    false,
								Computed:    true,
							},
							"default_rr_set_group": schema.ListNestedAttribute{
								Computed: true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"ttl": schema.Int64Attribute{
											Computed:    true,
											Description: "Time to Live (TTL) for the resource record set.",
										},
										"ns_record": schema.SingleNestedAttribute{
											Computed:    true,
											Description: "The Name Server (NS) record.",
											Attributes: map[string]schema.Attribute{
												"name": schema.StringAttribute{
													Computed: true,
												},
												"values": schema.ListAttribute{
													Computed:    true,
													ElementType: types.StringType,
												},
											},
										},
										"description": schema.StringAttribute{
											Computed:    true,
											Description: "A description for the record set group.",
										},
									},
								},
								Description: "The default resource record set group returned by the service.",
							},
							"dnssec_mode": schema.SingleNestedAttribute{
								Description: "DNSSEC Mode",
								Required:    true,
								Optional:    false,
								Computed:    false,
								Attributes: map[string]schema.Attribute{
									"mode_choice": schema.SingleNestedAttribute{
										Description: "DNSSEC Mode DNSSEC Mode of operation",
										Required:    true,
										Optional:    false,
										Computed:    false,
										Validators: []validator.Object{
											atmostoneofvalidator.AtMostOneOf(
												"enable",
											),
										},
										Attributes: map[string]schema.Attribute{
											"enable": schema.BoolAttribute{
												Description: "Enable DNSSEC enable",
												Required:    false,
												Optional:    true,
												Computed:    false,
												Validators: []validator.Bool{
													boolvalidator.Equals(true),
												},
											},
										},
									},
								},
							},
							"allow_http_lb_managed_records": schema.BoolAttribute{
								Description: "Allow HTTP Load Balancer Managed Records Option to allow user-created HTTP, TCP, and CDN load balancer related resource records to be automatically managed in a protected RRset.",
								Required:    false,
								Optional:    true,
								Computed:    true,
								Default:     booldefault.StaticBool(false),
							},
							"soa_record_parameters_choice": schema.SingleNestedAttribute{
								Description: "SOA Record Parameters Start Of Authority (SOA) Record Properties",
								Required:    true,
								Optional:    false,
								Computed:    false,
								Validators: []validator.Object{
									atmostoneofvalidator.AtMostOneOf(
										"default_soa_parameters",
										"soa_parameters",
									),
								},
								Attributes: map[string]schema.Attribute{
									"soa_parameters": schema.SingleNestedAttribute{
										Description: "SOARecordParameterConfig Configure SOA Record parameters",
										Required:    false,
										Optional:    true,
										Computed:    false,
										Attributes: map[string]schema.Attribute{
											"refresh": schema.Int32Attribute{
												Description: "Refresh (in seconds) refresh value indicates when secondary nameservers should query for the SOA record to detect zone changes",
												Required:    false,
												Optional:    true,
												Computed:    true,
												Default:     int32default.StaticInt32(86400),
												Validators: []validator.Int32{
													int32validator.Between(3600, 2147483647),
												},
											},
											"retry": schema.Int32Attribute{
												Description: "Retry (in seconds) retry value indicates when secondary nameservers should retry to request the serial number if primary does not respond",
												Required:    false,
												Optional:    true,
												Computed:    true,
												Default:     int32default.StaticInt32(7200),
												Validators: []validator.Int32{
													int32validator.Between(60, 2147483647),
												},
											},
											"expire": schema.Int32Attribute{
												Description: "Expire (in seconds) expire value indicates when secondary nameservers should stop answering request for this zone if primary does not respond",
												Required:    false,
												Optional:    true,
												Computed:    true,
												Default:     int32default.StaticInt32(360000),
												Validators: []validator.Int32{
													int32validator.Between(0, 2147483647),
												},
											},
											"negative_ttl": schema.Int32Attribute{
												Description: "Negative TTL (in seconds) negative ttl value indicates how long to cache non-existent resource record for this zone",
												Required:    false,
												Optional:    true,
												Computed:    true,
												Default:     int32default.StaticInt32(1800),
												Validators: []validator.Int32{
													int32validator.Between(0, 2147483647),
												},
											},
											"ttl": schema.Int32Attribute{
												Description: "TTL SOA record time to live (in seconds)",
												Required:    false,
												Optional:    true,
												Computed:    true,
												Default:     int32default.StaticInt32(86400),
												Validators: []validator.Int32{
													int32validator.Between(0, 2147483647),
												},
											},
										},
									},
									"default_soa_parameters": schema.BoolAttribute{
										Description: "Default parameters Use default parameters",
										Required:    false,
										Optional:    true,
										Computed:    false,
										Validators: []validator.Bool{
											boolvalidator.Equals(true),
										},
									},
								},
							},
							"rr_set_group": schema.ListNestedAttribute{
								Description: "Record Sets Group",
								Required:    false,
								Optional:    true,
								Computed:    false,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"metadata": schema.SingleNestedAttribute{
											Description: "metadata",
											Required:    true,
											Optional:    false,
											Computed:    false,
											Attributes: map[string]schema.Attribute{
												"name": schema.StringAttribute{
													Description: "name This is the name of the message. The value of name has to follow DNS-1035 format.",
													Required:    true,
													Optional:    false,
													Computed:    false,
													Validators: []validator.String{
														stringvalidator.LengthAtLeast(1),
													},
												},
												"description": schema.StringAttribute{
													Description: "description Human readable description.",
													Required:    false,
													Optional:    true,
													Computed:    false,
													Validators: []validator.String{
														stringvalidator.LengthAtMost(256),
													},
												},
											},
										},
										"rr_set": schema.ListNestedAttribute{
											Description: "Record Sets Collection of DNS resource record sets",
											Required:    false,
											Optional:    true,
											Computed:    false,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"ttl": schema.Int32Attribute{
														Description: "TTL",
														Required:    false,
														Optional:    true,
														Computed:    true,
														Default:     int32default.StaticInt32(3600),
														Validators: []validator.Int32{
															int32validator.Between(60, 2147483647),
														},
													},
													"type_record_set_choice": schema.SingleNestedAttribute{
														Description: "Record Set",
														Required:    true,
														Optional:    false,
														Computed:    false,
														Validators: []validator.Object{
															atmostoneofvalidator.AtMostOneOf(
																"cds_record",
																"alias_record",
																"caa_record",
																"mx_record",
																"ns_record",
																"naptr_record",
																"afsdb_record",
																"loc_record",
																"srv_record",
																"lb_record",
																"sshfp_record",
																"cert_record",
																"ds_record",
																"cname_record",
																"ptr_record",
																"txt_record",
																"tlsa_record",
																"a_record",
																"aaaa_record",
																"eui48_record",
																"eui64_record",
															),
														},
														Attributes: map[string]schema.Attribute{
															"naptr_record": schema.SingleNestedAttribute{
																Description: "NAPTR Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name NAPTR Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^([a-zA-Z0-9*?]|([a-zA-Z0-9?*]+-[a-zA-Z0-9*?]+)){0,253}`),
																				"must be valid name",
																			),
																		},
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "NAPTR Values",
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"order": schema.Int32Attribute{
																					Description: "Order Order in which the NAPTR records must be processed. A lower number indicates a higher preference.",
																					Required:    false,
																					Optional:    true,
																					Computed:    true,
																					Default:     int32default.StaticInt32(10),
																					Validators: []validator.Int32{
																						int32validator.Between(0, 65535),
																					},
																				},
																				"preference": schema.Int32Attribute{
																					Description: "Preference Preference when records have the same order. A lower number indicates a higher preference.",
																					Required:    false,
																					Optional:    true,
																					Computed:    true,
																					Default:     int32default.StaticInt32(10),
																					Validators: []validator.Int32{
																						int32validator.Between(0, 65535),
																					},
																				},
																				"flags": schema.StringAttribute{
																					Description: "Flags Flag to control aspects of the rewriting and interpretation of the fields in the record. At this time only four flags, S/A/U/P, are defined.",
																					Required:    false,
																					Optional:    true,
																					Computed:    true,
																					Default:     stringdefault.StaticString(""),
																					Validators: []validator.String{
																						stringvalidator.LengthAtMost(255),
																						stringvalidator.RegexMatches(
																							regexp.MustCompile(`^(S|s|A|a|U|u|P|p)$`),
																							"must be valid flags",
																						),
																					},
																				},
																				"service": schema.StringAttribute{
																					Description: "Protocol Resolution Service Specifies the service(s) available down this rewrite path.",
																					Required:    false,
																					Optional:    true,
																					Computed:    true,
																					Default:     stringdefault.StaticString(""),
																					Validators: []validator.String{
																						stringvalidator.LengthAtMost(255),
																						stringvalidator.RegexMatches(
																							regexp.MustCompile(`^([A-Za-z][A-Za-z0-9]{0,31}(\\+[A-Za-z][A-Za-z0-9]{0,31})*$|^$)`),
																							"must be valid service",
																						),
																					},
																				},
																				"regexp": schema.StringAttribute{
																					Description: "Regular Expression Regular expression to construct the next domain name to lookup.",
																					Required:    false,
																					Optional:    true,
																					Computed:    true,
																					Default:     stringdefault.StaticString(""),
																					Validators: []validator.String{
																						stringvalidator.LengthAtMost(255),
																					},
																				},
																				"replacement": schema.StringAttribute{
																					Description: "Replacement The next NAME to query for NAPTR, SRV, or address records depending on the value of the flags field.",
																					Required:    false,
																					Optional:    true,
																					Computed:    true,
																					Default:     stringdefault.StaticString(""),
																				},
																			},
																		},
																	},
																},
															},
															"srv_record": schema.SingleNestedAttribute{
																Description: "SRV Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name SRV Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^([*]|[a-zA-Z0-9-_]{1,63})([.][a-zA-Z0-9-_]{1,63})*$`),
																				"must be valid name",
																			),
																		},
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "SRV Value",
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"priority": schema.Int32Attribute{
																					Description: "Priority Priority of the target. A lower number indicates a higher preference.",
																					Required:    false,
																					Optional:    true,
																					Computed:    true,
																					Default:     int32default.StaticInt32(10),
																					Validators: []validator.Int32{
																						int32validator.Between(0, 65535),
																					},
																				},
																				"weight": schema.Int32Attribute{
																					Description: "Weight Weight of the target. A higher number indicates a higher preference.",
																					Required:    false,
																					Optional:    true,
																					Computed:    true,
																					Default:     int32default.StaticInt32(10),
																					Validators: []validator.Int32{
																						int32validator.Between(0, 65535),
																					},
																				},
																				"port": schema.Int32Attribute{
																					Description: "Port Port on which the service can be found",
																					Required:    false,
																					Optional:    true,
																					Computed:    true,
																					Default:     int32default.StaticInt32(10),
																					Validators: []validator.Int32{
																						int32validator.Between(0, 65535),
																					},
																				},
																				"target": schema.StringAttribute{
																					Description: "Target Hostname of the machine providing the service",
																					Required:    false,
																					Optional:    true,
																					Computed:    false,
																					Validators: []validator.String{
																						stringvalidator.RegexMatches(
																							regexp.MustCompile(`^[.]$|^([a-zA-Z0-9]{1}[a-zA-Z0-9_-]{0,62})(\\.[a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62})*?(\\.[a-zA-Z]{1}[a-zA-Z0-9]{0,62})\\.?$`),
																							"must be valid target",
																						),
																					},
																				},
																			},
																		},
																	},
																},
															},
															"tlsa_record": schema.SingleNestedAttribute{
																Description: "TLSA Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name TLSA Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^([a-zA-Z0-9*?]|([a-zA-Z0-9?*]+-[a-zA-Z0-9*?]+)){0,253}`),
																				"must be valid name",
																			),
																		},
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "TLSA Value",
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"certificate_usage": schema.ListAttribute{
																					Description: "Certificate Usage TLSA Record Certificate Usage.",
																					ElementType: types.StringType,
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.List{
																						listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"CertificateAuthorityConstraint", "ServiceCertificateConstraint", "TrustAnchorAssertion", "DomainIssuedCertificate"}...)),
																						listvalidator.SizeAtMost(1),
																					},
																				},
																				"selector": schema.ListAttribute{
																					Description: "Selector TLSA Record Selector.",
																					ElementType: types.StringType,
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.List{
																						listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"FullCertificate", "UseSubjectPublicKey"}...)),
																						listvalidator.SizeAtMost(1),
																					},
																				},
																				"matching_type": schema.ListAttribute{
																					Description: "Matching Type TLSA Record Matching Type.",
																					ElementType: types.StringType,
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.List{
																						listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"SHA256", "SHA512", "NoHash"}...)),
																						listvalidator.SizeAtMost(1),
																					},
																				},
																				"certificate_association_data": schema.StringAttribute{
																					Description: "Certificate Association Data The actual data to be matched given the settings of the other fields.",
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.String{
																						stringvalidator.LengthBetween(1, 4096),
																					},
																				},
																			},
																		},
																	},
																},
															},
															"eui48_record": schema.SingleNestedAttribute{
																Description: "EUI48 Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"value": schema.StringAttribute{
																		Description: "EUI48 Identifier A valid eui48 identifier, for example: 01-23-45-67-89-ab",
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.LengthBetween(17, 17),
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^([0-9A-Fa-f]{2}-){5}([0-9A-Fa-f]{2})$`),
																				"must be valid value",
																			),
																		},
																	},
																	"name": schema.StringAttribute{
																		Description: "Record Name EUI48 Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^([a-zA-Z0-9*?]|([a-zA-Z0-9?*]+-[a-zA-Z0-9*?]+)){0,253}`),
																				"must be valid name",
																			),
																		},
																	},
																},
															},
															"eui64_record": schema.SingleNestedAttribute{
																Description: "EUI64 Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name EUI64 Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^([a-zA-Z0-9*?]|([a-zA-Z0-9?*]+-[a-zA-Z0-9*?]+)){0,253}`),
																				"must be valid name",
																			),
																		},
																	},
																	"value": schema.StringAttribute{
																		Description: "EUI64 Identifier A valid EUI64 identifier, for example: 01-23-45-67-89-ab-cd-ef",
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.LengthBetween(23, 23),
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^([0-9A-Fa-f]{2}-){7}([0-9A-Fa-f]{2})$`),
																				"must be valid value",
																			),
																		},
																	},
																},
															},
															"cds_record": schema.SingleNestedAttribute{
																Description: "CDS Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name CDS Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^([a-zA-Z0-9*?]|([a-zA-Z0-9?*]+-[a-zA-Z0-9*?]+)){0,253}`),
																				"must be valid name",
																			),
																		},
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "DS Value",
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"key_tag": schema.Int32Attribute{
																					Description: "Key Tag A short numeric value which can help quickly identify the referenced DNSKEY-record.",
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.Int32{
																						int32validator.Between(1, 65535),
																					},
																				},
																				"ds_key_algorithm": schema.ListAttribute{
																					Description: "DS Key Algorithm DS key value must be compatible with the specified algorithm.",
																					ElementType: types.StringType,
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.List{
																						listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"RSASHA1", "RSASHA1NSEC3SHA1", "RSASHA256", "RSASHA512", "ECDSAP384SHA384", "ED25519", "UNSPECIFIED", "ECDSAP256SHA256", "ED448"}...)),
																						listvalidator.SizeAtMost(1),
																					},
																				},
																				"digest_choice": schema.SingleNestedAttribute{
																					Description: "Digest",
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.Object{
																						atmostoneofvalidator.AtMostOneOf(
																							"sha1_digest",
																							"sha256_digest",
																							"sha384_digest",
																						),
																					},
																					Attributes: map[string]schema.Attribute{
																						"sha1_digest": schema.SingleNestedAttribute{
																							Description: "SHA1 Digest",
																							Required:    false,
																							Optional:    true,
																							Computed:    false,
																							Attributes: map[string]schema.Attribute{
																								"digest": schema.StringAttribute{
																									Description: "Digest The 'digest' is the DS key and the actual contents of the DS record.",
																									Required:    true,
																									Optional:    false,
																									Computed:    false,
																									Validators: []validator.String{
																										stringvalidator.LengthBetween(40, 40),
																									},
																								},
																							},
																						},
																						"sha256_digest": schema.SingleNestedAttribute{
																							Description: "SHA256 Digest",
																							Required:    false,
																							Optional:    true,
																							Computed:    false,
																							Attributes: map[string]schema.Attribute{
																								"digest": schema.StringAttribute{
																									Description: "Digest The 'digest' is the DS key and the actual contents of the DS record.",
																									Required:    true,
																									Optional:    false,
																									Computed:    false,
																									Validators: []validator.String{
																										stringvalidator.LengthBetween(64, 64),
																									},
																								},
																							},
																						},
																						"sha384_digest": schema.SingleNestedAttribute{
																							Description: "SHA384 Digest",
																							Required:    false,
																							Optional:    true,
																							Computed:    false,
																							Attributes: map[string]schema.Attribute{
																								"digest": schema.StringAttribute{
																									Description: "Digest The 'digest' is the DS key and the actual contents of the DS record.",
																									Required:    true,
																									Optional:    false,
																									Computed:    false,
																									Validators: []validator.String{
																										stringvalidator.LengthBetween(96, 96),
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
															},
															"caa_record": schema.SingleNestedAttribute{
																Description: "CAA Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name CAA Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^$|^([*]|[a-zA-Z0-9-/_]{1,63})([.][a-zA-Z0-9-/_]{1,63})*$`),
																				"must be valid name",
																			),
																		},
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "CAA Record Value",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"tag": schema.StringAttribute{
																					Description: "Certification authority authorization tag",
																					Required:    false,
																					Optional:    true,
																					Computed:    false,
																				},
																				"value": schema.StringAttribute{
																					Description: "Certification authority authorization value",
																					Required:    false,
																					Optional:    true,
																					Computed:    false,
																					Validators: []validator.String{
																						stringvalidator.LengthBetween(1, 1024),
																					},
																				},
																				"flags": schema.Int32Attribute{
																					Description: "Certification authority authorization flags This flag should be an integer between 0 and 255.",
																					Required:    false,
																					Optional:    true,
																					Computed:    true,
																					Default:     int32default.StaticInt32(0),
																					Validators: []validator.Int32{
																						int32validator.Between(0, 255),
																					},
																				},
																			},
																		},
																	},
																},
															},
															"ns_record": schema.SingleNestedAttribute{
																Description: "NS Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name NS Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^$|^([*]|[a-zA-Z0-9-/_]{1,63})([.][a-zA-Z0-9-/_]{1,63})*$`),
																				"must be valid name",
																			),
																		},
																	},
																	"values": schema.ListAttribute{
																		Description: "Name-servers",
																		ElementType: types.StringType,
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		Validators: []validator.List{
																			listvalidator.UniqueValues(),
																		},
																	},
																},
															},
															"afsdb_record": schema.SingleNestedAttribute{
																Description: "AFSDB Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name AFSDB Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^([a-zA-Z0-9*?]|([a-zA-Z0-9?*]+-[a-zA-Z0-9*?]+)){0,253}`),
																				"must be valid name",
																			),
																		},
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "AFSDB Value",
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"subtype": schema.ListAttribute{
																					Description: "Subtype AFSDB Record Subtype.",
																					ElementType: types.StringType,
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.List{
																						listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"AFSVolumeLocationServer", "DCEAuthenticationServer", "NONE"}...)),
																						listvalidator.SizeAtMost(1),
																					},
																				},
																				"hostname": schema.StringAttribute{
																					Description: "Hostname Server name of the AFS cell database server or the DCE name server.",
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																				},
																			},
																		},
																	},
																},
															},
															"txt_record": schema.SingleNestedAttribute{
																Description: "TXT Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name TXT Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^$|^([*]|[a-zA-Z0-9-/_]{1,63})([.][a-zA-Z0-9-/_]{1,63})*$`),
																				"must be valid name",
																			),
																		},
																	},
																	"values": schema.ListAttribute{
																		Description: "Text",
																		ElementType: types.StringType,
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		Validators: []validator.List{
																			listvalidator.UniqueValues(),
																		},
																	},
																},
															},
															"a_record": schema.SingleNestedAttribute{
																Description: "A Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name A Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^$|^([*]|[a-zA-Z0-9-/_]{1,63})([.][a-zA-Z0-9-/_]{1,63})*$`),
																				"must be valid name",
																			),
																		},
																	},
																	"values": schema.ListAttribute{
																		Description: "IPv4 addresses A valid IPv4 address, for example: 1.1.1.1",
																		ElementType: types.StringType,
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		Validators: []validator.List{
																			listvalidator.UniqueValues(),
																		},
																	},
																},
															},
															"alias_record": schema.SingleNestedAttribute{
																Description: "ALIAS Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"value": schema.StringAttribute{
																		Description: "Domain A valid domain name, for example: example.com",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.LengthAtMost(255),
																		},
																	},
																},
															},
															"lb_record": schema.SingleNestedAttribute{
																Description: "DNS LB Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name Load Balancer record name (except for SRV DNS Load balancer record) should be a simple record name and not a subdomain of a subdomain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.LengthAtMost(255),
																		},
																	},
																	"value": schema.SingleNestedAttribute{
																		Description: "DNS LB Record",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Attributes: map[string]schema.Attribute{
																			"tenant": schema.StringAttribute{
																				Description: "",
																				Required:    true,
																				Optional:    false,
																				Computed:    false,
																				Validators: []validator.String{
																					stringvalidator.LengthAtLeast(2),
																				},
																			},
																			"namespace": schema.StringAttribute{
																				Description: "",
																				Required:    true,
																				Optional:    false,
																				Computed:    false,
																				Validators: []validator.String{
																					stringvalidator.LengthAtLeast(2),
																				},
																			},
																			"name": schema.StringAttribute{
																				Description: "",
																				Required:    true,
																				Optional:    false,
																				Computed:    false,
																				Validators: []validator.String{
																					stringvalidator.LengthAtLeast(2),
																				},
																			},
																		},
																	},
																},
															},
															"sshfp_record": schema.SingleNestedAttribute{
																Description: "SSHFP Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"values": schema.ListNestedAttribute{
																		Description: "SSHFP Value",
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"algorithm": schema.ListAttribute{
																					Description: "SSHFP Algorithm Algorithm of the public key.",
																					ElementType: types.StringType,
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.List{
																						listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"UNSPECIFIEDALGORITHM", "RSA", "DSA", "ECDSA", "Ed25519", "Ed448"}...)),
																						listvalidator.SizeAtMost(1),
																					},
																				},
																				"fingerprint_type_choice": schema.SingleNestedAttribute{
																					Description: "Fingerprint Type Algorithm used to calculate the fingerprint of the public key and the hexadecimal representation of the hash result of the SSH key as text.",
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.Object{
																						atmostoneofvalidator.AtMostOneOf(
																							"sha1_fingerprint",
																							"sha256_fingerprint",
																						),
																					},
																					Attributes: map[string]schema.Attribute{
																						"sha1_fingerprint": schema.SingleNestedAttribute{
																							Description: "SHA1 Fingerprint",
																							Required:    false,
																							Optional:    true,
																							Computed:    false,
																							Attributes: map[string]schema.Attribute{
																								"fingerprint": schema.StringAttribute{
																									Description: "Fingerprint The 'fingerprint' is the DS key and the actual contents of the DS record.",
																									Required:    true,
																									Optional:    false,
																									Computed:    false,
																									Validators: []validator.String{
																										stringvalidator.LengthBetween(40, 40),
																									},
																								},
																							},
																						},
																						"sha256_fingerprint": schema.SingleNestedAttribute{
																							Description: "SHA256 Digest",
																							Required:    false,
																							Optional:    true,
																							Computed:    false,
																							Attributes: map[string]schema.Attribute{
																								"fingerprint": schema.StringAttribute{
																									Description: "Fingerprint The 'fingerprint' is the DS key and the actual contents of the DS record.",
																									Required:    true,
																									Optional:    false,
																									Computed:    false,
																									Validators: []validator.String{
																										stringvalidator.LengthBetween(64, 64),
																									},
																								},
																							},
																						},
																					},
																				},
																			},
																		},
																	},
																	"name": schema.StringAttribute{
																		Description: "Record Name SSHFP Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^([a-zA-Z0-9*?]|([a-zA-Z0-9?*]+-[a-zA-Z0-9*?]+)){0,253}`),
																				"must be valid name",
																			),
																		},
																	},
																},
															},
															"cert_record": schema.SingleNestedAttribute{
																Description: "CERT Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name CERT Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^([a-zA-Z0-9*?]|([a-zA-Z0-9?*]+-[a-zA-Z0-9*?]+)){0,253}`),
																				"must be valid name",
																			),
																		},
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "CERT Value",
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"cert_key_tag": schema.Int32Attribute{
																					Description: "Key Tag",
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.Int32{
																						int32validator.Between(0, 65535),
																					},
																				},
																				"algorithm": schema.ListAttribute{
																					Description: "Algorithm",
																					ElementType: types.StringType,
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.List{
																						listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"PRIVATEDNS", "PRIVATEOID", "RSAMD5", "DH", "DSASHA1", "ECC", "RSASHA1ALGORITHM", "INDIRECT", "RESERVEDALGORITHM"}...)),
																						listvalidator.SizeAtMost(1),
																					},
																				},
																				"certificate": schema.StringAttribute{
																					Description: "Certificate Certificate in base 64 format.",
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.String{
																						stringvalidator.LengthBetween(1, 4096),
																					},
																				},
																				"cert_type": schema.ListAttribute{
																					Description: "Type",
																					ElementType: types.StringType,
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.List{
																						listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"URI_", "OID", "INVALIDCERTTYPE", "PGP", "IPKIX", "ACPKIX", "PKIX", "SPKI", "ISPKI", "IPGP", "IACPKIX"}...)),
																						listvalidator.SizeAtMost(1),
																					},
																				},
																			},
																		},
																	},
																},
															},
															"cname_record": schema.SingleNestedAttribute{
																Description: "CNAME Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"value": schema.StringAttribute{
																		Description: "Domain",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.LengthAtMost(255),
																		},
																	},
																	"name": schema.StringAttribute{
																		Description: "Record Name CName Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^([*]|[a-zA-Z0-9-/_]{1,63})([.][a-zA-Z0-9-/_]{1,63})*$`),
																				"must be valid name",
																			),
																		},
																	},
																},
															},
															"ptr_record": schema.SingleNestedAttribute{
																Description: "PTR Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name PTR Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^$|^([*]|[a-zA-Z0-9-/_]{1,63})([.][a-zA-Z0-9-/_]{1,63})*$`),
																				"must be valid name",
																			),
																		},
																	},
																	"values": schema.ListAttribute{
																		Description: "Domain names",
																		ElementType: types.StringType,
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		Validators: []validator.List{
																			listvalidator.UniqueValues(),
																		},
																	},
																},
															},
															"aaaa_record": schema.SingleNestedAttribute{
																Description: "AAAA Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name AAAA Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^$|^([*]|[a-zA-Z0-9-/_]{1,63})([.][a-zA-Z0-9-/_]{1,63})*$`),
																				"must be valid name",
																			),
																		},
																	},
																	"values": schema.ListAttribute{
																		Description: "IPv6 addresses A valid IPv6 address, for example: 2001:0db8:85a3:0000:0000:8a2e:0370:7334",
																		ElementType: types.StringType,
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		Validators: []validator.List{
																			listvalidator.UniqueValues(),
																		},
																	},
																},
															},
															"loc_record": schema.SingleNestedAttribute{
																Description: "LOC Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"values": schema.ListNestedAttribute{
																		Description: "LOC Values",
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"latitude_second": schema.Float64Attribute{
																					Description: "Latitude second Latitude second, an decimal between 0 and 59.999, including 0 and 59.999",
																					Required:    false,
																					Optional:    true,
																					Computed:    false,
																				},
																				"longitude_degree": schema.Int32Attribute{
																					Description: "Longitude degree Longitude degree, an integer between 0 and 180, including 0 and 180",
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																				},
																				"longitude_minute": schema.Int32Attribute{
																					Description: "Longitude minute Longitude minute, an integer between 0 and 59, including 0 and 59",
																					Required:    false,
																					Optional:    true,
																					Computed:    false,
																				},
																				"longitude_second": schema.Float64Attribute{
																					Description: "Longitude second Longitude second, an decimal between 0 and 59.999, including 0 and 59.999",
																					Required:    false,
																					Optional:    true,
																					Computed:    false,
																				},
																				"horizontal_precision": schema.Float64Attribute{
																					Description: "Horizontal Precision Horizontal Precision in meters",
																					Required:    false,
																					Optional:    true,
																					Computed:    false,
																				},
																				"latitude_degree": schema.Int32Attribute{
																					Description: "Latitude degree Latitude degree, an integer between 0 and 90, including 0 and 90",
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																				},
																				"latitude_hemisphere": schema.ListAttribute{
																					Description: "Latitude hemisphere Latitude hemisphere, N or S",
																					ElementType: types.StringType,
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.List{
																						listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"N", "S"}...)),
																						listvalidator.SizeAtMost(1),
																					},
																				},
																				"longitude_hemisphere": schema.ListAttribute{
																					Description: "Longitude hemisphere Longitude hemisphere, E or W",
																					ElementType: types.StringType,
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.List{
																						listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"E", "W"}...)),
																						listvalidator.SizeAtMost(1),
																					},
																				},
																				"altitude": schema.Float64Attribute{
																					Description: "Altitude Altitude in meters",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"location_diameter": schema.Float64Attribute{
																					Description: "Size Diameter of a sphere enclosing the described entity, in meters",
																					Required:    false,
																					Optional:    true,
																					Computed:    false,
																				},
																				"vertical_precision": schema.Float64Attribute{
																					Description: "Vertical Precision Vertical Precision in meters",
																					Required:    false,
																					Optional:    true,
																					Computed:    false,
																				},
																				"latitude_minute": schema.Int32Attribute{
																					Description: "Latitude minute Latitude minute, an integer between 0 and 59, including 0 and 59",
																					Required:    false,
																					Optional:    true,
																					Computed:    false,
																				},
																			},
																		},
																	},
																	"name": schema.StringAttribute{
																		Description: "Record Name LOC Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^([a-zA-Z0-9*?]|([a-zA-Z0-9?*]+-[a-zA-Z0-9*?]+)){0,253}`),
																				"must be valid name",
																			),
																		},
																	},
																},
															},
															"ds_record": schema.SingleNestedAttribute{
																Description: "DS Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name DS Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^([a-zA-Z0-9*?]|([a-zA-Z0-9?*]+-[a-zA-Z0-9*?]+)){0,253}`),
																				"must be valid name",
																			),
																		},
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "DS Value",
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"key_tag": schema.Int32Attribute{
																					Description: "Key Tag A short numeric value which can help quickly identify the referenced DNSKEY-record.",
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.Int32{
																						int32validator.Between(1, 65535),
																					},
																				},
																				"ds_key_algorithm": schema.ListAttribute{
																					Description: "DS Key Algorithm DS key value must be compatible with the specified algorithm.",
																					ElementType: types.StringType,
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.List{
																						listvalidator.ValueStringsAre(stringvalidator.OneOf([]string{"RSASHA1", "RSASHA1NSEC3SHA1", "RSASHA256", "RSASHA512", "ECDSAP384SHA384", "ED25519", "UNSPECIFIED", "ECDSAP256SHA256", "ED448"}...)),
																						listvalidator.SizeAtMost(1),
																					},
																				},
																				"digest_choice": schema.SingleNestedAttribute{
																					Description: "Digest",
																					Required:    true,
																					Optional:    false,
																					Computed:    false,
																					Validators: []validator.Object{
																						atmostoneofvalidator.AtMostOneOf(
																							"sha1_digest",
																							"sha256_digest",
																							"sha384_digest",
																						),
																					},
																					Attributes: map[string]schema.Attribute{
																						"sha1_digest": schema.SingleNestedAttribute{
																							Description: "SHA1 Digest",
																							Required:    false,
																							Optional:    true,
																							Computed:    false,
																							Attributes: map[string]schema.Attribute{
																								"digest": schema.StringAttribute{
																									Description: "Digest The 'digest' is the DS key and the actual contents of the DS record.",
																									Required:    true,
																									Optional:    false,
																									Computed:    false,
																									Validators: []validator.String{
																										stringvalidator.LengthBetween(40, 40),
																									},
																								},
																							},
																						},
																						"sha256_digest": schema.SingleNestedAttribute{
																							Description: "SHA256 Digest",
																							Required:    false,
																							Optional:    true,
																							Computed:    false,
																							Attributes: map[string]schema.Attribute{
																								"digest": schema.StringAttribute{
																									Description: "Digest The 'digest' is the DS key and the actual contents of the DS record.",
																									Required:    true,
																									Optional:    false,
																									Computed:    false,
																									Validators: []validator.String{
																										stringvalidator.LengthBetween(64, 64),
																									},
																								},
																							},
																						},
																						"sha384_digest": schema.SingleNestedAttribute{
																							Description: "SHA384 Digest",
																							Required:    false,
																							Optional:    true,
																							Computed:    false,
																							Attributes: map[string]schema.Attribute{
																								"digest": schema.StringAttribute{
																									Description: "Digest The 'digest' is the DS key and the actual contents of the DS record.",
																									Required:    true,
																									Optional:    false,
																									Computed:    false,
																									Validators: []validator.String{
																										stringvalidator.LengthBetween(96, 96),
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
															},
															"mx_record": schema.SingleNestedAttribute{
																Description: "MX Record",
																Required:    false,
																Optional:    true,
																Computed:    false,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name MX Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    true,
																		Computed:    false,
																		Validators: []validator.String{
																			stringvalidator.RegexMatches(
																				regexp.MustCompile(`^$|^([*]|[a-zA-Z0-9-/_]{1,63})([.][a-zA-Z0-9-/_]{1,63})*$`),
																				"must be valid name",
																			),
																		},
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "Value",
																		Required:    true,
																		Optional:    false,
																		Computed:    false,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"domain": schema.StringAttribute{
																					Description: "Domain Mail exchanger domain name, please provide the full hostname, for example: mail.example.com",
																					Required:    false,
																					Optional:    true,
																					Computed:    false,
																				},
																				"priority": schema.Int32Attribute{
																					Description: "Priority Mail exchanger priority code",
																					Required:    false,
																					Optional:    true,
																					Computed:    true,
																					Default:     int32default.StaticInt32(10),
																					Validators: []validator.Int32{
																						int32validator.Between(0, 65535),
																					},
																				},
																			},
																		},
																	},
																},
															},
														},
													},
													"description": schema.StringAttribute{
														Description: "Comment",
														Required:    false,
														Optional:    true,
														Computed:    false,
													},
												},
											},
										},
									},
								},
							},
							"serial": schema.Int32Attribute{
								Description: "Serial number Secondary nameservers should initiate a zone transfer when this number is incremented",
								Required:    false,
								Optional:    false,
								Computed:    true,
							},
						},
					},
				},
			},
			"subscription_id": schema.StringAttribute{
				Description: "Subscription ID Subscription ID is an internal identifier for backend dns services",
				Required:    false,
				Optional:    true,
				Computed:    false,
			},
		},
	}
}

func (r *ResourceF5XCDnsZone) Configure(ctx context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if request.ProviderData == nil {
		return
	}

	client, ok := request.ProviderData.(*client.APIClient)

	if !ok {
		response.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *APIClient, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *ResourceF5XCDnsZone) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data ResourceF5XCDnsZoneModel

	// Read Terraform plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	meta := &ves_io_schema.ObjectCreateMetaType{}
	spec := &ves_io_schema_dns_zone.CreateSpecType{}
	req := &ves_io_schema_dns_zone.CreateRequest{
		Metadata: meta,
		Spec:     spec,
	}

	// Populate createReq from data model
	// resource meta modelToSpecRoot RequiredStringField --> Name
	if !data.Name.IsNull() {
		meta.Name = string(data.Name.ValueString())
	}
	// resource meta modelToSpecRoot StringField --> Namespace
	if !data.Namespace.IsNull() {
		meta.Namespace = string(data.Namespace.ValueString())
	}

	// resource meta modelToSpecRoot MapOfStringField --> Labels
	meta.Labels = make(map[string]string, len(data.Labels.Elements()))
	diags = data.Labels.ElementsAs(ctx, &meta.Labels, false)
	response.Diagnostics.Append(diags...)
	// resource meta modelToSpecRoot MapOfStringField --> Annotations
	meta.Annotations = make(map[string]string, len(data.Annotations.Elements()))
	diags = data.Annotations.ElementsAs(ctx, &meta.Annotations, false)
	response.Diagnostics.Append(diags...)
	// resource meta modelToSpecRoot StringField --> Description
	if !data.Description.IsNull() {
		meta.Description = string(data.Description.ValueString())
	}

	// resource modelToSpecRoot RequiredOneOfField --> DnsType

	if data.DnsTypeChoice != nil {

		// resource modelToSpecNested PointerOfOneOfField --> Secondary
		if data.DnsTypeChoice.Secondary != nil {
			secondaryOneOfSpec := &ves_io_schema_dns_zone.CreateSpecType_Secondary{}
			secondaryOneOfSpec.Secondary = &ves_io_schema_dns_zone.SecondaryDNSCreateSpecType{}
			spec.DnsType = secondaryOneOfSpec

			// resource modelToSpecNested StringField --> TsigKeyName
			if !data.DnsTypeChoice.Secondary.TsigKeyName.IsNull() {
				secondaryOneOfSpec.Secondary.TsigKeyName = string(data.DnsTypeChoice.Secondary.TsigKeyName.ValueString())
			}

			// resource modelToSpecNested EnumField --> TsigKeyAlgorithm
			if !data.DnsTypeChoice.Secondary.TsigKeyAlgorithm.IsNull() && !data.DnsTypeChoice.Secondary.TsigKeyAlgorithm.IsUnknown() {
				tsigKeyAlgorithmElements := make([]types.String, 0, len(data.DnsTypeChoice.Secondary.TsigKeyAlgorithm.Elements()))
				diags = data.DnsTypeChoice.Secondary.TsigKeyAlgorithm.ElementsAs(ctx, &tsigKeyAlgorithmElements, false)
				response.Diagnostics.Append(diags...)
				secondaryOneOfSpec.Secondary.TsigKeyAlgorithm = ves_io_schema_dns_zone.TSIGKeyAlgorithm(ves_io_schema_dns_zone.TSIGKeyAlgorithm_value[tsigKeyAlgorithmElements[0].ValueString()])
			}

			// resource modelToSpecNested PointerField --> TsigKeyValue
			if data.DnsTypeChoice.Secondary.TsigKeyValue != nil {
				tsigKeyValueSpec := &ves_io_schema.SecretType{}
				secondaryOneOfSpec.Secondary.TsigKeyValue = tsigKeyValueSpec

				// resource modelToSpecNested RequiredOneOfField --> SecretInfoOneof

				// resource modelToSpecNested PointerOfOneOfField --> BlindfoldSecretInfo
				if data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo != nil {
					blindfoldSecretInfoOneOfSpec := &ves_io_schema.SecretType_BlindfoldSecretInfo{}
					blindfoldSecretInfoOneOfSpec.BlindfoldSecretInfo = &ves_io_schema.BlindfoldSecretInfoType{}
					tsigKeyValueSpec.SecretInfoOneof = blindfoldSecretInfoOneOfSpec

					// resource modelToSpecNested StringField --> DecryptionProvider
					if !data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.DecryptionProvider.IsNull() {
						blindfoldSecretInfoOneOfSpec.BlindfoldSecretInfo.DecryptionProvider = string(data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.DecryptionProvider.ValueString())
					}

					// resource modelToSpecNested StringField --> StoreProvider
					if !data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.StoreProvider.IsNull() {
						blindfoldSecretInfoOneOfSpec.BlindfoldSecretInfo.StoreProvider = string(data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.StoreProvider.ValueString())
					}

					// resource modelToSpecNested RequiredStringField --> Location
					blindfoldSecretInfoOneOfSpec.BlindfoldSecretInfo.Location = string(data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.Location.ValueString())
				}

				// resource modelToSpecNested PointerOfOneOfField --> ClearSecretInfo
				if data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo != nil {
					clearSecretInfoOneOfSpec := &ves_io_schema.SecretType_ClearSecretInfo{}
					clearSecretInfoOneOfSpec.ClearSecretInfo = &ves_io_schema.ClearSecretInfoType{}
					tsigKeyValueSpec.SecretInfoOneof = clearSecretInfoOneOfSpec

					// resource modelToSpecNested StringField --> Provider
					if !data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo.Provider.IsNull() {
						clearSecretInfoOneOfSpec.ClearSecretInfo.Provider = string(data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo.Provider.ValueString())
					}

					// resource modelToSpecNested RequiredStringField --> Url
					clearSecretInfoOneOfSpec.ClearSecretInfo.Url = string(data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo.Url.ValueString())
				}
			}

			// resource modelToSpecNested RequiredArrayOfStringField --> PrimaryServers
			elements := make([]string, 0, len(data.DnsTypeChoice.Secondary.PrimaryServers.Elements()))
			diags := data.DnsTypeChoice.Secondary.PrimaryServers.ElementsAs(ctx, &elements, false)
			response.Diagnostics.Append(diags...)
			secondaryOneOfSpec.Secondary.PrimaryServers = elements
		}

		// resource modelToSpecNested PointerOfOneOfField --> Primary
		if data.DnsTypeChoice.Primary != nil {
			primaryOneOfSpec := &ves_io_schema_dns_zone.CreateSpecType_Primary{}
			primaryOneOfSpec.Primary = &ves_io_schema_dns_zone.PrimaryDNSCreateSpecType{}
			spec.DnsType = primaryOneOfSpec

			// resource modelToSpecNested PointerField --> DnssecMode
			if data.DnsTypeChoice.Primary.DnssecMode != nil {
				dnssecModeSpec := &ves_io_schema_dns_zone.DNSSECMode{}
				primaryOneOfSpec.Primary.DnssecMode = dnssecModeSpec

				// resource modelToSpecNested RequiredOneOfField --> Mode

				// resource modelToSpecNested BoolOfOneOfPointerField --> Enable
				if !data.DnsTypeChoice.Primary.DnssecMode.ModeChoice.Enable.IsNull() {
					enableSpec := &ves_io_schema_dns_zone.DNSSECMode_Enable{}
					enableSpec.Enable = &ves_io_schema_dns_zone.DNSSECModeEnable{}
					dnssecModeSpec.Mode = enableSpec
				}
			}

			// resource modelToSpecNested BoolField --> AllowHttpLbManagedRecords
			if !data.DnsTypeChoice.Primary.AllowHttpLbManagedRecords.IsNull() {
				primaryOneOfSpec.Primary.AllowHttpLbManagedRecords = bool(data.DnsTypeChoice.Primary.AllowHttpLbManagedRecords.ValueBool())
			}

			// resource modelToSpecNested RequiredOneOfField --> SoaRecordParametersChoice

			// resource modelToSpecNested PointerOfOneOfField --> SoaParameters
			if data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters != nil {
				soaParametersOneOfSpec := &ves_io_schema_dns_zone.PrimaryDNSCreateSpecType_SoaParameters{}
				soaParametersOneOfSpec.SoaParameters = &ves_io_schema_dns_zone.SOARecordParameterConfig{}
				primaryOneOfSpec.Primary.SoaRecordParametersChoice = soaParametersOneOfSpec

				// resource modelToSpecNested UintField --> Refresh
				if !data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Refresh.IsNull() {
					soaParametersOneOfSpec.SoaParameters.Refresh = uint32(data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Refresh.ValueInt32())
				}

				// resource modelToSpecNested UintField --> Retry
				if !data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Retry.IsNull() {
					soaParametersOneOfSpec.SoaParameters.Retry = uint32(data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Retry.ValueInt32())
				}

				// resource modelToSpecNested UintField --> Expire
				if !data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Expire.IsNull() {
					soaParametersOneOfSpec.SoaParameters.Expire = uint32(data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Expire.ValueInt32())
				}

				// resource modelToSpecNested UintField --> NegativeTtl
				if !data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.NegativeTtl.IsNull() {
					soaParametersOneOfSpec.SoaParameters.NegativeTtl = uint32(data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.NegativeTtl.ValueInt32())
				}

				// resource modelToSpecNested UintField --> Ttl
				if !data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Ttl.IsNull() {
					soaParametersOneOfSpec.SoaParameters.Ttl = uint32(data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Ttl.ValueInt32())
				}
			}

			// resource modelToSpecNested BoolOfOneOfPointerField --> DefaultSoaParameters
			if !data.DnsTypeChoice.Primary.SoaRecordParametersChoice.DefaultSoaParameters.IsNull() {
				defaultSoaParametersSpec := &ves_io_schema_dns_zone.PrimaryDNSCreateSpecType_DefaultSoaParameters{}
				defaultSoaParametersSpec.DefaultSoaParameters = &ves_io_schema.Empty{}
				primaryOneOfSpec.Primary.SoaRecordParametersChoice = defaultSoaParametersSpec
			}

			// resource modelToSpecNested ArrayOfPointerField --> rr_set_group
			if len(data.DnsTypeChoice.Primary.RrSetGroup) > 0 {
				rrSetGroupSpec := make([]*ves_io_schema_dns_zone.RRSetGroup, len(data.DnsTypeChoice.Primary.RrSetGroup))
				primaryOneOfSpec.Primary.RrSetGroup = rrSetGroupSpec

				for i := range data.DnsTypeChoice.Primary.RrSetGroup {
					primaryOneOfSpec.Primary.RrSetGroup[i] = &ves_io_schema_dns_zone.RRSetGroup{}

					// resource modelToSpecNested RequiredPointerField --> Metadata
					metadataSpec := &ves_io_schema.MessageMetaType{}
					rrSetGroupSpec[i].Metadata = metadataSpec

					// resource modelToSpecNested RequiredStringField --> Name
					metadataSpec.Name = data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata.Name.ValueString()

					// resource modelToSpecNested StringField --> Description
					if !data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata.Description.IsNull() {
						metadataSpec.Description = data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata.Description.ValueString()
					}

					// resource modelToSpecNested ArrayOfPointerField --> rr_set
					if len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet) > 0 {
						rrSetSpec := make([]*ves_io_schema_dns_zone.RRSet, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet))
						rrSetGroupSpec[i].RrSet = rrSetSpec

						for ii := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet {
							rrSetGroupSpec[i].RrSet[ii] = &ves_io_schema_dns_zone.RRSet{}

							// resource modelToSpecNested UintField --> Ttl
							if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].Ttl.IsNull() {
								rrSetSpec[i].Ttl = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].Ttl.ValueInt32())
							}

							// resource modelToSpecNested RequiredOneOfField --> TypeRecordSet

							// resource modelToSpecNested PointerOfOneOfField --> NaptrRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord != nil {
								naptrRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_NaptrRecord{}
								naptrRecordOneOfSpec.NaptrRecord = &ves_io_schema_dns_zone.DNSNAPTRResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = naptrRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Name.IsNull() {
									naptrRecordOneOfSpec.NaptrRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.NAPTRValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values))
								naptrRecordOneOfSpec.NaptrRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values {
									naptrRecordOneOfSpec.NaptrRecord.Values[iii] = &ves_io_schema_dns_zone.NAPTRValue{}

									// resource modelToSpecNested RequiredUintField --> Order
									valuesSpec[i].Order = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Order.ValueInt32())

									// resource modelToSpecNested RequiredUintField --> Preference
									valuesSpec[i].Preference = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Preference.ValueInt32())

									// resource modelToSpecNested RequiredStringField --> Flags
									valuesSpec[i].Flags = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Flags.ValueString())

									// resource modelToSpecNested StringField --> Service
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Service.IsNull() {
										valuesSpec[i].Service = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Service.ValueString())
									}

									// resource modelToSpecNested StringField --> Regexp
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Regexp.IsNull() {
										valuesSpec[i].Regexp = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Regexp.ValueString())
									}

									// resource modelToSpecNested StringField --> Replacement
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Replacement.IsNull() {
										valuesSpec[i].Replacement = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Replacement.ValueString())
									}
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> SrvRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord != nil {
								srvRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_SrvRecord{}
								srvRecordOneOfSpec.SrvRecord = &ves_io_schema_dns_zone.DNSSRVResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = srvRecordOneOfSpec

								// resource modelToSpecNested RequiredStringField --> Name
								srvRecordOneOfSpec.SrvRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Name.ValueString())

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.SRVService, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values))
								srvRecordOneOfSpec.SrvRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values {
									srvRecordOneOfSpec.SrvRecord.Values[iii] = &ves_io_schema_dns_zone.SRVService{}

									// resource modelToSpecNested UintField --> Priority
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Priority.IsNull() {
										valuesSpec[i].Priority = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Priority.ValueInt32())
									}

									// resource modelToSpecNested UintField --> Weight
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Weight.IsNull() {
										valuesSpec[i].Weight = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Weight.ValueInt32())
									}

									// resource modelToSpecNested UintField --> Port
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Port.IsNull() {
										valuesSpec[i].Port = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Port.ValueInt32())
									}

									// resource modelToSpecNested StringField --> Target
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Target.IsNull() {
										valuesSpec[i].Target = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Target.ValueString())
									}
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> TlsaRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord != nil {
								tlsaRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_TlsaRecord{}
								tlsaRecordOneOfSpec.TlsaRecord = &ves_io_schema_dns_zone.TLSAResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = tlsaRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Name.IsNull() {
									tlsaRecordOneOfSpec.TlsaRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.TLSARecordValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values))
								tlsaRecordOneOfSpec.TlsaRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values {
									tlsaRecordOneOfSpec.TlsaRecord.Values[iii] = &ves_io_schema_dns_zone.TLSARecordValue{}

									// resource modelToSpecNested RequiredEnumField --> CertificateUsage
									certificateUsageElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].CertificateUsage.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].CertificateUsage.ElementsAs(ctx, &certificateUsageElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].CertificateUsage = ves_io_schema_dns_zone.TLSARecordCertificateUsage(ves_io_schema_dns_zone.TLSARecordCertificateUsage_value[certificateUsageElements[0].ValueString()])

									// resource modelToSpecNested RequiredEnumField --> Selector
									selectorElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].Selector.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].Selector.ElementsAs(ctx, &selectorElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].Selector = ves_io_schema_dns_zone.TLSARecordCSelector(ves_io_schema_dns_zone.TLSARecordCSelector_value[selectorElements[0].ValueString()])

									// resource modelToSpecNested RequiredEnumField --> MatchingType
									matchingTypeElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].MatchingType.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].MatchingType.ElementsAs(ctx, &matchingTypeElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].MatchingType = ves_io_schema_dns_zone.TLSARecordMatchingType(ves_io_schema_dns_zone.TLSARecordMatchingType_value[matchingTypeElements[0].ValueString()])

									// resource modelToSpecNested RequiredStringField --> CertificateAssociationData
									valuesSpec[i].CertificateAssociationData = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].CertificateAssociationData.ValueString())
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> Eui48Record
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record != nil {
								eui48RecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_Eui48Record{}
								eui48RecordOneOfSpec.Eui48Record = &ves_io_schema_dns_zone.DNSEUI48ResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = eui48RecordOneOfSpec

								// resource modelToSpecNested RequiredStringField --> Value
								eui48RecordOneOfSpec.Eui48Record.Value = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record.Value.ValueString())

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record.Name.IsNull() {
									eui48RecordOneOfSpec.Eui48Record.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record.Name.ValueString())
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> Eui64Record
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record != nil {
								eui64RecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_Eui64Record{}
								eui64RecordOneOfSpec.Eui64Record = &ves_io_schema_dns_zone.DNSEUI64ResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = eui64RecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record.Name.IsNull() {
									eui64RecordOneOfSpec.Eui64Record.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record.Name.ValueString())
								}

								// resource modelToSpecNested RequiredStringField --> Value
								eui64RecordOneOfSpec.Eui64Record.Value = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record.Value.ValueString())
							}

							// resource modelToSpecNested PointerOfOneOfField --> CdsRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord != nil {
								cdsRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_CdsRecord{}
								cdsRecordOneOfSpec.CdsRecord = &ves_io_schema_dns_zone.DNSCDSRecord{}
								rrSetSpec[ii].TypeRecordSet = cdsRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Name.IsNull() {
									cdsRecordOneOfSpec.CdsRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.DSRecordValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values))
								cdsRecordOneOfSpec.CdsRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values {
									cdsRecordOneOfSpec.CdsRecord.Values[iii] = &ves_io_schema_dns_zone.DSRecordValue{}

									// resource modelToSpecNested RequiredUintField --> KeyTag
									valuesSpec[i].KeyTag = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].KeyTag.ValueInt32())

									// resource modelToSpecNested RequiredEnumField --> DsKeyAlgorithm
									dsKeyAlgorithmElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DsKeyAlgorithm.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DsKeyAlgorithm.ElementsAs(ctx, &dsKeyAlgorithmElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].DsKeyAlgorithm = ves_io_schema_dns_zone.DSKeyAlgorithm(ves_io_schema_dns_zone.DSKeyAlgorithm_value[dsKeyAlgorithmElements[0].ValueString()])

									// resource modelToSpecNested RequiredOneOfField --> DigestChoice

									// resource modelToSpecNested PointerOfOneOfField --> Sha1Digest
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha1Digest != nil {
										sha1DigestOneOfSpec := &ves_io_schema_dns_zone.DSRecordValue_Sha1Digest{}
										sha1DigestOneOfSpec.Sha1Digest = &ves_io_schema_dns_zone.SHA1Digest{}
										valuesSpec[iii].DigestChoice = sha1DigestOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Digest
										sha1DigestOneOfSpec.Sha1Digest.Digest = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha1Digest.Digest.ValueString())
									}

									// resource modelToSpecNested PointerOfOneOfField --> Sha256Digest
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha256Digest != nil {
										sha256DigestOneOfSpec := &ves_io_schema_dns_zone.DSRecordValue_Sha256Digest{}
										sha256DigestOneOfSpec.Sha256Digest = &ves_io_schema_dns_zone.SHA256Digest{}
										valuesSpec[iii].DigestChoice = sha256DigestOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Digest
										sha256DigestOneOfSpec.Sha256Digest.Digest = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha256Digest.Digest.ValueString())
									}

									// resource modelToSpecNested PointerOfOneOfField --> Sha384Digest
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha384Digest != nil {
										sha384DigestOneOfSpec := &ves_io_schema_dns_zone.DSRecordValue_Sha384Digest{}
										sha384DigestOneOfSpec.Sha384Digest = &ves_io_schema_dns_zone.SHA384Digest{}
										valuesSpec[iii].DigestChoice = sha384DigestOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Digest
										sha384DigestOneOfSpec.Sha384Digest.Digest = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha384Digest.Digest.ValueString())
									}
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> CaaRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord != nil {
								caaRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_CaaRecord{}
								caaRecordOneOfSpec.CaaRecord = &ves_io_schema_dns_zone.DNSCAAResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = caaRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Name.IsNull() {
									caaRecordOneOfSpec.CaaRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Name.ValueString())
								}

								// resource modelToSpecNested ArrayOfPointerField --> values
								if len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values) > 0 {
									valuesSpec := make([]*ves_io_schema_dns_zone.CertificationAuthorityAuthorization, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values))
									caaRecordOneOfSpec.CaaRecord.Values = valuesSpec

									for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values {
										caaRecordOneOfSpec.CaaRecord.Values[iii] = &ves_io_schema_dns_zone.CertificationAuthorityAuthorization{}

										// resource modelToSpecNested StringField --> Tag
										if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Tag.IsNull() {
											valuesSpec[i].Tag = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Tag.ValueString())
										}

										// resource modelToSpecNested StringField --> Value
										if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Value.IsNull() {
											valuesSpec[i].Value = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Value.ValueString())
										}

										// resource modelToSpecNested UintField --> Flags
										if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Flags.IsNull() {
											valuesSpec[i].Flags = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Flags.ValueInt32())
										}
									}
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> NsRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord != nil {
								nsRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_NsRecord{}
								nsRecordOneOfSpec.NsRecord = &ves_io_schema_dns_zone.DNSNSResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = nsRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord.Name.IsNull() {
									nsRecordOneOfSpec.NsRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfStringField --> Values
								elements := make([]string, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord.Values.Elements()))
								diags := data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord.Values.ElementsAs(ctx, &elements, false)
								response.Diagnostics.Append(diags...)
								nsRecordOneOfSpec.NsRecord.Values = elements
							}

							// resource modelToSpecNested PointerOfOneOfField --> AfsdbRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord != nil {
								afsdbRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_AfsdbRecord{}
								afsdbRecordOneOfSpec.AfsdbRecord = &ves_io_schema_dns_zone.DNSAFSDBRecord{}
								rrSetSpec[ii].TypeRecordSet = afsdbRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Name.IsNull() {
									afsdbRecordOneOfSpec.AfsdbRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.AFSDBRecordValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values))
								afsdbRecordOneOfSpec.AfsdbRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values {
									afsdbRecordOneOfSpec.AfsdbRecord.Values[iii] = &ves_io_schema_dns_zone.AFSDBRecordValue{}

									// resource modelToSpecNested RequiredEnumField --> Subtype
									subtypeElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values[iii].Subtype.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values[iii].Subtype.ElementsAs(ctx, &subtypeElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].Subtype = ves_io_schema_dns_zone.AFSDBRecordSubtype(ves_io_schema_dns_zone.AFSDBRecordSubtype_value[subtypeElements[0].ValueString()])

									// resource modelToSpecNested RequiredStringField --> Hostname
									valuesSpec[i].Hostname = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values[iii].Hostname.ValueString())
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> TxtRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord != nil {
								txtRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_TxtRecord{}
								txtRecordOneOfSpec.TxtRecord = &ves_io_schema_dns_zone.DNSTXTResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = txtRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord.Name.IsNull() {
									txtRecordOneOfSpec.TxtRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfStringField --> Values
								elements := make([]string, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord.Values.Elements()))
								diags := data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord.Values.ElementsAs(ctx, &elements, false)
								response.Diagnostics.Append(diags...)
								txtRecordOneOfSpec.TxtRecord.Values = elements
							}

							// resource modelToSpecNested PointerOfOneOfField --> ARecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord != nil {
								aRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_ARecord{}
								aRecordOneOfSpec.ARecord = &ves_io_schema_dns_zone.DNSAResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = aRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord.Name.IsNull() {
									aRecordOneOfSpec.ARecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfStringField --> Values
								elements := make([]string, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord.Values.Elements()))
								diags := data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord.Values.ElementsAs(ctx, &elements, false)
								response.Diagnostics.Append(diags...)
								aRecordOneOfSpec.ARecord.Values = elements
							}

							// resource modelToSpecNested PointerOfOneOfField --> AliasRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AliasRecord != nil {
								aliasRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_AliasRecord{}
								aliasRecordOneOfSpec.AliasRecord = &ves_io_schema_dns_zone.DNSAliasResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = aliasRecordOneOfSpec

								// resource modelToSpecNested StringField --> Value
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AliasRecord.Value.IsNull() {
									aliasRecordOneOfSpec.AliasRecord.Value = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AliasRecord.Value.ValueString())
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> LbRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord != nil {
								lbRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_LbRecord{}
								lbRecordOneOfSpec.LbRecord = &ves_io_schema_dns_zone.DNSLBResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = lbRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Name.IsNull() {
									lbRecordOneOfSpec.LbRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Name.ValueString())
								}

								// resource modelToSpecNested PointerField --> Value
								if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value != nil {
									valueSpec := &ves_io_schema_views.ObjectRefType{}
									lbRecordOneOfSpec.LbRecord.Value = valueSpec

									// resource modelToSpecNested StringField --> Tenant
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Tenant.IsNull() {
										valueSpec.Tenant = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Tenant.ValueString())
									}

									// resource modelToSpecNested StringField --> Namespace
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Namespace.IsNull() {
										valueSpec.Namespace = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Namespace.ValueString())
									}

									// resource modelToSpecNested StringField --> Name
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Name.IsNull() {
										valueSpec.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Name.ValueString())
									}
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> SshfpRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord != nil {
								sshfpRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_SshfpRecord{}
								sshfpRecordOneOfSpec.SshfpRecord = &ves_io_schema_dns_zone.SSHFPResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = sshfpRecordOneOfSpec

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.SSHFPRecordValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values))
								sshfpRecordOneOfSpec.SshfpRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values {
									sshfpRecordOneOfSpec.SshfpRecord.Values[iii] = &ves_io_schema_dns_zone.SSHFPRecordValue{}

									// resource modelToSpecNested RequiredEnumField --> Algorithm
									algorithmElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].Algorithm.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].Algorithm.ElementsAs(ctx, &algorithmElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].Algorithm = ves_io_schema_dns_zone.SSHFPAlgorithm(ves_io_schema_dns_zone.SSHFPAlgorithm_value[algorithmElements[0].ValueString()])

									// resource modelToSpecNested RequiredOneOfField --> FingerprintType

									// resource modelToSpecNested PointerOfOneOfField --> Sha1Fingerprint
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha1Fingerprint != nil {
										sha1FingerprintOneOfSpec := &ves_io_schema_dns_zone.SSHFPRecordValue_Sha1Fingerprint{}
										sha1FingerprintOneOfSpec.Sha1Fingerprint = &ves_io_schema_dns_zone.SHA1Fingerprint{}
										valuesSpec[iii].FingerprintType = sha1FingerprintOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Fingerprint
										sha1FingerprintOneOfSpec.Sha1Fingerprint.Fingerprint = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha1Fingerprint.Fingerprint.ValueString())
									}

									// resource modelToSpecNested PointerOfOneOfField --> Sha256Fingerprint
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha256Fingerprint != nil {
										sha256FingerprintOneOfSpec := &ves_io_schema_dns_zone.SSHFPRecordValue_Sha256Fingerprint{}
										sha256FingerprintOneOfSpec.Sha256Fingerprint = &ves_io_schema_dns_zone.SHA256Fingerprint{}
										valuesSpec[iii].FingerprintType = sha256FingerprintOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Fingerprint
										sha256FingerprintOneOfSpec.Sha256Fingerprint.Fingerprint = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha256Fingerprint.Fingerprint.ValueString())
									}
								}

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Name.IsNull() {
									sshfpRecordOneOfSpec.SshfpRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Name.ValueString())
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> CertRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord != nil {
								certRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_CertRecord{}
								certRecordOneOfSpec.CertRecord = &ves_io_schema_dns_zone.CERTResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = certRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Name.IsNull() {
									certRecordOneOfSpec.CertRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.CERTRecordValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values))
								certRecordOneOfSpec.CertRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values {
									certRecordOneOfSpec.CertRecord.Values[iii] = &ves_io_schema_dns_zone.CERTRecordValue{}

									// resource modelToSpecNested RequiredUintField --> CertKeyTag
									valuesSpec[i].CertKeyTag = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].CertKeyTag.ValueInt32())

									// resource modelToSpecNested RequiredEnumField --> Algorithm
									algorithmElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].Algorithm.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].Algorithm.ElementsAs(ctx, &algorithmElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].Algorithm = ves_io_schema_dns_zone.CERTAlgorithm(ves_io_schema_dns_zone.CERTAlgorithm_value[algorithmElements[0].ValueString()])

									// resource modelToSpecNested RequiredStringField --> Certificate
									valuesSpec[i].Certificate = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].Certificate.ValueString())

									// resource modelToSpecNested RequiredEnumField --> CertType
									certTypeElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].CertType.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].CertType.ElementsAs(ctx, &certTypeElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].CertType = ves_io_schema_dns_zone.CERTType(ves_io_schema_dns_zone.CERTType_value[certTypeElements[0].ValueString()])
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> CnameRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord != nil {
								cnameRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_CnameRecord{}
								cnameRecordOneOfSpec.CnameRecord = &ves_io_schema_dns_zone.DNSCNAMEResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = cnameRecordOneOfSpec

								// resource modelToSpecNested StringField --> Value
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord.Value.IsNull() {
									cnameRecordOneOfSpec.CnameRecord.Value = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord.Value.ValueString())
								}

								// resource modelToSpecNested RequiredStringField --> Name
								cnameRecordOneOfSpec.CnameRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord.Name.ValueString())
							}

							// resource modelToSpecNested PointerOfOneOfField --> PtrRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord != nil {
								ptrRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_PtrRecord{}
								ptrRecordOneOfSpec.PtrRecord = &ves_io_schema_dns_zone.DNSPTRResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = ptrRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord.Name.IsNull() {
									ptrRecordOneOfSpec.PtrRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfStringField --> Values
								elements := make([]string, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord.Values.Elements()))
								diags := data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord.Values.ElementsAs(ctx, &elements, false)
								response.Diagnostics.Append(diags...)
								ptrRecordOneOfSpec.PtrRecord.Values = elements
							}

							// resource modelToSpecNested PointerOfOneOfField --> AaaaRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord != nil {
								aaaaRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_AaaaRecord{}
								aaaaRecordOneOfSpec.AaaaRecord = &ves_io_schema_dns_zone.DNSAAAAResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = aaaaRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord.Name.IsNull() {
									aaaaRecordOneOfSpec.AaaaRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfStringField --> Values
								elements := make([]string, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord.Values.Elements()))
								diags := data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord.Values.ElementsAs(ctx, &elements, false)
								response.Diagnostics.Append(diags...)
								aaaaRecordOneOfSpec.AaaaRecord.Values = elements
							}

							// resource modelToSpecNested PointerOfOneOfField --> LocRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord != nil {
								locRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_LocRecord{}
								locRecordOneOfSpec.LocRecord = &ves_io_schema_dns_zone.DNSLOCResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = locRecordOneOfSpec

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.LOCValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values))
								locRecordOneOfSpec.LocRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values {
									locRecordOneOfSpec.LocRecord.Values[iii] = &ves_io_schema_dns_zone.LOCValue{}

									// resource modelToSpecNested FloatField --> LatitudeSecond
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeSecond.IsNull() {
										valuesSpec[i].LatitudeSecond = float32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeSecond.ValueFloat32())
									}

									// resource modelToSpecNested RequiredIntField --> LongitudeDegree
									valuesSpec[i].LongitudeDegree = int32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeDegree.ValueInt32())

									// resource modelToSpecNested IntField --> LongitudeMinute
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeMinute.IsNull() {
										valuesSpec[i].LongitudeMinute = int32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeMinute.ValueInt32())
									}

									// resource modelToSpecNested FloatField --> LongitudeSecond
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeSecond.IsNull() {
										valuesSpec[i].LongitudeSecond = float32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeSecond.ValueFloat32())
									}

									// resource modelToSpecNested FloatField --> HorizontalPrecision
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].HorizontalPrecision.IsNull() {
										valuesSpec[i].HorizontalPrecision = float32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].HorizontalPrecision.ValueFloat32())
									}

									// resource modelToSpecNested RequiredIntField --> LatitudeDegree
									valuesSpec[i].LatitudeDegree = int32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeDegree.ValueInt32())

									// resource modelToSpecNested RequiredEnumField --> LatitudeHemisphere
									latitudeHemisphereElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeHemisphere.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeHemisphere.ElementsAs(ctx, &latitudeHemisphereElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].LatitudeHemisphere = ves_io_schema_dns_zone.LatitudeHemisphere(ves_io_schema_dns_zone.LatitudeHemisphere_value[latitudeHemisphereElements[0].ValueString()])

									// resource modelToSpecNested RequiredEnumField --> LongitudeHemisphere
									longitudeHemisphereElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeHemisphere.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeHemisphere.ElementsAs(ctx, &longitudeHemisphereElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].LongitudeHemisphere = ves_io_schema_dns_zone.LongitudeHemisphere(ves_io_schema_dns_zone.LongitudeHemisphere_value[longitudeHemisphereElements[0].ValueString()])

									// resource modelToSpecNested FloatField --> LocationDiameter
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LocationDiameter.IsNull() {
										valuesSpec[i].LocationDiameter = float32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LocationDiameter.ValueFloat32())
									}

									// resource modelToSpecNested FloatField --> VerticalPrecision
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].VerticalPrecision.IsNull() {
										valuesSpec[i].VerticalPrecision = float32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].VerticalPrecision.ValueFloat32())
									}

									// resource modelToSpecNested IntField --> LatitudeMinute
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeMinute.IsNull() {
										valuesSpec[i].LatitudeMinute = int32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeMinute.ValueInt32())
									}
								}

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Name.IsNull() {
									locRecordOneOfSpec.LocRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Name.ValueString())
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> DsRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord != nil {
								dsRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_DsRecord{}
								dsRecordOneOfSpec.DsRecord = &ves_io_schema_dns_zone.DNSDSRecord{}
								rrSetSpec[ii].TypeRecordSet = dsRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Name.IsNull() {
									dsRecordOneOfSpec.DsRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.DSRecordValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values))
								dsRecordOneOfSpec.DsRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values {
									dsRecordOneOfSpec.DsRecord.Values[iii] = &ves_io_schema_dns_zone.DSRecordValue{}

									// resource modelToSpecNested RequiredUintField --> KeyTag
									valuesSpec[i].KeyTag = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].KeyTag.ValueInt32())

									// resource modelToSpecNested RequiredEnumField --> DsKeyAlgorithm
									dsKeyAlgorithmElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DsKeyAlgorithm.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DsKeyAlgorithm.ElementsAs(ctx, &dsKeyAlgorithmElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].DsKeyAlgorithm = ves_io_schema_dns_zone.DSKeyAlgorithm(ves_io_schema_dns_zone.DSKeyAlgorithm_value[dsKeyAlgorithmElements[0].ValueString()])

									// resource modelToSpecNested RequiredOneOfField --> DigestChoice

									// resource modelToSpecNested PointerOfOneOfField --> Sha1Digest
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha1Digest != nil {
										sha1DigestOneOfSpec := &ves_io_schema_dns_zone.DSRecordValue_Sha1Digest{}
										sha1DigestOneOfSpec.Sha1Digest = &ves_io_schema_dns_zone.SHA1Digest{}
										valuesSpec[iii].DigestChoice = sha1DigestOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Digest
										sha1DigestOneOfSpec.Sha1Digest.Digest = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha1Digest.Digest.ValueString())
									}

									// resource modelToSpecNested PointerOfOneOfField --> Sha256Digest
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha256Digest != nil {
										sha256DigestOneOfSpec := &ves_io_schema_dns_zone.DSRecordValue_Sha256Digest{}
										sha256DigestOneOfSpec.Sha256Digest = &ves_io_schema_dns_zone.SHA256Digest{}
										valuesSpec[iii].DigestChoice = sha256DigestOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Digest
										sha256DigestOneOfSpec.Sha256Digest.Digest = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha256Digest.Digest.ValueString())
									}

									// resource modelToSpecNested PointerOfOneOfField --> Sha384Digest
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha384Digest != nil {
										sha384DigestOneOfSpec := &ves_io_schema_dns_zone.DSRecordValue_Sha384Digest{}
										sha384DigestOneOfSpec.Sha384Digest = &ves_io_schema_dns_zone.SHA384Digest{}
										valuesSpec[iii].DigestChoice = sha384DigestOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Digest
										sha384DigestOneOfSpec.Sha384Digest.Digest = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha384Digest.Digest.ValueString())
									}
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> MxRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord != nil {
								mxRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_MxRecord{}
								mxRecordOneOfSpec.MxRecord = &ves_io_schema_dns_zone.DNSMXResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = mxRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Name.IsNull() {
									mxRecordOneOfSpec.MxRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.MailExchanger, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values))
								mxRecordOneOfSpec.MxRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values {
									mxRecordOneOfSpec.MxRecord.Values[iii] = &ves_io_schema_dns_zone.MailExchanger{}

									// resource modelToSpecNested StringField --> Domain
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values[iii].Domain.IsNull() {
										valuesSpec[i].Domain = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values[iii].Domain.ValueString())
									}

									// resource modelToSpecNested UintField --> Priority
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values[iii].Priority.IsNull() {
										valuesSpec[i].Priority = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values[iii].Priority.ValueInt32())
									}
								}
							}

							// resource modelToSpecNested StringField --> Description
							if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].Description.IsNull() {
								rrSetSpec[i].Description = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].Description.ValueString())
							}
						}
					}
				}
			}

		}
	}

	if response.Diagnostics.HasError() {
		return
	}

	createRspType := strings.Replace(ves_io_schema_dns_zone.ObjectType, "Object", "GetResponse", 1)
	protoMsgType := proto.MessageType(createRspType).Elem()
	protoMsg := reflect.New(protoMsgType).Interface().(proto.Message)
	callRsp := &server.CallResponse{
		ProtoMsg: protoMsg,
	}

	opts := []vesapi.CallOpt{
		vesapi.WithOutCallResponse(callRsp),
	}

	httpResp, err := r.client.CreateObject(ctx, ves_io_schema_dns_zone.ObjectType, req, opts...)
	if err != nil {
		response.Diagnostics.AddError(
			"Client Error", fmt.Sprintf("Unable to create %s, got error: %s", "dns_zone", err),
		)
		return
	}

	metaData := httpResp.GetObjMetadata()
	specData := httpResp.GetObjSpec().(*ves_io_schema_dns_zone.SpecType).GcSpec
	systemMetaData := httpResp.GetObjSystemMetadata()

	tflog.Info(ctx, fmt.Sprintf("meta_type: %v", metaData))
	tflog.Info(ctx, fmt.Sprintf("spec_type: %v", specData))
	tflog.Info(ctx, fmt.Sprintf("meta_system_type_uid_field: %v", systemMetaData.GetUid()))

	// Write back the response to the model

	data.Name = types.StringValue(metaData.GetName())

	data.Namespace = types.StringValue(metaData.GetNamespace())

	data.Uid = types.StringValue(systemMetaData.GetUid())
	labelsValue, diags := types.MapValueFrom(ctx, types.StringType, metaData.GetLabels())
	data.Labels = labelsValue
	response.Diagnostics.Append(diags...)
	annotationsValue, diags := types.MapValueFrom(ctx, types.StringType, metaData.GetAnnotations())
	data.Annotations = annotationsValue
	response.Diagnostics.Append(diags...)

	data.Description = types.StringValue(metaData.GetDescription())
	// resource SpecToModelRoot StringField --> ServiceInstanceId
	if specData.GetServiceInstanceId() != "" {
		data.ServiceInstanceId = types.StringValue(specData.GetServiceInstanceId())
	}

	// resource SpecToModelRoot UintField --> NumOfDnsRecords
	numOfDnsRecordsValue := specData.GetNumOfDnsRecords()
	if numOfDnsRecordsValue > 0 {
	}

	// resource SpecToModelRoot RequiredOneOfField --> DnsType
	dnsTypeChoiceChoices := specData.GetDnsType()
	data.DnsTypeChoice = &GlobalSpecTypeDnsType{}

	// resource specToModelNested PointerOfOneOfField --> Secondary
	secondaryOptions, isSecondary := dnsTypeChoiceChoices.(*ves_io_schema_dns_zone.GlobalSpecType_Secondary)
	tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", secondaryOptions))
	if isSecondary {
		data.DnsTypeChoice.Secondary = &GlobalSpecTypeSecondary{}

		// resource specToModelNested StringField --> TsigKeyName
		if secondaryOptions.Secondary.GetTsigKeyName() != "" {
			data.DnsTypeChoice.Secondary.TsigKeyName = types.StringValue(secondaryOptions.Secondary.GetTsigKeyName())
		}
		// resource specToModelNested EnumField --> TsigKeyAlgorithm
		if secondaryOptions.Secondary.GetTsigKeyAlgorithm().String() != "" {
			stringValue := types.StringValue(secondaryOptions.Secondary.GetTsigKeyAlgorithm().String())
			stringSlice := []types.String{stringValue}
			listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
			response.Diagnostics.Append(diags...)
			data.DnsTypeChoice.Secondary.TsigKeyAlgorithm = listValue
		}
		// resource specToModelNested PointerField --> TsigKeyValue
		if secondaryOptions.Secondary.GetTsigKeyValue() != nil {
			data.DnsTypeChoice.Secondary.TsigKeyValue = &SecretType{}

			// resource specToModelNested RequiredOneOfField --> SecretInfoOneof
			secretInfoChoiceChoices := secondaryOptions.Secondary.GetTsigKeyValue().GetSecretInfoOneof()
			data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice = &SecretTypeSecretInfoOneof{}

			// resource specToModelNested PointerOfOneOfField --> BlindfoldSecretInfo
			blindfoldSecretInfoOptions, isBlindfoldSecretInfo := secretInfoChoiceChoices.(*ves_io_schema.SecretType_BlindfoldSecretInfo)
			tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", blindfoldSecretInfoOptions))
			if isBlindfoldSecretInfo {
				data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo = &SecretTypeBlindfoldSecretInfo{}

				// resource specToModelNested StringField --> DecryptionProvider
				if blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetDecryptionProvider() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.DecryptionProvider = types.StringValue(blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetDecryptionProvider())
				}
				// resource specToModelNested StringField --> StoreProvider
				if blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetStoreProvider() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.StoreProvider = types.StringValue(blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetStoreProvider())
				}
				// resource specToModelNested RequiredStringField --> Location
				if blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetLocation() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.Location = types.StringValue(blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetLocation())
				}
			}
			// resource specToModelNested PointerOfOneOfField --> ClearSecretInfo
			clearSecretInfoOptions, isClearSecretInfo := secretInfoChoiceChoices.(*ves_io_schema.SecretType_ClearSecretInfo)
			tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", clearSecretInfoOptions))
			if isClearSecretInfo {
				data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo = &SecretTypeClearSecretInfo{}

				// resource specToModelNested StringField --> Provider
				if clearSecretInfoOptions.ClearSecretInfo.GetProvider() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo.Provider = types.StringValue(clearSecretInfoOptions.ClearSecretInfo.GetProvider())
				}
				// resource specToModelNested RequiredStringField --> Url
				if clearSecretInfoOptions.ClearSecretInfo.GetUrl() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo.Url = types.StringValue(clearSecretInfoOptions.ClearSecretInfo.GetUrl())
				}
			}
		}
		// resource specToModelNested TimeStampField --> LastAxfrTimestamp
		tsAsString := secondaryOptions.Secondary.GetLastAxfrTimestamp().String()
		tfTimeValue, diags := timetypes.NewRFC3339Value(tsAsString)
		response.Diagnostics.Append(diags...)
		data.DnsTypeChoice.Secondary.LastAxfrTimestamp = tfTimeValue // resource specToModelNested RequiredArrayOfStringField --> PrimaryServers
		if len(secondaryOptions.Secondary.GetPrimaryServers()) > 0 {
			listValue, diags := types.ListValueFrom(ctx, types.StringType, secondaryOptions.Secondary.GetPrimaryServers())
			response.Diagnostics.Append(diags...)
			data.DnsTypeChoice.Secondary.PrimaryServers = listValue
		}
	}
	// resource specToModelNested PointerOfOneOfField --> Primary
	primaryOptions, isPrimary := dnsTypeChoiceChoices.(*ves_io_schema_dns_zone.GlobalSpecType_Primary)
	tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", primaryOptions))
	if isPrimary {
		data.DnsTypeChoice.Primary = &GlobalSpecTypePrimary{}

		// resource specToModelNested StringField --> Admin
		if primaryOptions.Primary.GetAdmin() != "" {
			data.DnsTypeChoice.Primary.Admin = types.StringValue(primaryOptions.Primary.GetAdmin())
		}

		// resource specToModelNested ArrayOfPointerField --> DefaultRrSetGroup
		var defaultRrSetGroup = make([]DefaultRrSetGroupModel, len(primaryOptions.Primary.GetDefaultRrSetGroup()))
		a := primaryOptions.Primary.GetDefaultRrSetGroup()
		tflog.Info(ctx, fmt.Sprintf("DefaultRrSetGroup length: %v", len(a)))

		for i, elemnet := range primaryOptions.Primary.GetDefaultRrSetGroup() {
			var ns NsRecordModel
			tflog.Info(ctx, fmt.Sprintf("DefaultRrSetGroup Element: %v", primaryOptions.Primary.GetDefaultRrSetGroup()[i]))
			defaultRrSetGroup[i].Ttl = types.Int64Value(int64(elemnet.GetTtl()))
			defaultRrSetGroup[i].Description = types.StringValue(elemnet.GetDescription())
			ns.Name = types.StringValue(elemnet.GetNsRecord().GetName())
			listValue, diags := types.ListValueFrom(ctx, types.StringType, elemnet.GetNsRecord().GetValues())
			response.Diagnostics.Append(diags...)
			ns.Values = listValue
			nsObject, d := types.ObjectValueFrom(ctx,
				nsRecordObjectType.AttrTypes, // ⬅️ Pass the map[string]attr.Type here
				ns,
			)
			diags.Append(d...)
			defaultRrSetGroup[i].NsRecord = nsObject
		}

		response.Diagnostics.Append(diags...)
		if response.Diagnostics.HasError() {
			return
		}

		rrSetGroupObjects, d := types.ListValueFrom(
			ctx,
			defaultRrSetGroupObjectType, // Global type definition for the list element
			defaultRrSetGroup,           // The fully populated slice
		)

		response.Diagnostics.Append(d...)
		if response.Diagnostics.HasError() {
			return
		}

		data.DnsTypeChoice.Primary.DefaultRrSetGroup = rrSetGroupObjects

		// resource specToModelNested PointerField --> DnssecMode
		if primaryOptions.Primary.GetDnssecMode() != nil {
			data.DnsTypeChoice.Primary.DnssecMode = &DnsZoneDNSSECMode{}

			// resource specToModelNested RequiredOneOfField --> Mode
			modeChoiceChoices := primaryOptions.Primary.GetDnssecMode().GetMode()
			data.DnsTypeChoice.Primary.DnssecMode.ModeChoice = &DNSSECModeMode{}

			// resource specToModelNested BoolOfOneOfPointerField --> Enable
			_, isEnable := modeChoiceChoices.(*ves_io_schema_dns_zone.DNSSECMode_Enable)
			if isEnable {
				data.DnsTypeChoice.Primary.DnssecMode.ModeChoice.Enable = types.BoolValue(true)
			}
		}
		// resource specToModelNested BoolField --> AllowHttpLbManagedRecords
		data.DnsTypeChoice.Primary.AllowHttpLbManagedRecords = types.BoolValue(bool(primaryOptions.Primary.GetAllowHttpLbManagedRecords()))
		// resource specToModelNested RequiredOneOfField --> SoaRecordParametersChoice
		soaRecordParametersChoiceChoices := primaryOptions.Primary.GetSoaRecordParametersChoice()
		data.DnsTypeChoice.Primary.SoaRecordParametersChoice = &PrimaryDNSConfigSoaRecordParametersChoice{}

		// resource specToModelNested PointerOfOneOfField --> SoaParameters
		soaParametersOptions, isSoaParameters := soaRecordParametersChoiceChoices.(*ves_io_schema_dns_zone.PrimaryDNSConfig_SoaParameters)
		tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", soaParametersOptions))
		if isSoaParameters {
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters = &PrimaryDNSConfigSoaParameters{}

			// resource specToModelNested UintField --> Refresh
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Refresh = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetRefresh()))
			// resource specToModelNested UintField --> Retry
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Retry = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetRetry()))
			// resource specToModelNested UintField --> Expire
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Expire = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetExpire()))
			// resource specToModelNested UintField --> NegativeTtl
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.NegativeTtl = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetNegativeTtl()))
			// resource specToModelNested UintField --> Ttl
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Ttl = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetTtl()))
		}
		// resource specToModelNested BoolOfOneOfPointerField --> DefaultSoaParameters
		_, isDefaultSoaParameters := soaRecordParametersChoiceChoices.(*ves_io_schema_dns_zone.PrimaryDNSConfig_DefaultSoaParameters)
		if isDefaultSoaParameters {
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.DefaultSoaParameters = types.BoolValue(true)
		}
		// resource specToModelNested ArrayOfPointerField --> RrSetGroup
		if len(primaryOptions.Primary.GetRrSetGroup()) > 0 {
			data.DnsTypeChoice.Primary.RrSetGroup = make([]DnsZoneRRSetGroup, len(primaryOptions.Primary.GetRrSetGroup()))
			for i := range primaryOptions.Primary.GetRrSetGroup() {

				// resource specToModelNested RequiredPointerField --> Metadata
				data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata = MessageMetaType{}

				// resource specToModelNested RequiredStringField --> Name
				if primaryOptions.Primary.GetRrSetGroup()[i].GetMetadata().GetName() != "" {
					data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata.Name = types.StringValue(primaryOptions.Primary.GetRrSetGroup()[i].GetMetadata().GetName())
				}
				// resource specToModelNested StringField --> Description
				if primaryOptions.Primary.GetRrSetGroup()[i].GetMetadata().GetDescription() != "" {
					data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata.Description = types.StringValue(primaryOptions.Primary.GetRrSetGroup()[i].GetMetadata().GetDescription())
				}
				// resource specToModelNested ArrayOfPointerField --> RrSet
				if len(primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()) > 0 {
					data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet = make([]DnsZoneRRSet, len(primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()))
					for ii := range primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet() {

						// resource specToModelNested UintField --> Ttl
						data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].Ttl = types.Int32Value(int32(primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()[ii].GetTtl()))
						// resource specToModelNested RequiredOneOfField --> TypeRecordSet
						typeRecordSetChoiceChoices := primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()[ii].GetTypeRecordSet()
						data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice = &RRSetTypeRecordSet{}

						// resource specToModelNested PointerOfOneOfField --> NaptrRecord
						naptrRecordOptions, isNaptrRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_NaptrRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", naptrRecordOptions))
						if isNaptrRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord = &RRSetNaptrRecord{}

							// resource specToModelNested StringField --> Name
							if naptrRecordOptions.NaptrRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Name = types.StringValue(naptrRecordOptions.NaptrRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range naptrRecordOptions.NaptrRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredUintField --> Order
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Order = types.Int32Value(int32(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetOrder()))
								// resource specToModelNested RequiredUintField --> Preference
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Preference = types.Int32Value(int32(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetPreference()))
								// resource specToModelNested RequiredStringField --> Flags
								if naptrRecordOptions.NaptrRecord.GetValues()[iii].GetFlags() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Flags = types.StringValue(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetFlags())
								}
								// resource specToModelNested StringField --> Service
								if naptrRecordOptions.NaptrRecord.GetValues()[iii].GetService() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Service = types.StringValue(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetService())
								}
								// resource specToModelNested StringField --> Regexp
								if naptrRecordOptions.NaptrRecord.GetValues()[iii].GetRegexp() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Regexp = types.StringValue(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetRegexp())
								}
								// resource specToModelNested StringField --> Replacement
								if naptrRecordOptions.NaptrRecord.GetValues()[iii].GetReplacement() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Replacement = types.StringValue(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetReplacement())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> SrvRecord
						srvRecordOptions, isSrvRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_SrvRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", srvRecordOptions))
						if isSrvRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord = &RRSetSrvRecord{}

							// resource specToModelNested RequiredStringField --> Name
							if srvRecordOptions.SrvRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Name = types.StringValue(srvRecordOptions.SrvRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range srvRecordOptions.SrvRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested UintField --> Priority
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Priority = types.Int32Value(int32(srvRecordOptions.SrvRecord.GetValues()[iii].GetPriority()))
								// resource specToModelNested UintField --> Weight
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Weight = types.Int32Value(int32(srvRecordOptions.SrvRecord.GetValues()[iii].GetWeight()))
								// resource specToModelNested UintField --> Port
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Port = types.Int32Value(int32(srvRecordOptions.SrvRecord.GetValues()[iii].GetPort()))
								// resource specToModelNested StringField --> Target
								if srvRecordOptions.SrvRecord.GetValues()[iii].GetTarget() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Target = types.StringValue(srvRecordOptions.SrvRecord.GetValues()[iii].GetTarget())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> TlsaRecord
						tlsaRecordOptions, isTlsaRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_TlsaRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", tlsaRecordOptions))
						if isTlsaRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord = &RRSetTlsaRecord{}

							// resource specToModelNested StringField --> Name
							if tlsaRecordOptions.TlsaRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Name = types.StringValue(tlsaRecordOptions.TlsaRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range tlsaRecordOptions.TlsaRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredEnumField --> CertificateUsage
								if tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetCertificateUsage().String() != "" {
									stringValue := types.StringValue(tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetCertificateUsage().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].CertificateUsage = listValue
								}
								// resource specToModelNested RequiredEnumField --> Selector
								if tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetSelector().String() != "" {
									stringValue := types.StringValue(tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetSelector().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].Selector = listValue
								}
								// resource specToModelNested RequiredEnumField --> MatchingType
								if tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetMatchingType().String() != "" {
									stringValue := types.StringValue(tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetMatchingType().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].MatchingType = listValue
								}
								// resource specToModelNested RequiredStringField --> CertificateAssociationData
								if tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetCertificateAssociationData() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].CertificateAssociationData = types.StringValue(tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetCertificateAssociationData())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> Eui48Record
						eui48RecordOptions, isEui48Record := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_Eui48Record)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", eui48RecordOptions))
						if isEui48Record {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record = &RRSetEui48Record{}

							// resource specToModelNested RequiredStringField --> Value
							if eui48RecordOptions.Eui48Record.GetValue() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record.Value = types.StringValue(eui48RecordOptions.Eui48Record.GetValue())
							}
							// resource specToModelNested StringField --> Name
							if eui48RecordOptions.Eui48Record.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record.Name = types.StringValue(eui48RecordOptions.Eui48Record.GetName())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> Eui64Record
						eui64RecordOptions, isEui64Record := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_Eui64Record)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", eui64RecordOptions))
						if isEui64Record {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record = &RRSetEui64Record{}

							// resource specToModelNested StringField --> Name
							if eui64RecordOptions.Eui64Record.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record.Name = types.StringValue(eui64RecordOptions.Eui64Record.GetName())
							}
							// resource specToModelNested RequiredStringField --> Value
							if eui64RecordOptions.Eui64Record.GetValue() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record.Value = types.StringValue(eui64RecordOptions.Eui64Record.GetValue())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> CdsRecord
						cdsRecordOptions, isCdsRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_CdsRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", cdsRecordOptions))
						if isCdsRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord = &RRSetCdsRecord{}

							// resource specToModelNested StringField --> Name
							if cdsRecordOptions.CdsRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Name = types.StringValue(cdsRecordOptions.CdsRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range cdsRecordOptions.CdsRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredUintField --> KeyTag
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].KeyTag = types.Int32Value(int32(cdsRecordOptions.CdsRecord.GetValues()[iii].GetKeyTag()))
								// resource specToModelNested RequiredEnumField --> DsKeyAlgorithm
								if cdsRecordOptions.CdsRecord.GetValues()[iii].GetDsKeyAlgorithm().String() != "" {
									stringValue := types.StringValue(cdsRecordOptions.CdsRecord.GetValues()[iii].GetDsKeyAlgorithm().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DsKeyAlgorithm = listValue
								}
								// resource specToModelNested RequiredOneOfField --> DigestChoice
								digestChoiceChoices := cdsRecordOptions.CdsRecord.GetValues()[iii].GetDigestChoice()
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice = &DSRecordValueDigestChoice{}

								// resource specToModelNested PointerOfOneOfField --> Sha1Digest
								sha1DigestOptions, isSha1Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha1Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha1DigestOptions))
								if isSha1Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha1Digest = &DSRecordValueSha1Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha1DigestOptions.Sha1Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha1Digest.Digest = types.StringValue(sha1DigestOptions.Sha1Digest.GetDigest())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha256Digest
								sha256DigestOptions, isSha256Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha256Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha256DigestOptions))
								if isSha256Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha256Digest = &DSRecordValueSha256Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha256DigestOptions.Sha256Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha256Digest.Digest = types.StringValue(sha256DigestOptions.Sha256Digest.GetDigest())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha384Digest
								sha384DigestOptions, isSha384Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha384Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha384DigestOptions))
								if isSha384Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha384Digest = &DSRecordValueSha384Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha384DigestOptions.Sha384Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha384Digest.Digest = types.StringValue(sha384DigestOptions.Sha384Digest.GetDigest())
									}
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> CaaRecord
						caaRecordOptions, isCaaRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_CaaRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", caaRecordOptions))
						if isCaaRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord = &RRSetCaaRecord{}

							// resource specToModelNested StringField --> Name
							if caaRecordOptions.CaaRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Name = types.StringValue(caaRecordOptions.CaaRecord.GetName())
							}
							// resource specToModelNested ArrayOfPointerField --> Values
							if len(caaRecordOptions.CaaRecord.GetValues()) > 0 {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values = make([]DnsZoneCertificationAuthorityAuthorization, len(caaRecordOptions.CaaRecord.GetValues()))
								for iii := range caaRecordOptions.CaaRecord.GetValues() {

									// resource specToModelNested StringField --> Tag
									if caaRecordOptions.CaaRecord.GetValues()[iii].GetTag() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Tag = types.StringValue(caaRecordOptions.CaaRecord.GetValues()[iii].GetTag())
									}
									// resource specToModelNested StringField --> Value
									if caaRecordOptions.CaaRecord.GetValues()[iii].GetValue() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Value = types.StringValue(caaRecordOptions.CaaRecord.GetValues()[iii].GetValue())
									}
									// resource specToModelNested UintField --> Flags
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Flags = types.Int32Value(int32(caaRecordOptions.CaaRecord.GetValues()[iii].GetFlags()))
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> NsRecord
						nsRecordOptions, isNsRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_NsRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", nsRecordOptions))
						if isNsRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord = &RRSetNsRecord{}

							// resource specToModelNested StringField --> Name
							if nsRecordOptions.NsRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord.Name = types.StringValue(nsRecordOptions.NsRecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(nsRecordOptions.NsRecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, nsRecordOptions.NsRecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> AfsdbRecord
						afsdbRecordOptions, isAfsdbRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_AfsdbRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", afsdbRecordOptions))
						if isAfsdbRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord = &RRSetAfsdbRecord{}

							// resource specToModelNested StringField --> Name
							if afsdbRecordOptions.AfsdbRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Name = types.StringValue(afsdbRecordOptions.AfsdbRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range afsdbRecordOptions.AfsdbRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredEnumField --> Subtype
								if afsdbRecordOptions.AfsdbRecord.GetValues()[iii].GetSubtype().String() != "" {
									stringValue := types.StringValue(afsdbRecordOptions.AfsdbRecord.GetValues()[iii].GetSubtype().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values[iii].Subtype = listValue
								}
								// resource specToModelNested RequiredStringField --> Hostname
								if afsdbRecordOptions.AfsdbRecord.GetValues()[iii].GetHostname() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values[iii].Hostname = types.StringValue(afsdbRecordOptions.AfsdbRecord.GetValues()[iii].GetHostname())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> TxtRecord
						txtRecordOptions, isTxtRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_TxtRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", txtRecordOptions))
						if isTxtRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord = &RRSetTxtRecord{}

							// resource specToModelNested StringField --> Name
							if txtRecordOptions.TxtRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord.Name = types.StringValue(txtRecordOptions.TxtRecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(txtRecordOptions.TxtRecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, txtRecordOptions.TxtRecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> ARecord
						aRecordOptions, isARecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_ARecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", aRecordOptions))
						if isARecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord = &RRSetARecord{}

							// resource specToModelNested StringField --> Name
							if aRecordOptions.ARecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord.Name = types.StringValue(aRecordOptions.ARecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(aRecordOptions.ARecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, aRecordOptions.ARecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> AliasRecord
						aliasRecordOptions, isAliasRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_AliasRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", aliasRecordOptions))
						if isAliasRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AliasRecord = &RRSetAliasRecord{}

							// resource specToModelNested StringField --> Value
							if aliasRecordOptions.AliasRecord.GetValue() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AliasRecord.Value = types.StringValue(aliasRecordOptions.AliasRecord.GetValue())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> LbRecord
						lbRecordOptions, isLbRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_LbRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", lbRecordOptions))
						if isLbRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord = &RRSetLbRecord{}

							// resource specToModelNested StringField --> Name
							if lbRecordOptions.LbRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Name = types.StringValue(lbRecordOptions.LbRecord.GetName())
							}
							// resource specToModelNested PointerField --> Value
							if lbRecordOptions.LbRecord.GetValue() != nil {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value = &ObjectRefTypeModel{}

								// resource specToModelNested StringField --> Tenant
								if lbRecordOptions.LbRecord.GetValue().GetTenant() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Tenant = types.StringValue(lbRecordOptions.LbRecord.GetValue().GetTenant())
								}
								// resource specToModelNested StringField --> Namespace
								if lbRecordOptions.LbRecord.GetValue().GetNamespace() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Namespace = types.StringValue(lbRecordOptions.LbRecord.GetValue().GetNamespace())
								}
								// resource specToModelNested StringField --> Name
								if lbRecordOptions.LbRecord.GetValue().GetName() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Name = types.StringValue(lbRecordOptions.LbRecord.GetValue().GetName())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> SshfpRecord
						sshfpRecordOptions, isSshfpRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_SshfpRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sshfpRecordOptions))
						if isSshfpRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord = &RRSetSshfpRecord{}

							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range sshfpRecordOptions.SshfpRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredEnumField --> Algorithm
								if sshfpRecordOptions.SshfpRecord.GetValues()[iii].GetAlgorithm().String() != "" {
									stringValue := types.StringValue(sshfpRecordOptions.SshfpRecord.GetValues()[iii].GetAlgorithm().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].Algorithm = listValue
								}
								// resource specToModelNested RequiredOneOfField --> FingerprintType
								fingerprintTypeChoiceChoices := sshfpRecordOptions.SshfpRecord.GetValues()[iii].GetFingerprintType()
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice = &SSHFPRecordValueFingerprintType{}

								// resource specToModelNested PointerOfOneOfField --> Sha1Fingerprint
								sha1FingerprintOptions, isSha1Fingerprint := fingerprintTypeChoiceChoices.(*ves_io_schema_dns_zone.SSHFPRecordValue_Sha1Fingerprint)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha1FingerprintOptions))
								if isSha1Fingerprint {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha1Fingerprint = &SSHFPRecordValueSha1Fingerprint{}

									// resource specToModelNested RequiredStringField --> Fingerprint
									if sha1FingerprintOptions.Sha1Fingerprint.GetFingerprint() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha1Fingerprint.Fingerprint = types.StringValue(sha1FingerprintOptions.Sha1Fingerprint.GetFingerprint())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha256Fingerprint
								sha256FingerprintOptions, isSha256Fingerprint := fingerprintTypeChoiceChoices.(*ves_io_schema_dns_zone.SSHFPRecordValue_Sha256Fingerprint)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha256FingerprintOptions))
								if isSha256Fingerprint {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha256Fingerprint = &SSHFPRecordValueSha256Fingerprint{}

									// resource specToModelNested RequiredStringField --> Fingerprint
									if sha256FingerprintOptions.Sha256Fingerprint.GetFingerprint() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha256Fingerprint.Fingerprint = types.StringValue(sha256FingerprintOptions.Sha256Fingerprint.GetFingerprint())
									}
								}
							}
							// resource specToModelNested StringField --> Name
							if sshfpRecordOptions.SshfpRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Name = types.StringValue(sshfpRecordOptions.SshfpRecord.GetName())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> CertRecord
						certRecordOptions, isCertRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_CertRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", certRecordOptions))
						if isCertRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord = &RRSetCertRecord{}

							// resource specToModelNested StringField --> Name
							if certRecordOptions.CertRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Name = types.StringValue(certRecordOptions.CertRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range certRecordOptions.CertRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredUintField --> CertKeyTag
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].CertKeyTag = types.Int32Value(int32(certRecordOptions.CertRecord.GetValues()[iii].GetCertKeyTag()))
								// resource specToModelNested RequiredEnumField --> Algorithm
								if certRecordOptions.CertRecord.GetValues()[iii].GetAlgorithm().String() != "" {
									stringValue := types.StringValue(certRecordOptions.CertRecord.GetValues()[iii].GetAlgorithm().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].Algorithm = listValue
								}
								// resource specToModelNested RequiredStringField --> Certificate
								if certRecordOptions.CertRecord.GetValues()[iii].GetCertificate() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].Certificate = types.StringValue(certRecordOptions.CertRecord.GetValues()[iii].GetCertificate())
								}
								// resource specToModelNested RequiredEnumField --> CertType
								if certRecordOptions.CertRecord.GetValues()[iii].GetCertType().String() != "" {
									stringValue := types.StringValue(certRecordOptions.CertRecord.GetValues()[iii].GetCertType().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].CertType = listValue
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> CnameRecord
						cnameRecordOptions, isCnameRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_CnameRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", cnameRecordOptions))
						if isCnameRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord = &RRSetCnameRecord{}

							// resource specToModelNested StringField --> Value
							if cnameRecordOptions.CnameRecord.GetValue() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord.Value = types.StringValue(cnameRecordOptions.CnameRecord.GetValue())
							}
							// resource specToModelNested RequiredStringField --> Name
							if cnameRecordOptions.CnameRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord.Name = types.StringValue(cnameRecordOptions.CnameRecord.GetName())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> PtrRecord
						ptrRecordOptions, isPtrRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_PtrRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", ptrRecordOptions))
						if isPtrRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord = &RRSetPtrRecord{}

							// resource specToModelNested StringField --> Name
							if ptrRecordOptions.PtrRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord.Name = types.StringValue(ptrRecordOptions.PtrRecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(ptrRecordOptions.PtrRecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, ptrRecordOptions.PtrRecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> AaaaRecord
						aaaaRecordOptions, isAaaaRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_AaaaRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", aaaaRecordOptions))
						if isAaaaRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord = &RRSetAaaaRecord{}

							// resource specToModelNested StringField --> Name
							if aaaaRecordOptions.AaaaRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord.Name = types.StringValue(aaaaRecordOptions.AaaaRecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(aaaaRecordOptions.AaaaRecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, aaaaRecordOptions.AaaaRecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> LocRecord
						locRecordOptions, isLocRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_LocRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", locRecordOptions))
						if isLocRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord = &RRSetLocRecord{}

							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range locRecordOptions.LocRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested FloatField --> LatitudeSecond
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeSecond = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeSecond()))
								// resource specToModelNested RequiredIntField --> LongitudeDegree
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeDegree = types.Int32Value(locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeDegree())
								// resource specToModelNested IntField --> LongitudeMinute
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeMinute = types.Int32Value(locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeMinute())
								// resource specToModelNested FloatField --> LongitudeSecond
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeSecond = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeSecond()))
								// resource specToModelNested FloatField --> HorizontalPrecision
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].HorizontalPrecision = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetHorizontalPrecision()))
								// resource specToModelNested RequiredIntField --> LatitudeDegree
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeDegree = types.Int32Value(locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeDegree())
								// resource specToModelNested RequiredEnumField --> LatitudeHemisphere
								if locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeHemisphere().String() != "" {
									stringValue := types.StringValue(locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeHemisphere().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeHemisphere = listValue
								}
								// resource specToModelNested RequiredEnumField --> LongitudeHemisphere
								if locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeHemisphere().String() != "" {
									stringValue := types.StringValue(locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeHemisphere().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeHemisphere = listValue
								}
								// resource specToModelNested RequiredFloatField --> Altitude
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].Altitude = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetAltitude()))
								// resource specToModelNested FloatField --> LocationDiameter
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LocationDiameter = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetLocationDiameter()))
								// resource specToModelNested FloatField --> VerticalPrecision
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].VerticalPrecision = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetVerticalPrecision()))
								// resource specToModelNested IntField --> LatitudeMinute
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeMinute = types.Int32Value(locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeMinute())
							}
							// resource specToModelNested StringField --> Name
							if locRecordOptions.LocRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Name = types.StringValue(locRecordOptions.LocRecord.GetName())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> DsRecord
						dsRecordOptions, isDsRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_DsRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", dsRecordOptions))
						if isDsRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord = &RRSetDsRecord{}

							// resource specToModelNested StringField --> Name
							if dsRecordOptions.DsRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Name = types.StringValue(dsRecordOptions.DsRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range dsRecordOptions.DsRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredUintField --> KeyTag
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].KeyTag = types.Int32Value(int32(dsRecordOptions.DsRecord.GetValues()[iii].GetKeyTag()))
								// resource specToModelNested RequiredEnumField --> DsKeyAlgorithm
								if dsRecordOptions.DsRecord.GetValues()[iii].GetDsKeyAlgorithm().String() != "" {
									stringValue := types.StringValue(dsRecordOptions.DsRecord.GetValues()[iii].GetDsKeyAlgorithm().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DsKeyAlgorithm = listValue
								}
								// resource specToModelNested RequiredOneOfField --> DigestChoice
								digestChoiceChoices := dsRecordOptions.DsRecord.GetValues()[iii].GetDigestChoice()
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice = &DSRecordValueDigestChoice{}

								// resource specToModelNested PointerOfOneOfField --> Sha1Digest
								sha1DigestOptions, isSha1Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha1Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha1DigestOptions))
								if isSha1Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha1Digest = &DSRecordValueSha1Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha1DigestOptions.Sha1Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha1Digest.Digest = types.StringValue(sha1DigestOptions.Sha1Digest.GetDigest())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha256Digest
								sha256DigestOptions, isSha256Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha256Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha256DigestOptions))
								if isSha256Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha256Digest = &DSRecordValueSha256Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha256DigestOptions.Sha256Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha256Digest.Digest = types.StringValue(sha256DigestOptions.Sha256Digest.GetDigest())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha384Digest
								sha384DigestOptions, isSha384Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha384Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha384DigestOptions))
								if isSha384Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha384Digest = &DSRecordValueSha384Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha384DigestOptions.Sha384Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha384Digest.Digest = types.StringValue(sha384DigestOptions.Sha384Digest.GetDigest())
									}
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> MxRecord
						mxRecordOptions, isMxRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_MxRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", mxRecordOptions))
						if isMxRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord = &RRSetMxRecord{}

							// resource specToModelNested StringField --> Name
							if mxRecordOptions.MxRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Name = types.StringValue(mxRecordOptions.MxRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range mxRecordOptions.MxRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested StringField --> Domain
								if mxRecordOptions.MxRecord.GetValues()[iii].GetDomain() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values[iii].Domain = types.StringValue(mxRecordOptions.MxRecord.GetValues()[iii].GetDomain())
								}
								// resource specToModelNested UintField --> Priority
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values[iii].Priority = types.Int32Value(int32(mxRecordOptions.MxRecord.GetValues()[iii].GetPriority()))
							}
						}
						// resource specToModelNested StringField --> Description
						if primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()[ii].GetDescription() != "" {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].Description = types.StringValue(primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()[ii].GetDescription())
						}
					}
				}
			}
		}
		// resource specToModelNested UintField --> Serial
		data.DnsTypeChoice.Primary.Serial = types.Int32Value(int32(primaryOptions.Primary.GetSerial()))
	}
	// resource SpecToModelRoot StringField --> SubscriptionId
	if specData.GetSubscriptionId() != "" {
		data.SubscriptionId = types.StringValue(specData.GetSubscriptionId())
	}

	if response.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, fmt.Sprintf("created a %s resource", "DnsZone"))

	// Set state to fully populated data
	diags = response.State.Set(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}
}

func (r *ResourceF5XCDnsZone) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data ResourceF5XCDnsZoneModel

	// Read Terraform prior state data into the model
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	opts := []vesapi.CallOpt{
		vesapi.WithFailIfReferred(),
	}

	httpResp, err := r.client.GetObject(ctx, ves_io_schema_dns_zone.ObjectType, data.Namespace.ValueString(), data.Name.ValueString(), opts...)
	if err != nil {
		response.Diagnostics.AddError(
			"Client Error", fmt.Sprintf("Unable to read %s, got error: %s", "dns_zone", err),
		)
		return
	}

	metaData := httpResp.GetObjMetadata()
	specData := httpResp.GetObjSpec().(*ves_io_schema_dns_zone.SpecType).GcSpec
	systemMetaData := httpResp.GetObjSystemMetadata()

	tflog.Info(ctx, fmt.Sprintf("meta_type: %v", metaData))
	tflog.Info(ctx, fmt.Sprintf("spec_type: %v", specData))
	tflog.Info(ctx, fmt.Sprintf("meta_system_type_uid_field: %v", systemMetaData.GetUid()))

	// Write back the response to the model

	data.Name = types.StringValue(metaData.GetName())

	data.Namespace = types.StringValue(metaData.GetNamespace())

	data.Uid = types.StringValue(systemMetaData.GetUid())
	labelsValue, diags := types.MapValueFrom(ctx, types.StringType, metaData.GetLabels())
	data.Labels = labelsValue
	response.Diagnostics.Append(diags...)
	annotationsValue, diags := types.MapValueFrom(ctx, types.StringType, metaData.GetAnnotations())
	data.Annotations = annotationsValue
	response.Diagnostics.Append(diags...)

	data.Description = types.StringValue(metaData.GetDescription())
	// resource SpecToModelRoot StringField --> ServiceInstanceId
	if specData.GetServiceInstanceId() != "" {
		data.ServiceInstanceId = types.StringValue(specData.GetServiceInstanceId())
	}

	// resource SpecToModelRoot UintField --> NumOfDnsRecords
	numOfDnsRecordsValue := specData.GetNumOfDnsRecords()
	if numOfDnsRecordsValue > 0 {
		tflog.Info(ctx, fmt.Sprintf("NumOfDnsRecords: %d", numOfDnsRecordsValue))
	}

	// resource SpecToModelRoot RequiredOneOfField --> DnsType
	dnsTypeChoiceChoices := specData.GetDnsType()
	data.DnsTypeChoice = &GlobalSpecTypeDnsType{}

	// resource specToModelNested PointerOfOneOfField --> Secondary
	secondaryOptions, isSecondary := dnsTypeChoiceChoices.(*ves_io_schema_dns_zone.GlobalSpecType_Secondary)
	tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", secondaryOptions))
	if isSecondary {
		data.DnsTypeChoice.Secondary = &GlobalSpecTypeSecondary{}

		// resource specToModelNested StringField --> TsigKeyName
		if secondaryOptions.Secondary.GetTsigKeyName() != "" {
			data.DnsTypeChoice.Secondary.TsigKeyName = types.StringValue(secondaryOptions.Secondary.GetTsigKeyName())
		}
		// resource specToModelNested EnumField --> TsigKeyAlgorithm
		if secondaryOptions.Secondary.GetTsigKeyAlgorithm().String() != "" {
			stringValue := types.StringValue(secondaryOptions.Secondary.GetTsigKeyAlgorithm().String())
			stringSlice := []types.String{stringValue}
			listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
			response.Diagnostics.Append(diags...)
			data.DnsTypeChoice.Secondary.TsigKeyAlgorithm = listValue
		}
		// resource specToModelNested PointerField --> TsigKeyValue
		if secondaryOptions.Secondary.GetTsigKeyValue() != nil {
			data.DnsTypeChoice.Secondary.TsigKeyValue = &SecretType{}

			// resource specToModelNested RequiredOneOfField --> SecretInfoOneof
			secretInfoChoiceChoices := secondaryOptions.Secondary.GetTsigKeyValue().GetSecretInfoOneof()
			data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice = &SecretTypeSecretInfoOneof{}

			// resource specToModelNested PointerOfOneOfField --> BlindfoldSecretInfo
			blindfoldSecretInfoOptions, isBlindfoldSecretInfo := secretInfoChoiceChoices.(*ves_io_schema.SecretType_BlindfoldSecretInfo)
			tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", blindfoldSecretInfoOptions))
			if isBlindfoldSecretInfo {
				data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo = &SecretTypeBlindfoldSecretInfo{}

				// resource specToModelNested StringField --> DecryptionProvider
				if blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetDecryptionProvider() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.DecryptionProvider = types.StringValue(blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetDecryptionProvider())
				}
				// resource specToModelNested StringField --> StoreProvider
				if blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetStoreProvider() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.StoreProvider = types.StringValue(blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetStoreProvider())
				}
				// resource specToModelNested RequiredStringField --> Location
				if blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetLocation() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.Location = types.StringValue(blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetLocation())
				}
			}
			// resource specToModelNested PointerOfOneOfField --> ClearSecretInfo
			clearSecretInfoOptions, isClearSecretInfo := secretInfoChoiceChoices.(*ves_io_schema.SecretType_ClearSecretInfo)
			tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", clearSecretInfoOptions))
			if isClearSecretInfo {
				data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo = &SecretTypeClearSecretInfo{}

				// resource specToModelNested StringField --> Provider
				if clearSecretInfoOptions.ClearSecretInfo.GetProvider() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo.Provider = types.StringValue(clearSecretInfoOptions.ClearSecretInfo.GetProvider())
				}
				// resource specToModelNested RequiredStringField --> Url
				if clearSecretInfoOptions.ClearSecretInfo.GetUrl() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo.Url = types.StringValue(clearSecretInfoOptions.ClearSecretInfo.GetUrl())
				}
			}
		}
		// resource specToModelNested TimeStampField --> LastAxfrTimestamp
		tsAsString := secondaryOptions.Secondary.GetLastAxfrTimestamp().String()
		tfTimeValue, diags := timetypes.NewRFC3339Value(tsAsString)
		response.Diagnostics.Append(diags...)
		data.DnsTypeChoice.Secondary.LastAxfrTimestamp = tfTimeValue // resource specToModelNested RequiredArrayOfStringField --> PrimaryServers
		if len(secondaryOptions.Secondary.GetPrimaryServers()) > 0 {
			listValue, diags := types.ListValueFrom(ctx, types.StringType, secondaryOptions.Secondary.GetPrimaryServers())
			response.Diagnostics.Append(diags...)
			data.DnsTypeChoice.Secondary.PrimaryServers = listValue
		}
	}
	// resource specToModelNested PointerOfOneOfField --> Primary
	primaryOptions, isPrimary := dnsTypeChoiceChoices.(*ves_io_schema_dns_zone.GlobalSpecType_Primary)
	tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", primaryOptions))
	if isPrimary {
		data.DnsTypeChoice.Primary = &GlobalSpecTypePrimary{}

		// resource specToModelNested StringField --> Admin
		if primaryOptions.Primary.GetAdmin() != "" {
			data.DnsTypeChoice.Primary.Admin = types.StringValue(primaryOptions.Primary.GetAdmin())
		}

		// resource specToModelNested ArrayOfPointerField --> DefaultRrSetGroup
		var defaultRrSetGroup = make([]DefaultRrSetGroupModel, len(primaryOptions.Primary.GetDefaultRrSetGroup()))
		a := primaryOptions.Primary.GetDefaultRrSetGroup()
		tflog.Info(ctx, fmt.Sprintf("DefaultRrSetGroup length: %v", len(a)))

		for i, elemnet := range primaryOptions.Primary.GetDefaultRrSetGroup() {
			var ns NsRecordModel
			tflog.Info(ctx, fmt.Sprintf("DefaultRrSetGroup Element: %v", primaryOptions.Primary.GetDefaultRrSetGroup()[i]))
			defaultRrSetGroup[i].Ttl = types.Int64Value(int64(elemnet.GetTtl()))
			defaultRrSetGroup[i].Description = types.StringValue(elemnet.GetDescription())
			ns.Name = types.StringValue(elemnet.GetNsRecord().GetName())
			listValue, diags := types.ListValueFrom(ctx, types.StringType, elemnet.GetNsRecord().GetValues())
			response.Diagnostics.Append(diags...)
			ns.Values = listValue
			nsObject, d := types.ObjectValueFrom(ctx,
				nsRecordObjectType.AttrTypes, // ⬅️ Pass the map[string]attr.Type here
				ns,
			)
			diags.Append(d...)
			defaultRrSetGroup[i].NsRecord = nsObject
		}

		response.Diagnostics.Append(diags...)
		if response.Diagnostics.HasError() {
			return
		}

		rrSetGroupObjects, d := types.ListValueFrom(
			ctx,
			defaultRrSetGroupObjectType, // Global type definition for the list element
			defaultRrSetGroup,           // The fully populated slice
		)

		response.Diagnostics.Append(d...)
		if response.Diagnostics.HasError() {
			return
		}

		data.DnsTypeChoice.Primary.DefaultRrSetGroup = rrSetGroupObjects

		// resource specToModelNested PointerField --> DnssecMode
		if primaryOptions.Primary.GetDnssecMode() != nil {
			data.DnsTypeChoice.Primary.DnssecMode = &DnsZoneDNSSECMode{}

			// resource specToModelNested RequiredOneOfField --> Mode
			modeChoiceChoices := primaryOptions.Primary.GetDnssecMode().GetMode()
			data.DnsTypeChoice.Primary.DnssecMode.ModeChoice = &DNSSECModeMode{}

			// resource specToModelNested BoolOfOneOfPointerField --> Enable
			_, isEnable := modeChoiceChoices.(*ves_io_schema_dns_zone.DNSSECMode_Enable)
			if isEnable {
				data.DnsTypeChoice.Primary.DnssecMode.ModeChoice.Enable = types.BoolValue(true)
			}
		}
		// resource specToModelNested BoolField --> AllowHttpLbManagedRecords
		data.DnsTypeChoice.Primary.AllowHttpLbManagedRecords = types.BoolValue(bool(primaryOptions.Primary.GetAllowHttpLbManagedRecords()))
		// resource specToModelNested RequiredOneOfField --> SoaRecordParametersChoice
		soaRecordParametersChoiceChoices := primaryOptions.Primary.GetSoaRecordParametersChoice()
		data.DnsTypeChoice.Primary.SoaRecordParametersChoice = &PrimaryDNSConfigSoaRecordParametersChoice{}

		// resource specToModelNested PointerOfOneOfField --> SoaParameters
		soaParametersOptions, isSoaParameters := soaRecordParametersChoiceChoices.(*ves_io_schema_dns_zone.PrimaryDNSConfig_SoaParameters)
		tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", soaParametersOptions))
		if isSoaParameters {
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters = &PrimaryDNSConfigSoaParameters{}

			// resource specToModelNested UintField --> Refresh
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Refresh = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetRefresh()))
			// resource specToModelNested UintField --> Retry
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Retry = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetRetry()))
			// resource specToModelNested UintField --> Expire
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Expire = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetExpire()))
			// resource specToModelNested UintField --> NegativeTtl
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.NegativeTtl = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetNegativeTtl()))
			// resource specToModelNested UintField --> Ttl
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Ttl = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetTtl()))
		}
		// resource specToModelNested BoolOfOneOfPointerField --> DefaultSoaParameters
		_, isDefaultSoaParameters := soaRecordParametersChoiceChoices.(*ves_io_schema_dns_zone.PrimaryDNSConfig_DefaultSoaParameters)
		if isDefaultSoaParameters {
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.DefaultSoaParameters = types.BoolValue(true)
		}
		// resource specToModelNested ArrayOfPointerField --> RrSetGroup
		if len(primaryOptions.Primary.GetRrSetGroup()) > 0 {
			data.DnsTypeChoice.Primary.RrSetGroup = make([]DnsZoneRRSetGroup, len(primaryOptions.Primary.GetRrSetGroup()))
			for i := range primaryOptions.Primary.GetRrSetGroup() {

				// resource specToModelNested RequiredPointerField --> Metadata
				data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata = MessageMetaType{}

				// resource specToModelNested RequiredStringField --> Name
				if primaryOptions.Primary.GetRrSetGroup()[i].GetMetadata().GetName() != "" {
					data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata.Name = types.StringValue(primaryOptions.Primary.GetRrSetGroup()[i].GetMetadata().GetName())
				}
				// resource specToModelNested StringField --> Description
				if primaryOptions.Primary.GetRrSetGroup()[i].GetMetadata().GetDescription() != "" {
					data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata.Description = types.StringValue(primaryOptions.Primary.GetRrSetGroup()[i].GetMetadata().GetDescription())
				}
				// resource specToModelNested ArrayOfPointerField --> RrSet
				if len(primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()) > 0 {
					data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet = make([]DnsZoneRRSet, len(primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()))
					for ii := range primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet() {

						// resource specToModelNested UintField --> Ttl
						data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].Ttl = types.Int32Value(int32(primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()[ii].GetTtl()))
						// resource specToModelNested RequiredOneOfField --> TypeRecordSet
						typeRecordSetChoiceChoices := primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()[ii].GetTypeRecordSet()
						data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice = &RRSetTypeRecordSet{}

						// resource specToModelNested PointerOfOneOfField --> NaptrRecord
						naptrRecordOptions, isNaptrRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_NaptrRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", naptrRecordOptions))
						if isNaptrRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord = &RRSetNaptrRecord{}

							// resource specToModelNested StringField --> Name
							if naptrRecordOptions.NaptrRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Name = types.StringValue(naptrRecordOptions.NaptrRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range naptrRecordOptions.NaptrRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredUintField --> Order
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Order = types.Int32Value(int32(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetOrder()))
								// resource specToModelNested RequiredUintField --> Preference
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Preference = types.Int32Value(int32(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetPreference()))
								// resource specToModelNested RequiredStringField --> Flags
								if naptrRecordOptions.NaptrRecord.GetValues()[iii].GetFlags() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Flags = types.StringValue(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetFlags())
								}
								// resource specToModelNested StringField --> Service
								if naptrRecordOptions.NaptrRecord.GetValues()[iii].GetService() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Service = types.StringValue(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetService())
								}
								// resource specToModelNested StringField --> Regexp
								if naptrRecordOptions.NaptrRecord.GetValues()[iii].GetRegexp() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Regexp = types.StringValue(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetRegexp())
								}
								// resource specToModelNested StringField --> Replacement
								if naptrRecordOptions.NaptrRecord.GetValues()[iii].GetReplacement() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Replacement = types.StringValue(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetReplacement())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> SrvRecord
						srvRecordOptions, isSrvRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_SrvRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", srvRecordOptions))
						if isSrvRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord = &RRSetSrvRecord{}

							// resource specToModelNested RequiredStringField --> Name
							if srvRecordOptions.SrvRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Name = types.StringValue(srvRecordOptions.SrvRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range srvRecordOptions.SrvRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested UintField --> Priority
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Priority = types.Int32Value(int32(srvRecordOptions.SrvRecord.GetValues()[iii].GetPriority()))
								// resource specToModelNested UintField --> Weight
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Weight = types.Int32Value(int32(srvRecordOptions.SrvRecord.GetValues()[iii].GetWeight()))
								// resource specToModelNested UintField --> Port
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Port = types.Int32Value(int32(srvRecordOptions.SrvRecord.GetValues()[iii].GetPort()))
								// resource specToModelNested StringField --> Target
								if srvRecordOptions.SrvRecord.GetValues()[iii].GetTarget() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Target = types.StringValue(srvRecordOptions.SrvRecord.GetValues()[iii].GetTarget())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> TlsaRecord
						tlsaRecordOptions, isTlsaRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_TlsaRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", tlsaRecordOptions))
						if isTlsaRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord = &RRSetTlsaRecord{}

							// resource specToModelNested StringField --> Name
							if tlsaRecordOptions.TlsaRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Name = types.StringValue(tlsaRecordOptions.TlsaRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range tlsaRecordOptions.TlsaRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredEnumField --> CertificateUsage
								if tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetCertificateUsage().String() != "" {
									stringValue := types.StringValue(tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetCertificateUsage().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].CertificateUsage = listValue
								}
								// resource specToModelNested RequiredEnumField --> Selector
								if tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetSelector().String() != "" {
									stringValue := types.StringValue(tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetSelector().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].Selector = listValue
								}
								// resource specToModelNested RequiredEnumField --> MatchingType
								if tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetMatchingType().String() != "" {
									stringValue := types.StringValue(tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetMatchingType().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].MatchingType = listValue
								}
								// resource specToModelNested RequiredStringField --> CertificateAssociationData
								if tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetCertificateAssociationData() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].CertificateAssociationData = types.StringValue(tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetCertificateAssociationData())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> Eui48Record
						eui48RecordOptions, isEui48Record := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_Eui48Record)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", eui48RecordOptions))
						if isEui48Record {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record = &RRSetEui48Record{}

							// resource specToModelNested RequiredStringField --> Value
							if eui48RecordOptions.Eui48Record.GetValue() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record.Value = types.StringValue(eui48RecordOptions.Eui48Record.GetValue())
							}
							// resource specToModelNested StringField --> Name
							if eui48RecordOptions.Eui48Record.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record.Name = types.StringValue(eui48RecordOptions.Eui48Record.GetName())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> Eui64Record
						eui64RecordOptions, isEui64Record := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_Eui64Record)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", eui64RecordOptions))
						if isEui64Record {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record = &RRSetEui64Record{}

							// resource specToModelNested StringField --> Name
							if eui64RecordOptions.Eui64Record.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record.Name = types.StringValue(eui64RecordOptions.Eui64Record.GetName())
							}
							// resource specToModelNested RequiredStringField --> Value
							if eui64RecordOptions.Eui64Record.GetValue() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record.Value = types.StringValue(eui64RecordOptions.Eui64Record.GetValue())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> CdsRecord
						cdsRecordOptions, isCdsRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_CdsRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", cdsRecordOptions))
						if isCdsRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord = &RRSetCdsRecord{}

							// resource specToModelNested StringField --> Name
							if cdsRecordOptions.CdsRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Name = types.StringValue(cdsRecordOptions.CdsRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range cdsRecordOptions.CdsRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredUintField --> KeyTag
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].KeyTag = types.Int32Value(int32(cdsRecordOptions.CdsRecord.GetValues()[iii].GetKeyTag()))
								// resource specToModelNested RequiredEnumField --> DsKeyAlgorithm
								if cdsRecordOptions.CdsRecord.GetValues()[iii].GetDsKeyAlgorithm().String() != "" {
									stringValue := types.StringValue(cdsRecordOptions.CdsRecord.GetValues()[iii].GetDsKeyAlgorithm().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DsKeyAlgorithm = listValue
								}
								// resource specToModelNested RequiredOneOfField --> DigestChoice
								digestChoiceChoices := cdsRecordOptions.CdsRecord.GetValues()[iii].GetDigestChoice()
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice = &DSRecordValueDigestChoice{}

								// resource specToModelNested PointerOfOneOfField --> Sha1Digest
								sha1DigestOptions, isSha1Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha1Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha1DigestOptions))
								if isSha1Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha1Digest = &DSRecordValueSha1Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha1DigestOptions.Sha1Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha1Digest.Digest = types.StringValue(sha1DigestOptions.Sha1Digest.GetDigest())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha256Digest
								sha256DigestOptions, isSha256Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha256Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha256DigestOptions))
								if isSha256Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha256Digest = &DSRecordValueSha256Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha256DigestOptions.Sha256Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha256Digest.Digest = types.StringValue(sha256DigestOptions.Sha256Digest.GetDigest())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha384Digest
								sha384DigestOptions, isSha384Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha384Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha384DigestOptions))
								if isSha384Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha384Digest = &DSRecordValueSha384Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha384DigestOptions.Sha384Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha384Digest.Digest = types.StringValue(sha384DigestOptions.Sha384Digest.GetDigest())
									}
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> CaaRecord
						caaRecordOptions, isCaaRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_CaaRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", caaRecordOptions))
						if isCaaRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord = &RRSetCaaRecord{}

							// resource specToModelNested StringField --> Name
							if caaRecordOptions.CaaRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Name = types.StringValue(caaRecordOptions.CaaRecord.GetName())
							}
							// resource specToModelNested ArrayOfPointerField --> Values
							if len(caaRecordOptions.CaaRecord.GetValues()) > 0 {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values = make([]DnsZoneCertificationAuthorityAuthorization, len(caaRecordOptions.CaaRecord.GetValues()))
								for iii := range caaRecordOptions.CaaRecord.GetValues() {

									// resource specToModelNested StringField --> Tag
									if caaRecordOptions.CaaRecord.GetValues()[iii].GetTag() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Tag = types.StringValue(caaRecordOptions.CaaRecord.GetValues()[iii].GetTag())
									}
									// resource specToModelNested StringField --> Value
									if caaRecordOptions.CaaRecord.GetValues()[iii].GetValue() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Value = types.StringValue(caaRecordOptions.CaaRecord.GetValues()[iii].GetValue())
									}
									// resource specToModelNested UintField --> Flags
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Flags = types.Int32Value(int32(caaRecordOptions.CaaRecord.GetValues()[iii].GetFlags()))
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> NsRecord
						nsRecordOptions, isNsRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_NsRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", nsRecordOptions))
						if isNsRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord = &RRSetNsRecord{}

							// resource specToModelNested StringField --> Name
							if nsRecordOptions.NsRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord.Name = types.StringValue(nsRecordOptions.NsRecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(nsRecordOptions.NsRecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, nsRecordOptions.NsRecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> AfsdbRecord
						afsdbRecordOptions, isAfsdbRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_AfsdbRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", afsdbRecordOptions))
						if isAfsdbRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord = &RRSetAfsdbRecord{}

							// resource specToModelNested StringField --> Name
							if afsdbRecordOptions.AfsdbRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Name = types.StringValue(afsdbRecordOptions.AfsdbRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range afsdbRecordOptions.AfsdbRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredEnumField --> Subtype
								if afsdbRecordOptions.AfsdbRecord.GetValues()[iii].GetSubtype().String() != "" {
									stringValue := types.StringValue(afsdbRecordOptions.AfsdbRecord.GetValues()[iii].GetSubtype().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values[iii].Subtype = listValue
								}
								// resource specToModelNested RequiredStringField --> Hostname
								if afsdbRecordOptions.AfsdbRecord.GetValues()[iii].GetHostname() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values[iii].Hostname = types.StringValue(afsdbRecordOptions.AfsdbRecord.GetValues()[iii].GetHostname())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> TxtRecord
						txtRecordOptions, isTxtRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_TxtRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", txtRecordOptions))
						if isTxtRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord = &RRSetTxtRecord{}

							// resource specToModelNested StringField --> Name
							if txtRecordOptions.TxtRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord.Name = types.StringValue(txtRecordOptions.TxtRecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(txtRecordOptions.TxtRecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, txtRecordOptions.TxtRecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> ARecord
						aRecordOptions, isARecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_ARecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", aRecordOptions))
						if isARecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord = &RRSetARecord{}

							// resource specToModelNested StringField --> Name
							if aRecordOptions.ARecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord.Name = types.StringValue(aRecordOptions.ARecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(aRecordOptions.ARecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, aRecordOptions.ARecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> AliasRecord
						aliasRecordOptions, isAliasRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_AliasRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", aliasRecordOptions))
						if isAliasRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AliasRecord = &RRSetAliasRecord{}

							// resource specToModelNested StringField --> Value
							if aliasRecordOptions.AliasRecord.GetValue() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AliasRecord.Value = types.StringValue(aliasRecordOptions.AliasRecord.GetValue())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> LbRecord
						lbRecordOptions, isLbRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_LbRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", lbRecordOptions))
						if isLbRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord = &RRSetLbRecord{}

							// resource specToModelNested StringField --> Name
							if lbRecordOptions.LbRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Name = types.StringValue(lbRecordOptions.LbRecord.GetName())
							}
							// resource specToModelNested PointerField --> Value
							if lbRecordOptions.LbRecord.GetValue() != nil {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value = &ObjectRefTypeModel{}

								// resource specToModelNested StringField --> Tenant
								if lbRecordOptions.LbRecord.GetValue().GetTenant() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Tenant = types.StringValue(lbRecordOptions.LbRecord.GetValue().GetTenant())
								}
								// resource specToModelNested StringField --> Namespace
								if lbRecordOptions.LbRecord.GetValue().GetNamespace() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Namespace = types.StringValue(lbRecordOptions.LbRecord.GetValue().GetNamespace())
								}
								// resource specToModelNested StringField --> Name
								if lbRecordOptions.LbRecord.GetValue().GetName() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Name = types.StringValue(lbRecordOptions.LbRecord.GetValue().GetName())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> SshfpRecord
						sshfpRecordOptions, isSshfpRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_SshfpRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sshfpRecordOptions))
						if isSshfpRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord = &RRSetSshfpRecord{}

							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range sshfpRecordOptions.SshfpRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredEnumField --> Algorithm
								if sshfpRecordOptions.SshfpRecord.GetValues()[iii].GetAlgorithm().String() != "" {
									stringValue := types.StringValue(sshfpRecordOptions.SshfpRecord.GetValues()[iii].GetAlgorithm().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].Algorithm = listValue
								}
								// resource specToModelNested RequiredOneOfField --> FingerprintType
								fingerprintTypeChoiceChoices := sshfpRecordOptions.SshfpRecord.GetValues()[iii].GetFingerprintType()
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice = &SSHFPRecordValueFingerprintType{}

								// resource specToModelNested PointerOfOneOfField --> Sha1Fingerprint
								sha1FingerprintOptions, isSha1Fingerprint := fingerprintTypeChoiceChoices.(*ves_io_schema_dns_zone.SSHFPRecordValue_Sha1Fingerprint)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha1FingerprintOptions))
								if isSha1Fingerprint {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha1Fingerprint = &SSHFPRecordValueSha1Fingerprint{}

									// resource specToModelNested RequiredStringField --> Fingerprint
									if sha1FingerprintOptions.Sha1Fingerprint.GetFingerprint() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha1Fingerprint.Fingerprint = types.StringValue(sha1FingerprintOptions.Sha1Fingerprint.GetFingerprint())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha256Fingerprint
								sha256FingerprintOptions, isSha256Fingerprint := fingerprintTypeChoiceChoices.(*ves_io_schema_dns_zone.SSHFPRecordValue_Sha256Fingerprint)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha256FingerprintOptions))
								if isSha256Fingerprint {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha256Fingerprint = &SSHFPRecordValueSha256Fingerprint{}

									// resource specToModelNested RequiredStringField --> Fingerprint
									if sha256FingerprintOptions.Sha256Fingerprint.GetFingerprint() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha256Fingerprint.Fingerprint = types.StringValue(sha256FingerprintOptions.Sha256Fingerprint.GetFingerprint())
									}
								}
							}
							// resource specToModelNested StringField --> Name
							if sshfpRecordOptions.SshfpRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Name = types.StringValue(sshfpRecordOptions.SshfpRecord.GetName())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> CertRecord
						certRecordOptions, isCertRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_CertRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", certRecordOptions))
						if isCertRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord = &RRSetCertRecord{}

							// resource specToModelNested StringField --> Name
							if certRecordOptions.CertRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Name = types.StringValue(certRecordOptions.CertRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range certRecordOptions.CertRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredUintField --> CertKeyTag
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].CertKeyTag = types.Int32Value(int32(certRecordOptions.CertRecord.GetValues()[iii].GetCertKeyTag()))
								// resource specToModelNested RequiredEnumField --> Algorithm
								if certRecordOptions.CertRecord.GetValues()[iii].GetAlgorithm().String() != "" {
									stringValue := types.StringValue(certRecordOptions.CertRecord.GetValues()[iii].GetAlgorithm().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].Algorithm = listValue
								}
								// resource specToModelNested RequiredStringField --> Certificate
								if certRecordOptions.CertRecord.GetValues()[iii].GetCertificate() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].Certificate = types.StringValue(certRecordOptions.CertRecord.GetValues()[iii].GetCertificate())
								}
								// resource specToModelNested RequiredEnumField --> CertType
								if certRecordOptions.CertRecord.GetValues()[iii].GetCertType().String() != "" {
									stringValue := types.StringValue(certRecordOptions.CertRecord.GetValues()[iii].GetCertType().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].CertType = listValue
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> CnameRecord
						cnameRecordOptions, isCnameRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_CnameRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", cnameRecordOptions))
						if isCnameRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord = &RRSetCnameRecord{}

							// resource specToModelNested StringField --> Value
							if cnameRecordOptions.CnameRecord.GetValue() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord.Value = types.StringValue(cnameRecordOptions.CnameRecord.GetValue())
							}
							// resource specToModelNested RequiredStringField --> Name
							if cnameRecordOptions.CnameRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord.Name = types.StringValue(cnameRecordOptions.CnameRecord.GetName())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> PtrRecord
						ptrRecordOptions, isPtrRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_PtrRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", ptrRecordOptions))
						if isPtrRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord = &RRSetPtrRecord{}

							// resource specToModelNested StringField --> Name
							if ptrRecordOptions.PtrRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord.Name = types.StringValue(ptrRecordOptions.PtrRecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(ptrRecordOptions.PtrRecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, ptrRecordOptions.PtrRecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> AaaaRecord
						aaaaRecordOptions, isAaaaRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_AaaaRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", aaaaRecordOptions))
						if isAaaaRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord = &RRSetAaaaRecord{}

							// resource specToModelNested StringField --> Name
							if aaaaRecordOptions.AaaaRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord.Name = types.StringValue(aaaaRecordOptions.AaaaRecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(aaaaRecordOptions.AaaaRecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, aaaaRecordOptions.AaaaRecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> LocRecord
						locRecordOptions, isLocRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_LocRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", locRecordOptions))
						if isLocRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord = &RRSetLocRecord{}

							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range locRecordOptions.LocRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested FloatField --> LatitudeSecond
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeSecond = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeSecond()))
								// resource specToModelNested RequiredIntField --> LongitudeDegree
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeDegree = types.Int32Value(locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeDegree())
								// resource specToModelNested IntField --> LongitudeMinute
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeMinute = types.Int32Value(locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeMinute())
								// resource specToModelNested FloatField --> LongitudeSecond
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeSecond = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeSecond()))
								// resource specToModelNested FloatField --> HorizontalPrecision
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].HorizontalPrecision = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetHorizontalPrecision()))
								// resource specToModelNested RequiredIntField --> LatitudeDegree
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeDegree = types.Int32Value(locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeDegree())
								// resource specToModelNested RequiredEnumField --> LatitudeHemisphere
								if locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeHemisphere().String() != "" {
									stringValue := types.StringValue(locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeHemisphere().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeHemisphere = listValue
								}
								// resource specToModelNested RequiredEnumField --> LongitudeHemisphere
								if locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeHemisphere().String() != "" {
									stringValue := types.StringValue(locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeHemisphere().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeHemisphere = listValue
								}
								// resource specToModelNested RequiredFloatField --> Altitude
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].Altitude = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetAltitude()))
								// resource specToModelNested FloatField --> LocationDiameter
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LocationDiameter = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetLocationDiameter()))
								// resource specToModelNested FloatField --> VerticalPrecision
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].VerticalPrecision = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetVerticalPrecision()))
								// resource specToModelNested IntField --> LatitudeMinute
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeMinute = types.Int32Value(locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeMinute())
							}
							// resource specToModelNested StringField --> Name
							if locRecordOptions.LocRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Name = types.StringValue(locRecordOptions.LocRecord.GetName())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> DsRecord
						dsRecordOptions, isDsRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_DsRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", dsRecordOptions))
						if isDsRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord = &RRSetDsRecord{}

							// resource specToModelNested StringField --> Name
							if dsRecordOptions.DsRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Name = types.StringValue(dsRecordOptions.DsRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range dsRecordOptions.DsRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredUintField --> KeyTag
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].KeyTag = types.Int32Value(int32(dsRecordOptions.DsRecord.GetValues()[iii].GetKeyTag()))
								// resource specToModelNested RequiredEnumField --> DsKeyAlgorithm
								if dsRecordOptions.DsRecord.GetValues()[iii].GetDsKeyAlgorithm().String() != "" {
									stringValue := types.StringValue(dsRecordOptions.DsRecord.GetValues()[iii].GetDsKeyAlgorithm().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DsKeyAlgorithm = listValue
								}
								// resource specToModelNested RequiredOneOfField --> DigestChoice
								digestChoiceChoices := dsRecordOptions.DsRecord.GetValues()[iii].GetDigestChoice()
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice = &DSRecordValueDigestChoice{}

								// resource specToModelNested PointerOfOneOfField --> Sha1Digest
								sha1DigestOptions, isSha1Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha1Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha1DigestOptions))
								if isSha1Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha1Digest = &DSRecordValueSha1Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha1DigestOptions.Sha1Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha1Digest.Digest = types.StringValue(sha1DigestOptions.Sha1Digest.GetDigest())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha256Digest
								sha256DigestOptions, isSha256Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha256Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha256DigestOptions))
								if isSha256Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha256Digest = &DSRecordValueSha256Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha256DigestOptions.Sha256Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha256Digest.Digest = types.StringValue(sha256DigestOptions.Sha256Digest.GetDigest())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha384Digest
								sha384DigestOptions, isSha384Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha384Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha384DigestOptions))
								if isSha384Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha384Digest = &DSRecordValueSha384Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha384DigestOptions.Sha384Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha384Digest.Digest = types.StringValue(sha384DigestOptions.Sha384Digest.GetDigest())
									}
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> MxRecord
						mxRecordOptions, isMxRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_MxRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", mxRecordOptions))
						if isMxRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord = &RRSetMxRecord{}

							// resource specToModelNested StringField --> Name
							if mxRecordOptions.MxRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Name = types.StringValue(mxRecordOptions.MxRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range mxRecordOptions.MxRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested StringField --> Domain
								if mxRecordOptions.MxRecord.GetValues()[iii].GetDomain() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values[iii].Domain = types.StringValue(mxRecordOptions.MxRecord.GetValues()[iii].GetDomain())
								}
								// resource specToModelNested UintField --> Priority
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values[iii].Priority = types.Int32Value(int32(mxRecordOptions.MxRecord.GetValues()[iii].GetPriority()))
							}
						}
						// resource specToModelNested StringField --> Description
						if primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()[ii].GetDescription() != "" {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].Description = types.StringValue(primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()[ii].GetDescription())
						}
					}
				}
			}
		}
		// resource specToModelNested UintField --> Serial
		data.DnsTypeChoice.Primary.Serial = types.Int32Value(int32(primaryOptions.Primary.GetSerial()))
	}
	// resource SpecToModelRoot StringField --> SubscriptionId
	if specData.GetSubscriptionId() != "" {
		data.SubscriptionId = types.StringValue(specData.GetSubscriptionId())
	}

	if response.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *ResourceF5XCDnsZone) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	//Update is a replace and read operation

	var data ResourceF5XCDnsZoneModel

	// Read Terraform plan data into the model
	diags := request.Plan.Get(ctx, &data)
	response.Diagnostics.Append(diags...)
	if response.Diagnostics.HasError() {
		return
	}

	meta := &ves_io_schema.ObjectReplaceMetaType{}
	spec := &ves_io_schema_dns_zone.ReplaceSpecType{}
	req := &ves_io_schema_dns_zone.ReplaceRequest{
		Metadata: meta,
		Spec:     spec,
	}

	// Populate createReq from data model
	// resource meta modelToSpecRoot RequiredStringField --> Name
	if !data.Name.IsNull() {
		meta.Name = string(data.Name.ValueString())
	}
	// resource meta modelToSpecRoot StringField --> Namespace
	if !data.Namespace.IsNull() {
		meta.Namespace = string(data.Namespace.ValueString())
	}

	// resource meta modelToSpecRoot MapOfStringField --> Labels
	meta.Labels = make(map[string]string, len(data.Labels.Elements()))
	diags = data.Labels.ElementsAs(ctx, &meta.Labels, false)
	response.Diagnostics.Append(diags...)
	// resource meta modelToSpecRoot MapOfStringField --> Annotations
	meta.Annotations = make(map[string]string, len(data.Annotations.Elements()))
	diags = data.Annotations.ElementsAs(ctx, &meta.Annotations, false)
	response.Diagnostics.Append(diags...)
	// resource meta modelToSpecRoot StringField --> Description
	if !data.Description.IsNull() {
		meta.Description = string(data.Description.ValueString())
	}

	// resource modelToSpecRoot RequiredOneOfField --> DnsType

	if data.DnsTypeChoice != nil {

		// resource modelToSpecNested PointerOfOneOfField --> Secondary
		if data.DnsTypeChoice.Secondary != nil {
			secondaryOneOfSpec := &ves_io_schema_dns_zone.ReplaceSpecType_Secondary{}
			secondaryOneOfSpec.Secondary = &ves_io_schema_dns_zone.SecondaryDNSCreateSpecType{}
			spec.DnsType = secondaryOneOfSpec

			// resource modelToSpecNested StringField --> TsigKeyName
			if !data.DnsTypeChoice.Secondary.TsigKeyName.IsNull() {
				secondaryOneOfSpec.Secondary.TsigKeyName = string(data.DnsTypeChoice.Secondary.TsigKeyName.ValueString())
			}

			// resource modelToSpecNested EnumField --> TsigKeyAlgorithm
			if !data.DnsTypeChoice.Secondary.TsigKeyAlgorithm.IsNull() && !data.DnsTypeChoice.Secondary.TsigKeyAlgorithm.IsUnknown() {
				tsigKeyAlgorithmElements := make([]types.String, 0, len(data.DnsTypeChoice.Secondary.TsigKeyAlgorithm.Elements()))
				diags = data.DnsTypeChoice.Secondary.TsigKeyAlgorithm.ElementsAs(ctx, &tsigKeyAlgorithmElements, false)
				response.Diagnostics.Append(diags...)
				secondaryOneOfSpec.Secondary.TsigKeyAlgorithm = ves_io_schema_dns_zone.TSIGKeyAlgorithm(ves_io_schema_dns_zone.TSIGKeyAlgorithm_value[tsigKeyAlgorithmElements[0].ValueString()])
			}

			// resource modelToSpecNested PointerField --> TsigKeyValue
			if data.DnsTypeChoice.Secondary.TsigKeyValue != nil {
				tsigKeyValueSpec := &ves_io_schema.SecretType{}
				secondaryOneOfSpec.Secondary.TsigKeyValue = tsigKeyValueSpec

				// resource modelToSpecNested RequiredOneOfField --> SecretInfoOneof

				// resource modelToSpecNested PointerOfOneOfField --> BlindfoldSecretInfo
				if data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo != nil {
					blindfoldSecretInfoOneOfSpec := &ves_io_schema.SecretType_BlindfoldSecretInfo{}
					blindfoldSecretInfoOneOfSpec.BlindfoldSecretInfo = &ves_io_schema.BlindfoldSecretInfoType{}
					tsigKeyValueSpec.SecretInfoOneof = blindfoldSecretInfoOneOfSpec

					// resource modelToSpecNested StringField --> DecryptionProvider
					if !data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.DecryptionProvider.IsNull() {
						blindfoldSecretInfoOneOfSpec.BlindfoldSecretInfo.DecryptionProvider = string(data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.DecryptionProvider.ValueString())
					}

					// resource modelToSpecNested StringField --> StoreProvider
					if !data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.StoreProvider.IsNull() {
						blindfoldSecretInfoOneOfSpec.BlindfoldSecretInfo.StoreProvider = string(data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.StoreProvider.ValueString())
					}

					// resource modelToSpecNested RequiredStringField --> Location
					blindfoldSecretInfoOneOfSpec.BlindfoldSecretInfo.Location = string(data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.Location.ValueString())
				}

				// resource modelToSpecNested PointerOfOneOfField --> ClearSecretInfo
				if data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo != nil {
					clearSecretInfoOneOfSpec := &ves_io_schema.SecretType_ClearSecretInfo{}
					clearSecretInfoOneOfSpec.ClearSecretInfo = &ves_io_schema.ClearSecretInfoType{}
					tsigKeyValueSpec.SecretInfoOneof = clearSecretInfoOneOfSpec

					// resource modelToSpecNested StringField --> Provider
					if !data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo.Provider.IsNull() {
						clearSecretInfoOneOfSpec.ClearSecretInfo.Provider = string(data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo.Provider.ValueString())
					}

					// resource modelToSpecNested RequiredStringField --> Url
					clearSecretInfoOneOfSpec.ClearSecretInfo.Url = string(data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo.Url.ValueString())
				}
			}

			// resource modelToSpecNested RequiredArrayOfStringField --> PrimaryServers
			elements := make([]string, 0, len(data.DnsTypeChoice.Secondary.PrimaryServers.Elements()))
			diags := data.DnsTypeChoice.Secondary.PrimaryServers.ElementsAs(ctx, &elements, false)
			response.Diagnostics.Append(diags...)
			secondaryOneOfSpec.Secondary.PrimaryServers = elements
		}

		// resource modelToSpecNested PointerOfOneOfField --> Primary
		if data.DnsTypeChoice.Primary != nil {
			primaryOneOfSpec := &ves_io_schema_dns_zone.ReplaceSpecType_Primary{}
			primaryOneOfSpec.Primary = &ves_io_schema_dns_zone.PrimaryDNSCreateSpecType{}
			spec.DnsType = primaryOneOfSpec

			// resource modelToSpecNested PointerField --> DnssecMode
			if data.DnsTypeChoice.Primary.DnssecMode != nil {
				dnssecModeSpec := &ves_io_schema_dns_zone.DNSSECMode{}
				primaryOneOfSpec.Primary.DnssecMode = dnssecModeSpec

				// resource modelToSpecNested RequiredOneOfField --> Mode

				// resource modelToSpecNested BoolOfOneOfPointerField --> Enable
				if !data.DnsTypeChoice.Primary.DnssecMode.ModeChoice.Enable.IsNull() {
					enableSpec := &ves_io_schema_dns_zone.DNSSECMode_Enable{}
					enableSpec.Enable = &ves_io_schema_dns_zone.DNSSECModeEnable{}
					dnssecModeSpec.Mode = enableSpec
				}
			}

			// resource modelToSpecNested BoolField --> AllowHttpLbManagedRecords
			if !data.DnsTypeChoice.Primary.AllowHttpLbManagedRecords.IsNull() {
				primaryOneOfSpec.Primary.AllowHttpLbManagedRecords = bool(data.DnsTypeChoice.Primary.AllowHttpLbManagedRecords.ValueBool())
			}

			// resource modelToSpecNested RequiredOneOfField --> SoaRecordParametersChoice

			// resource modelToSpecNested PointerOfOneOfField --> SoaParameters
			if data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters != nil {
				soaParametersOneOfSpec := &ves_io_schema_dns_zone.PrimaryDNSCreateSpecType_SoaParameters{}
				soaParametersOneOfSpec.SoaParameters = &ves_io_schema_dns_zone.SOARecordParameterConfig{}
				primaryOneOfSpec.Primary.SoaRecordParametersChoice = soaParametersOneOfSpec

				// resource modelToSpecNested UintField --> Refresh
				if !data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Refresh.IsNull() {
					soaParametersOneOfSpec.SoaParameters.Refresh = uint32(data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Refresh.ValueInt32())
				}

				// resource modelToSpecNested UintField --> Retry
				if !data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Retry.IsNull() {
					soaParametersOneOfSpec.SoaParameters.Retry = uint32(data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Retry.ValueInt32())
				}

				// resource modelToSpecNested UintField --> Expire
				if !data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Expire.IsNull() {
					soaParametersOneOfSpec.SoaParameters.Expire = uint32(data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Expire.ValueInt32())
				}

				// resource modelToSpecNested UintField --> NegativeTtl
				if !data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.NegativeTtl.IsNull() {
					soaParametersOneOfSpec.SoaParameters.NegativeTtl = uint32(data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.NegativeTtl.ValueInt32())
				}

				// resource modelToSpecNested UintField --> Ttl
				if !data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Ttl.IsNull() {
					soaParametersOneOfSpec.SoaParameters.Ttl = uint32(data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Ttl.ValueInt32())
				}
			}

			// resource modelToSpecNested BoolOfOneOfPointerField --> DefaultSoaParameters
			if !data.DnsTypeChoice.Primary.SoaRecordParametersChoice.DefaultSoaParameters.IsNull() {
				defaultSoaParametersSpec := &ves_io_schema_dns_zone.PrimaryDNSCreateSpecType_DefaultSoaParameters{}
				defaultSoaParametersSpec.DefaultSoaParameters = &ves_io_schema.Empty{}
				primaryOneOfSpec.Primary.SoaRecordParametersChoice = defaultSoaParametersSpec
			}

			// resource modelToSpecNested ArrayOfPointerField --> rr_set_group
			if len(data.DnsTypeChoice.Primary.RrSetGroup) > 0 {
				rrSetGroupSpec := make([]*ves_io_schema_dns_zone.RRSetGroup, len(data.DnsTypeChoice.Primary.RrSetGroup))
				primaryOneOfSpec.Primary.RrSetGroup = rrSetGroupSpec

				for i, _ := range data.DnsTypeChoice.Primary.RrSetGroup {
					primaryOneOfSpec.Primary.RrSetGroup[i] = &ves_io_schema_dns_zone.RRSetGroup{}

					// resource modelToSpecNested RequiredPointerField --> Metadata
					metadataSpec := &ves_io_schema.MessageMetaType{}
					rrSetGroupSpec[i].Metadata = metadataSpec

					// resource modelToSpecNested RequiredStringField --> Name
					metadataSpec.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata.Name.ValueString())

					// resource modelToSpecNested StringField --> Description
					if !data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata.Description.IsNull() {
						metadataSpec.Description = string(data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata.Description.ValueString())
					}

					// resource modelToSpecNested ArrayOfPointerField --> rr_set
					if len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet) > 0 {
						rrSetSpec := make([]*ves_io_schema_dns_zone.RRSet, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet))
						rrSetGroupSpec[i].RrSet = rrSetSpec

						for ii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet {
							rrSetGroupSpec[i].RrSet[ii] = &ves_io_schema_dns_zone.RRSet{}

							// resource modelToSpecNested UintField --> Ttl
							if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].Ttl.IsNull() {
								rrSetSpec[i].Ttl = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].Ttl.ValueInt32())
							}

							// resource modelToSpecNested RequiredOneOfField --> TypeRecordSet

							// resource modelToSpecNested PointerOfOneOfField --> NaptrRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord != nil {
								naptrRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_NaptrRecord{}
								naptrRecordOneOfSpec.NaptrRecord = &ves_io_schema_dns_zone.DNSNAPTRResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = naptrRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Name.IsNull() {
									naptrRecordOneOfSpec.NaptrRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.NAPTRValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values))
								naptrRecordOneOfSpec.NaptrRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values {
									naptrRecordOneOfSpec.NaptrRecord.Values[iii] = &ves_io_schema_dns_zone.NAPTRValue{}

									// resource modelToSpecNested RequiredUintField --> Order
									valuesSpec[i].Order = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Order.ValueInt32())

									// resource modelToSpecNested RequiredUintField --> Preference
									valuesSpec[i].Preference = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Preference.ValueInt32())

									// resource modelToSpecNested RequiredStringField --> Flags
									valuesSpec[i].Flags = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Flags.ValueString())

									// resource modelToSpecNested StringField --> Service
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Service.IsNull() {
										valuesSpec[i].Service = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Service.ValueString())
									}

									// resource modelToSpecNested StringField --> Regexp
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Regexp.IsNull() {
										valuesSpec[i].Regexp = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Regexp.ValueString())
									}

									// resource modelToSpecNested StringField --> Replacement
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Replacement.IsNull() {
										valuesSpec[i].Replacement = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Replacement.ValueString())
									}
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> SrvRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord != nil {
								srvRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_SrvRecord{}
								srvRecordOneOfSpec.SrvRecord = &ves_io_schema_dns_zone.DNSSRVResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = srvRecordOneOfSpec

								// resource modelToSpecNested RequiredStringField --> Name
								srvRecordOneOfSpec.SrvRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Name.ValueString())

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.SRVService, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values))
								srvRecordOneOfSpec.SrvRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values {
									srvRecordOneOfSpec.SrvRecord.Values[iii] = &ves_io_schema_dns_zone.SRVService{}

									// resource modelToSpecNested UintField --> Priority
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Priority.IsNull() {
										valuesSpec[i].Priority = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Priority.ValueInt32())
									}

									// resource modelToSpecNested UintField --> Weight
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Weight.IsNull() {
										valuesSpec[i].Weight = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Weight.ValueInt32())
									}

									// resource modelToSpecNested UintField --> Port
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Port.IsNull() {
										valuesSpec[i].Port = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Port.ValueInt32())
									}

									// resource modelToSpecNested StringField --> Target
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Target.IsNull() {
										valuesSpec[i].Target = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Target.ValueString())
									}
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> TlsaRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord != nil {
								tlsaRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_TlsaRecord{}
								tlsaRecordOneOfSpec.TlsaRecord = &ves_io_schema_dns_zone.TLSAResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = tlsaRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Name.IsNull() {
									tlsaRecordOneOfSpec.TlsaRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.TLSARecordValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values))
								tlsaRecordOneOfSpec.TlsaRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values {
									tlsaRecordOneOfSpec.TlsaRecord.Values[iii] = &ves_io_schema_dns_zone.TLSARecordValue{}

									// resource modelToSpecNested RequiredEnumField --> CertificateUsage
									certificateUsageElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].CertificateUsage.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].CertificateUsage.ElementsAs(ctx, &certificateUsageElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].CertificateUsage = ves_io_schema_dns_zone.TLSARecordCertificateUsage(ves_io_schema_dns_zone.TLSARecordCertificateUsage_value[certificateUsageElements[0].ValueString()])

									// resource modelToSpecNested RequiredEnumField --> Selector
									selectorElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].Selector.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].Selector.ElementsAs(ctx, &selectorElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].Selector = ves_io_schema_dns_zone.TLSARecordCSelector(ves_io_schema_dns_zone.TLSARecordCSelector_value[selectorElements[0].ValueString()])

									// resource modelToSpecNested RequiredEnumField --> MatchingType
									matchingTypeElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].MatchingType.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].MatchingType.ElementsAs(ctx, &matchingTypeElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].MatchingType = ves_io_schema_dns_zone.TLSARecordMatchingType(ves_io_schema_dns_zone.TLSARecordMatchingType_value[matchingTypeElements[0].ValueString()])

									// resource modelToSpecNested RequiredStringField --> CertificateAssociationData
									valuesSpec[i].CertificateAssociationData = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].CertificateAssociationData.ValueString())
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> Eui48Record
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record != nil {
								eui48RecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_Eui48Record{}
								eui48RecordOneOfSpec.Eui48Record = &ves_io_schema_dns_zone.DNSEUI48ResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = eui48RecordOneOfSpec

								// resource modelToSpecNested RequiredStringField --> Value
								eui48RecordOneOfSpec.Eui48Record.Value = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record.Value.ValueString())

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record.Name.IsNull() {
									eui48RecordOneOfSpec.Eui48Record.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record.Name.ValueString())
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> Eui64Record
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record != nil {
								eui64RecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_Eui64Record{}
								eui64RecordOneOfSpec.Eui64Record = &ves_io_schema_dns_zone.DNSEUI64ResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = eui64RecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record.Name.IsNull() {
									eui64RecordOneOfSpec.Eui64Record.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record.Name.ValueString())
								}

								// resource modelToSpecNested RequiredStringField --> Value
								eui64RecordOneOfSpec.Eui64Record.Value = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record.Value.ValueString())
							}

							// resource modelToSpecNested PointerOfOneOfField --> CdsRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord != nil {
								cdsRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_CdsRecord{}
								cdsRecordOneOfSpec.CdsRecord = &ves_io_schema_dns_zone.DNSCDSRecord{}
								rrSetSpec[ii].TypeRecordSet = cdsRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Name.IsNull() {
									cdsRecordOneOfSpec.CdsRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.DSRecordValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values))
								cdsRecordOneOfSpec.CdsRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values {
									cdsRecordOneOfSpec.CdsRecord.Values[iii] = &ves_io_schema_dns_zone.DSRecordValue{}

									// resource modelToSpecNested RequiredUintField --> KeyTag
									valuesSpec[i].KeyTag = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].KeyTag.ValueInt32())

									// resource modelToSpecNested RequiredEnumField --> DsKeyAlgorithm
									dsKeyAlgorithmElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DsKeyAlgorithm.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DsKeyAlgorithm.ElementsAs(ctx, &dsKeyAlgorithmElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].DsKeyAlgorithm = ves_io_schema_dns_zone.DSKeyAlgorithm(ves_io_schema_dns_zone.DSKeyAlgorithm_value[dsKeyAlgorithmElements[0].ValueString()])

									// resource modelToSpecNested RequiredOneOfField --> DigestChoice

									// resource modelToSpecNested PointerOfOneOfField --> Sha1Digest
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha1Digest != nil {
										sha1DigestOneOfSpec := &ves_io_schema_dns_zone.DSRecordValue_Sha1Digest{}
										sha1DigestOneOfSpec.Sha1Digest = &ves_io_schema_dns_zone.SHA1Digest{}
										valuesSpec[iii].DigestChoice = sha1DigestOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Digest
										sha1DigestOneOfSpec.Sha1Digest.Digest = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha1Digest.Digest.ValueString())
									}

									// resource modelToSpecNested PointerOfOneOfField --> Sha256Digest
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha256Digest != nil {
										sha256DigestOneOfSpec := &ves_io_schema_dns_zone.DSRecordValue_Sha256Digest{}
										sha256DigestOneOfSpec.Sha256Digest = &ves_io_schema_dns_zone.SHA256Digest{}
										valuesSpec[iii].DigestChoice = sha256DigestOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Digest
										sha256DigestOneOfSpec.Sha256Digest.Digest = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha256Digest.Digest.ValueString())
									}

									// resource modelToSpecNested PointerOfOneOfField --> Sha384Digest
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha384Digest != nil {
										sha384DigestOneOfSpec := &ves_io_schema_dns_zone.DSRecordValue_Sha384Digest{}
										sha384DigestOneOfSpec.Sha384Digest = &ves_io_schema_dns_zone.SHA384Digest{}
										valuesSpec[iii].DigestChoice = sha384DigestOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Digest
										sha384DigestOneOfSpec.Sha384Digest.Digest = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha384Digest.Digest.ValueString())
									}
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> CaaRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord != nil {
								caaRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_CaaRecord{}
								caaRecordOneOfSpec.CaaRecord = &ves_io_schema_dns_zone.DNSCAAResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = caaRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Name.IsNull() {
									caaRecordOneOfSpec.CaaRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Name.ValueString())
								}

								// resource modelToSpecNested ArrayOfPointerField --> values
								if len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values) > 0 {
									valuesSpec := make([]*ves_io_schema_dns_zone.CertificationAuthorityAuthorization, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values))
									caaRecordOneOfSpec.CaaRecord.Values = valuesSpec

									for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values {
										caaRecordOneOfSpec.CaaRecord.Values[iii] = &ves_io_schema_dns_zone.CertificationAuthorityAuthorization{}

										// resource modelToSpecNested StringField --> Tag
										if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Tag.IsNull() {
											valuesSpec[i].Tag = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Tag.ValueString())
										}

										// resource modelToSpecNested StringField --> Value
										if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Value.IsNull() {
											valuesSpec[i].Value = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Value.ValueString())
										}

										// resource modelToSpecNested UintField --> Flags
										if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Flags.IsNull() {
											valuesSpec[i].Flags = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Flags.ValueInt32())
										}
									}
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> NsRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord != nil {
								nsRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_NsRecord{}
								nsRecordOneOfSpec.NsRecord = &ves_io_schema_dns_zone.DNSNSResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = nsRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord.Name.IsNull() {
									nsRecordOneOfSpec.NsRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfStringField --> Values
								elements := make([]string, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord.Values.Elements()))
								diags := data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord.Values.ElementsAs(ctx, &elements, false)
								response.Diagnostics.Append(diags...)
								nsRecordOneOfSpec.NsRecord.Values = elements
							}

							// resource modelToSpecNested PointerOfOneOfField --> AfsdbRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord != nil {
								afsdbRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_AfsdbRecord{}
								afsdbRecordOneOfSpec.AfsdbRecord = &ves_io_schema_dns_zone.DNSAFSDBRecord{}
								rrSetSpec[ii].TypeRecordSet = afsdbRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Name.IsNull() {
									afsdbRecordOneOfSpec.AfsdbRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.AFSDBRecordValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values))
								afsdbRecordOneOfSpec.AfsdbRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values {
									afsdbRecordOneOfSpec.AfsdbRecord.Values[iii] = &ves_io_schema_dns_zone.AFSDBRecordValue{}

									// resource modelToSpecNested RequiredEnumField --> Subtype
									subtypeElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values[iii].Subtype.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values[iii].Subtype.ElementsAs(ctx, &subtypeElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].Subtype = ves_io_schema_dns_zone.AFSDBRecordSubtype(ves_io_schema_dns_zone.AFSDBRecordSubtype_value[subtypeElements[0].ValueString()])

									// resource modelToSpecNested RequiredStringField --> Hostname
									valuesSpec[i].Hostname = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values[iii].Hostname.ValueString())
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> TxtRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord != nil {
								txtRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_TxtRecord{}
								txtRecordOneOfSpec.TxtRecord = &ves_io_schema_dns_zone.DNSTXTResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = txtRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord.Name.IsNull() {
									txtRecordOneOfSpec.TxtRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfStringField --> Values
								elements := make([]string, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord.Values.Elements()))
								diags := data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord.Values.ElementsAs(ctx, &elements, false)
								response.Diagnostics.Append(diags...)
								txtRecordOneOfSpec.TxtRecord.Values = elements
							}

							// resource modelToSpecNested PointerOfOneOfField --> ARecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord != nil {
								aRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_ARecord{}
								aRecordOneOfSpec.ARecord = &ves_io_schema_dns_zone.DNSAResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = aRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord.Name.IsNull() {
									aRecordOneOfSpec.ARecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfStringField --> Values
								elements := make([]string, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord.Values.Elements()))
								diags := data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord.Values.ElementsAs(ctx, &elements, false)
								response.Diagnostics.Append(diags...)
								aRecordOneOfSpec.ARecord.Values = elements
							}

							// resource modelToSpecNested PointerOfOneOfField --> AliasRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AliasRecord != nil {
								aliasRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_AliasRecord{}
								aliasRecordOneOfSpec.AliasRecord = &ves_io_schema_dns_zone.DNSAliasResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = aliasRecordOneOfSpec

								// resource modelToSpecNested StringField --> Value
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AliasRecord.Value.IsNull() {
									aliasRecordOneOfSpec.AliasRecord.Value = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AliasRecord.Value.ValueString())
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> LbRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord != nil {
								lbRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_LbRecord{}
								lbRecordOneOfSpec.LbRecord = &ves_io_schema_dns_zone.DNSLBResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = lbRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Name.IsNull() {
									lbRecordOneOfSpec.LbRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Name.ValueString())
								}

								// resource modelToSpecNested PointerField --> Value
								if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value != nil {
									valueSpec := &ves_io_schema_views.ObjectRefType{}
									lbRecordOneOfSpec.LbRecord.Value = valueSpec

									// resource modelToSpecNested StringField --> Tenant
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Tenant.IsNull() {
										valueSpec.Tenant = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Tenant.ValueString())
									}

									// resource modelToSpecNested StringField --> Namespace
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Namespace.IsNull() {
										valueSpec.Namespace = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Namespace.ValueString())
									}

									// resource modelToSpecNested StringField --> Name
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Name.IsNull() {
										valueSpec.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Name.ValueString())
									}
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> SshfpRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord != nil {
								sshfpRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_SshfpRecord{}
								sshfpRecordOneOfSpec.SshfpRecord = &ves_io_schema_dns_zone.SSHFPResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = sshfpRecordOneOfSpec

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.SSHFPRecordValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values))
								sshfpRecordOneOfSpec.SshfpRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values {
									sshfpRecordOneOfSpec.SshfpRecord.Values[iii] = &ves_io_schema_dns_zone.SSHFPRecordValue{}

									// resource modelToSpecNested RequiredEnumField --> Algorithm
									algorithmElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].Algorithm.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].Algorithm.ElementsAs(ctx, &algorithmElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].Algorithm = ves_io_schema_dns_zone.SSHFPAlgorithm(ves_io_schema_dns_zone.SSHFPAlgorithm_value[algorithmElements[0].ValueString()])

									// resource modelToSpecNested RequiredOneOfField --> FingerprintType

									// resource modelToSpecNested PointerOfOneOfField --> Sha1Fingerprint
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha1Fingerprint != nil {
										sha1FingerprintOneOfSpec := &ves_io_schema_dns_zone.SSHFPRecordValue_Sha1Fingerprint{}
										sha1FingerprintOneOfSpec.Sha1Fingerprint = &ves_io_schema_dns_zone.SHA1Fingerprint{}
										valuesSpec[iii].FingerprintType = sha1FingerprintOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Fingerprint
										sha1FingerprintOneOfSpec.Sha1Fingerprint.Fingerprint = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha1Fingerprint.Fingerprint.ValueString())
									}

									// resource modelToSpecNested PointerOfOneOfField --> Sha256Fingerprint
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha256Fingerprint != nil {
										sha256FingerprintOneOfSpec := &ves_io_schema_dns_zone.SSHFPRecordValue_Sha256Fingerprint{}
										sha256FingerprintOneOfSpec.Sha256Fingerprint = &ves_io_schema_dns_zone.SHA256Fingerprint{}
										valuesSpec[iii].FingerprintType = sha256FingerprintOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Fingerprint
										sha256FingerprintOneOfSpec.Sha256Fingerprint.Fingerprint = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha256Fingerprint.Fingerprint.ValueString())
									}
								}

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Name.IsNull() {
									sshfpRecordOneOfSpec.SshfpRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Name.ValueString())
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> CertRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord != nil {
								certRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_CertRecord{}
								certRecordOneOfSpec.CertRecord = &ves_io_schema_dns_zone.CERTResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = certRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Name.IsNull() {
									certRecordOneOfSpec.CertRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.CERTRecordValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values))
								certRecordOneOfSpec.CertRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values {
									certRecordOneOfSpec.CertRecord.Values[iii] = &ves_io_schema_dns_zone.CERTRecordValue{}

									// resource modelToSpecNested RequiredUintField --> CertKeyTag
									valuesSpec[i].CertKeyTag = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].CertKeyTag.ValueInt32())

									// resource modelToSpecNested RequiredEnumField --> Algorithm
									algorithmElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].Algorithm.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].Algorithm.ElementsAs(ctx, &algorithmElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].Algorithm = ves_io_schema_dns_zone.CERTAlgorithm(ves_io_schema_dns_zone.CERTAlgorithm_value[algorithmElements[0].ValueString()])

									// resource modelToSpecNested RequiredStringField --> Certificate
									valuesSpec[i].Certificate = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].Certificate.ValueString())

									// resource modelToSpecNested RequiredEnumField --> CertType
									certTypeElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].CertType.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].CertType.ElementsAs(ctx, &certTypeElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].CertType = ves_io_schema_dns_zone.CERTType(ves_io_schema_dns_zone.CERTType_value[certTypeElements[0].ValueString()])
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> CnameRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord != nil {
								cnameRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_CnameRecord{}
								cnameRecordOneOfSpec.CnameRecord = &ves_io_schema_dns_zone.DNSCNAMEResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = cnameRecordOneOfSpec

								// resource modelToSpecNested StringField --> Value
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord.Value.IsNull() {
									cnameRecordOneOfSpec.CnameRecord.Value = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord.Value.ValueString())
								}

								// resource modelToSpecNested RequiredStringField --> Name
								cnameRecordOneOfSpec.CnameRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord.Name.ValueString())
							}

							// resource modelToSpecNested PointerOfOneOfField --> PtrRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord != nil {
								ptrRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_PtrRecord{}
								ptrRecordOneOfSpec.PtrRecord = &ves_io_schema_dns_zone.DNSPTRResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = ptrRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord.Name.IsNull() {
									ptrRecordOneOfSpec.PtrRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfStringField --> Values
								elements := make([]string, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord.Values.Elements()))
								diags := data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord.Values.ElementsAs(ctx, &elements, false)
								response.Diagnostics.Append(diags...)
								ptrRecordOneOfSpec.PtrRecord.Values = elements
							}

							// resource modelToSpecNested PointerOfOneOfField --> AaaaRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord != nil {
								aaaaRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_AaaaRecord{}
								aaaaRecordOneOfSpec.AaaaRecord = &ves_io_schema_dns_zone.DNSAAAAResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = aaaaRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord.Name.IsNull() {
									aaaaRecordOneOfSpec.AaaaRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfStringField --> Values
								elements := make([]string, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord.Values.Elements()))
								diags := data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord.Values.ElementsAs(ctx, &elements, false)
								response.Diagnostics.Append(diags...)
								aaaaRecordOneOfSpec.AaaaRecord.Values = elements
							}

							// resource modelToSpecNested PointerOfOneOfField --> LocRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord != nil {
								locRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_LocRecord{}
								locRecordOneOfSpec.LocRecord = &ves_io_schema_dns_zone.DNSLOCResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = locRecordOneOfSpec

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.LOCValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values))
								locRecordOneOfSpec.LocRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values {
									locRecordOneOfSpec.LocRecord.Values[iii] = &ves_io_schema_dns_zone.LOCValue{}

									// resource modelToSpecNested FloatField --> LatitudeSecond
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeSecond.IsNull() {
										valuesSpec[i].LatitudeSecond = float32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeSecond.ValueFloat32())
									}

									// resource modelToSpecNested RequiredIntField --> LongitudeDegree
									valuesSpec[i].LongitudeDegree = int32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeDegree.ValueInt32())

									// resource modelToSpecNested IntField --> LongitudeMinute
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeMinute.IsNull() {
										valuesSpec[i].LongitudeMinute = int32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeMinute.ValueInt32())
									}

									// resource modelToSpecNested FloatField --> LongitudeSecond
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeSecond.IsNull() {
										valuesSpec[i].LongitudeSecond = float32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeSecond.ValueFloat32())
									}

									// resource modelToSpecNested FloatField --> HorizontalPrecision
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].HorizontalPrecision.IsNull() {
										valuesSpec[i].HorizontalPrecision = float32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].HorizontalPrecision.ValueFloat32())
									}

									// resource modelToSpecNested RequiredIntField --> LatitudeDegree
									valuesSpec[i].LatitudeDegree = int32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeDegree.ValueInt32())

									// resource modelToSpecNested RequiredEnumField --> LatitudeHemisphere
									latitudeHemisphereElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeHemisphere.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeHemisphere.ElementsAs(ctx, &latitudeHemisphereElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].LatitudeHemisphere = ves_io_schema_dns_zone.LatitudeHemisphere(ves_io_schema_dns_zone.LatitudeHemisphere_value[latitudeHemisphereElements[0].ValueString()])

									// resource modelToSpecNested RequiredEnumField --> LongitudeHemisphere
									longitudeHemisphereElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeHemisphere.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeHemisphere.ElementsAs(ctx, &longitudeHemisphereElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].LongitudeHemisphere = ves_io_schema_dns_zone.LongitudeHemisphere(ves_io_schema_dns_zone.LongitudeHemisphere_value[longitudeHemisphereElements[0].ValueString()])

									// resource modelToSpecNested FloatField --> LocationDiameter
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LocationDiameter.IsNull() {
										valuesSpec[i].LocationDiameter = float32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LocationDiameter.ValueFloat32())
									}

									// resource modelToSpecNested FloatField --> VerticalPrecision
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].VerticalPrecision.IsNull() {
										valuesSpec[i].VerticalPrecision = float32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].VerticalPrecision.ValueFloat32())
									}

									// resource modelToSpecNested IntField --> LatitudeMinute
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeMinute.IsNull() {
										valuesSpec[i].LatitudeMinute = int32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeMinute.ValueInt32())
									}
								}

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Name.IsNull() {
									locRecordOneOfSpec.LocRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Name.ValueString())
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> DsRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord != nil {
								dsRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_DsRecord{}
								dsRecordOneOfSpec.DsRecord = &ves_io_schema_dns_zone.DNSDSRecord{}
								rrSetSpec[ii].TypeRecordSet = dsRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Name.IsNull() {
									dsRecordOneOfSpec.DsRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.DSRecordValue, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values))
								dsRecordOneOfSpec.DsRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values {
									dsRecordOneOfSpec.DsRecord.Values[iii] = &ves_io_schema_dns_zone.DSRecordValue{}

									// resource modelToSpecNested RequiredUintField --> KeyTag
									valuesSpec[i].KeyTag = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].KeyTag.ValueInt32())

									// resource modelToSpecNested RequiredEnumField --> DsKeyAlgorithm
									dsKeyAlgorithmElements := make([]types.String, 0, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DsKeyAlgorithm.Elements()))
									diags = data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DsKeyAlgorithm.ElementsAs(ctx, &dsKeyAlgorithmElements, false)
									response.Diagnostics.Append(diags...)
									valuesSpec[i].DsKeyAlgorithm = ves_io_schema_dns_zone.DSKeyAlgorithm(ves_io_schema_dns_zone.DSKeyAlgorithm_value[dsKeyAlgorithmElements[0].ValueString()])

									// resource modelToSpecNested RequiredOneOfField --> DigestChoice

									// resource modelToSpecNested PointerOfOneOfField --> Sha1Digest
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha1Digest != nil {
										sha1DigestOneOfSpec := &ves_io_schema_dns_zone.DSRecordValue_Sha1Digest{}
										sha1DigestOneOfSpec.Sha1Digest = &ves_io_schema_dns_zone.SHA1Digest{}
										valuesSpec[iii].DigestChoice = sha1DigestOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Digest
										sha1DigestOneOfSpec.Sha1Digest.Digest = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha1Digest.Digest.ValueString())
									}

									// resource modelToSpecNested PointerOfOneOfField --> Sha256Digest
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha256Digest != nil {
										sha256DigestOneOfSpec := &ves_io_schema_dns_zone.DSRecordValue_Sha256Digest{}
										sha256DigestOneOfSpec.Sha256Digest = &ves_io_schema_dns_zone.SHA256Digest{}
										valuesSpec[iii].DigestChoice = sha256DigestOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Digest
										sha256DigestOneOfSpec.Sha256Digest.Digest = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha256Digest.Digest.ValueString())
									}

									// resource modelToSpecNested PointerOfOneOfField --> Sha384Digest
									if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha384Digest != nil {
										sha384DigestOneOfSpec := &ves_io_schema_dns_zone.DSRecordValue_Sha384Digest{}
										sha384DigestOneOfSpec.Sha384Digest = &ves_io_schema_dns_zone.SHA384Digest{}
										valuesSpec[iii].DigestChoice = sha384DigestOneOfSpec

										// resource modelToSpecNested RequiredStringField --> Digest
										sha384DigestOneOfSpec.Sha384Digest.Digest = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha384Digest.Digest.ValueString())
									}
								}
							}

							// resource modelToSpecNested PointerOfOneOfField --> MxRecord
							if data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord != nil {
								mxRecordOneOfSpec := &ves_io_schema_dns_zone.RRSet_MxRecord{}
								mxRecordOneOfSpec.MxRecord = &ves_io_schema_dns_zone.DNSMXResourceRecord{}
								rrSetSpec[ii].TypeRecordSet = mxRecordOneOfSpec

								// resource modelToSpecNested StringField --> Name
								if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Name.IsNull() {
									mxRecordOneOfSpec.MxRecord.Name = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Name.ValueString())
								}

								// resource modelToSpecNested RequiredArrayOfPointerField --> values
								valuesSpec := make([]*ves_io_schema_dns_zone.MailExchanger, len(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values))
								mxRecordOneOfSpec.MxRecord.Values = valuesSpec

								for iii, _ := range data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values {
									mxRecordOneOfSpec.MxRecord.Values[iii] = &ves_io_schema_dns_zone.MailExchanger{}

									// resource modelToSpecNested StringField --> Domain
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values[iii].Domain.IsNull() {
										valuesSpec[i].Domain = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values[iii].Domain.ValueString())
									}

									// resource modelToSpecNested UintField --> Priority
									if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values[iii].Priority.IsNull() {
										valuesSpec[i].Priority = uint32(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values[iii].Priority.ValueInt32())
									}
								}
							}

							// resource modelToSpecNested StringField --> Description
							if !data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].Description.IsNull() {
								rrSetSpec[i].Description = string(data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].Description.ValueString())
							}
						}
					}
				}
			}

		}
	}

	if response.Diagnostics.HasError() {
		return
	}

	opts := []vesapi.CallOpt{
		vesapi.WithFailIfReferred(),
	}

	// Replace object
	err := r.client.ReplaceObject(ctx, ves_io_schema_views_origin_pool.ObjectType, req, opts...)
	if err != nil {
		response.Diagnostics.AddError(
			"Client Error", fmt.Sprintf("Unable to replace %s, got error: %s", "dns_zone", err),
		)
		return
	}

	// Read the replaced object
	httpResp, err := r.client.GetObject(ctx, ves_io_schema_dns_zone.ObjectType, data.Namespace.ValueString(), data.Name.ValueString(), opts...)
	if err != nil {
		response.Diagnostics.AddError(
			"Client Error", fmt.Sprintf("Unable to replace read %s, got error: %s", "dns_zone", err),
		)
		return
	}

	metaData := httpResp.GetObjMetadata()
	specData := httpResp.GetObjSpec().(*ves_io_schema_dns_zone.SpecType).GcSpec
	systemMetaData := httpResp.GetObjSystemMetadata()

	tflog.Info(ctx, fmt.Sprintf("meta_type: %v", metaData))
	tflog.Info(ctx, fmt.Sprintf("spec_type: %v", specData))
	tflog.Info(ctx, fmt.Sprintf("meta_system_type_uid_field: %v", systemMetaData.GetUid()))

	// Write back the response to the model

	data.Name = types.StringValue(metaData.GetName())

	data.Namespace = types.StringValue(metaData.GetNamespace())

	data.Uid = types.StringValue(systemMetaData.GetUid())
	labelsValue, diags := types.MapValueFrom(ctx, types.StringType, metaData.GetLabels())
	data.Labels = labelsValue
	response.Diagnostics.Append(diags...)
	annotationsValue, diags := types.MapValueFrom(ctx, types.StringType, metaData.GetAnnotations())
	data.Annotations = annotationsValue
	response.Diagnostics.Append(diags...)

	data.Description = types.StringValue(metaData.GetDescription())
	// resource SpecToModelRoot StringField --> ServiceInstanceId
	if specData.GetServiceInstanceId() != "" {
		data.ServiceInstanceId = types.StringValue(specData.GetServiceInstanceId())
	}

	// resource SpecToModelRoot UintField --> NumOfDnsRecords
	numOfDnsRecordsValue := specData.GetNumOfDnsRecords()
	if numOfDnsRecordsValue > 0 {
	}

	// resource SpecToModelRoot RequiredOneOfField --> DnsType
	dnsTypeChoiceChoices := specData.GetDnsType()
	data.DnsTypeChoice = &GlobalSpecTypeDnsType{}

	// resource specToModelNested PointerOfOneOfField --> Secondary
	secondaryOptions, isSecondary := dnsTypeChoiceChoices.(*ves_io_schema_dns_zone.GlobalSpecType_Secondary)
	tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", secondaryOptions))
	if isSecondary {
		data.DnsTypeChoice.Secondary = &GlobalSpecTypeSecondary{}

		// resource specToModelNested StringField --> TsigKeyName
		if secondaryOptions.Secondary.GetTsigKeyName() != "" {
			data.DnsTypeChoice.Secondary.TsigKeyName = types.StringValue(secondaryOptions.Secondary.GetTsigKeyName())
		}
		// resource specToModelNested EnumField --> TsigKeyAlgorithm
		if secondaryOptions.Secondary.GetTsigKeyAlgorithm().String() != "" {
			stringValue := types.StringValue(secondaryOptions.Secondary.GetTsigKeyAlgorithm().String())
			stringSlice := []types.String{stringValue}
			listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
			response.Diagnostics.Append(diags...)
			data.DnsTypeChoice.Secondary.TsigKeyAlgorithm = listValue
		}
		// resource specToModelNested PointerField --> TsigKeyValue
		if secondaryOptions.Secondary.GetTsigKeyValue() != nil {
			data.DnsTypeChoice.Secondary.TsigKeyValue = &SecretType{}

			// resource specToModelNested RequiredOneOfField --> SecretInfoOneof
			secretInfoChoiceChoices := secondaryOptions.Secondary.GetTsigKeyValue().GetSecretInfoOneof()
			data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice = &SecretTypeSecretInfoOneof{}

			// resource specToModelNested PointerOfOneOfField --> BlindfoldSecretInfo
			blindfoldSecretInfoOptions, isBlindfoldSecretInfo := secretInfoChoiceChoices.(*ves_io_schema.SecretType_BlindfoldSecretInfo)
			tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", blindfoldSecretInfoOptions))
			if isBlindfoldSecretInfo {
				data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo = &SecretTypeBlindfoldSecretInfo{}

				// resource specToModelNested StringField --> DecryptionProvider
				if blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetDecryptionProvider() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.DecryptionProvider = types.StringValue(blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetDecryptionProvider())
				}
				// resource specToModelNested StringField --> StoreProvider
				if blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetStoreProvider() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.StoreProvider = types.StringValue(blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetStoreProvider())
				}
				// resource specToModelNested RequiredStringField --> Location
				if blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetLocation() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.BlindfoldSecretInfo.Location = types.StringValue(blindfoldSecretInfoOptions.BlindfoldSecretInfo.GetLocation())
				}
			}
			// resource specToModelNested PointerOfOneOfField --> ClearSecretInfo
			clearSecretInfoOptions, isClearSecretInfo := secretInfoChoiceChoices.(*ves_io_schema.SecretType_ClearSecretInfo)
			tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", clearSecretInfoOptions))
			if isClearSecretInfo {
				data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo = &SecretTypeClearSecretInfo{}

				// resource specToModelNested StringField --> Provider
				if clearSecretInfoOptions.ClearSecretInfo.GetProvider() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo.Provider = types.StringValue(clearSecretInfoOptions.ClearSecretInfo.GetProvider())
				}
				// resource specToModelNested RequiredStringField --> Url
				if clearSecretInfoOptions.ClearSecretInfo.GetUrl() != "" {
					data.DnsTypeChoice.Secondary.TsigKeyValue.SecretInfoChoice.ClearSecretInfo.Url = types.StringValue(clearSecretInfoOptions.ClearSecretInfo.GetUrl())
				}
			}
		}
		// resource specToModelNested TimeStampField --> LastAxfrTimestamp
		tsAsString := secondaryOptions.Secondary.GetLastAxfrTimestamp().String()
		tfTimeValue, diags := timetypes.NewRFC3339Value(tsAsString)
		response.Diagnostics.Append(diags...)
		data.DnsTypeChoice.Secondary.LastAxfrTimestamp = tfTimeValue // resource specToModelNested RequiredArrayOfStringField --> PrimaryServers
		if len(secondaryOptions.Secondary.GetPrimaryServers()) > 0 {
			listValue, diags := types.ListValueFrom(ctx, types.StringType, secondaryOptions.Secondary.GetPrimaryServers())
			response.Diagnostics.Append(diags...)
			data.DnsTypeChoice.Secondary.PrimaryServers = listValue
		}
	}
	// resource specToModelNested PointerOfOneOfField --> Primary
	primaryOptions, isPrimary := dnsTypeChoiceChoices.(*ves_io_schema_dns_zone.GlobalSpecType_Primary)
	tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", primaryOptions))
	if isPrimary {
		data.DnsTypeChoice.Primary = &GlobalSpecTypePrimary{}

		// resource specToModelNested StringField --> Admin
		if primaryOptions.Primary.GetAdmin() != "" {
			data.DnsTypeChoice.Primary.Admin = types.StringValue(primaryOptions.Primary.GetAdmin())
		}

		// resource specToModelNested ArrayOfPointerField --> DefaultRrSetGroup
		var defaultRrSetGroup []DefaultRrSetGroupModel = make([]DefaultRrSetGroupModel, len(primaryOptions.Primary.GetDefaultRrSetGroup()))
		a := primaryOptions.Primary.GetDefaultRrSetGroup()
		tflog.Info(ctx, fmt.Sprintf("DefaultRrSetGroup length: %v", len(a)))

		for i, elemnet := range primaryOptions.Primary.GetDefaultRrSetGroup() {
			var ns NsRecordModel
			tflog.Info(ctx, fmt.Sprintf("DefaultRrSetGroup Element: %v", primaryOptions.Primary.GetDefaultRrSetGroup()[i]))
			defaultRrSetGroup[i].Ttl = types.Int64Value(int64(elemnet.GetTtl()))
			defaultRrSetGroup[i].Description = types.StringValue(elemnet.GetDescription())
			ns.Name = types.StringValue(elemnet.GetNsRecord().GetName())
			listValue, diags := types.ListValueFrom(ctx, types.StringType, elemnet.GetNsRecord().GetValues())
			response.Diagnostics.Append(diags...)
			ns.Values = listValue
			nsObject, d := types.ObjectValueFrom(ctx,
				nsRecordObjectType.AttrTypes, // ⬅️ Pass the map[string]attr.Type here
				ns,
			)
			diags.Append(d...)
			defaultRrSetGroup[i].NsRecord = nsObject
		}

		response.Diagnostics.Append(diags...)
		if response.Diagnostics.HasError() {
			return
		}

		rrSetGroupObjects, d := types.ListValueFrom(
			ctx,
			defaultRrSetGroupObjectType, // Global type definition for the list element
			defaultRrSetGroup,           // The fully populated slice
		)

		response.Diagnostics.Append(d...)
		if response.Diagnostics.HasError() {
			return
		}

		data.DnsTypeChoice.Primary.DefaultRrSetGroup = rrSetGroupObjects

		// resource specToModelNested PointerField --> DnssecMode
		if primaryOptions.Primary.GetDnssecMode() != nil {
			data.DnsTypeChoice.Primary.DnssecMode = &DnsZoneDNSSECMode{}

			// resource specToModelNested RequiredOneOfField --> Mode
			modeChoiceChoices := primaryOptions.Primary.GetDnssecMode().GetMode()
			data.DnsTypeChoice.Primary.DnssecMode.ModeChoice = &DNSSECModeMode{}

			// resource specToModelNested BoolOfOneOfPointerField --> Enable
			_, isEnable := modeChoiceChoices.(*ves_io_schema_dns_zone.DNSSECMode_Enable)
			if isEnable {
				data.DnsTypeChoice.Primary.DnssecMode.ModeChoice.Enable = types.BoolValue(true)
			}
		}
		// resource specToModelNested BoolField --> AllowHttpLbManagedRecords
		data.DnsTypeChoice.Primary.AllowHttpLbManagedRecords = types.BoolValue(bool(primaryOptions.Primary.GetAllowHttpLbManagedRecords()))
		// resource specToModelNested RequiredOneOfField --> SoaRecordParametersChoice
		soaRecordParametersChoiceChoices := primaryOptions.Primary.GetSoaRecordParametersChoice()
		data.DnsTypeChoice.Primary.SoaRecordParametersChoice = &PrimaryDNSConfigSoaRecordParametersChoice{}

		// resource specToModelNested PointerOfOneOfField --> SoaParameters
		soaParametersOptions, isSoaParameters := soaRecordParametersChoiceChoices.(*ves_io_schema_dns_zone.PrimaryDNSConfig_SoaParameters)
		tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", soaParametersOptions))
		if isSoaParameters {
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters = &PrimaryDNSConfigSoaParameters{}

			// resource specToModelNested UintField --> Refresh
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Refresh = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetRefresh()))
			// resource specToModelNested UintField --> Retry
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Retry = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetRetry()))
			// resource specToModelNested UintField --> Expire
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Expire = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetExpire()))
			// resource specToModelNested UintField --> NegativeTtl
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.NegativeTtl = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetNegativeTtl()))
			// resource specToModelNested UintField --> Ttl
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.SoaParameters.Ttl = types.Int32Value(int32(soaParametersOptions.SoaParameters.GetTtl()))
		}
		// resource specToModelNested BoolOfOneOfPointerField --> DefaultSoaParameters
		_, isDefaultSoaParameters := soaRecordParametersChoiceChoices.(*ves_io_schema_dns_zone.PrimaryDNSConfig_DefaultSoaParameters)
		if isDefaultSoaParameters {
			data.DnsTypeChoice.Primary.SoaRecordParametersChoice.DefaultSoaParameters = types.BoolValue(true)
		}
		// resource specToModelNested ArrayOfPointerField --> RrSetGroup
		if len(primaryOptions.Primary.GetRrSetGroup()) > 0 {
			data.DnsTypeChoice.Primary.RrSetGroup = make([]DnsZoneRRSetGroup, len(primaryOptions.Primary.GetRrSetGroup()))
			for i := range primaryOptions.Primary.GetRrSetGroup() {

				// resource specToModelNested RequiredPointerField --> Metadata
				data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata = MessageMetaType{}

				// resource specToModelNested RequiredStringField --> Name
				if primaryOptions.Primary.GetRrSetGroup()[i].GetMetadata().GetName() != "" {
					data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata.Name = types.StringValue(primaryOptions.Primary.GetRrSetGroup()[i].GetMetadata().GetName())
				}
				// resource specToModelNested StringField --> Description
				if primaryOptions.Primary.GetRrSetGroup()[i].GetMetadata().GetDescription() != "" {
					data.DnsTypeChoice.Primary.RrSetGroup[i].Metadata.Description = types.StringValue(primaryOptions.Primary.GetRrSetGroup()[i].GetMetadata().GetDescription())
				}
				// resource specToModelNested ArrayOfPointerField --> RrSet
				if len(primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()) > 0 {
					data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet = make([]DnsZoneRRSet, len(primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()))
					for ii := range primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet() {

						// resource specToModelNested UintField --> Ttl
						data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].Ttl = types.Int32Value(int32(primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()[ii].GetTtl()))
						// resource specToModelNested RequiredOneOfField --> TypeRecordSet
						typeRecordSetChoiceChoices := primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()[ii].GetTypeRecordSet()
						data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice = &RRSetTypeRecordSet{}

						// resource specToModelNested PointerOfOneOfField --> NaptrRecord
						naptrRecordOptions, isNaptrRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_NaptrRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", naptrRecordOptions))
						if isNaptrRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord = &RRSetNaptrRecord{}

							// resource specToModelNested StringField --> Name
							if naptrRecordOptions.NaptrRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Name = types.StringValue(naptrRecordOptions.NaptrRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range naptrRecordOptions.NaptrRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredUintField --> Order
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Order = types.Int32Value(int32(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetOrder()))
								// resource specToModelNested RequiredUintField --> Preference
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Preference = types.Int32Value(int32(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetPreference()))
								// resource specToModelNested RequiredStringField --> Flags
								if naptrRecordOptions.NaptrRecord.GetValues()[iii].GetFlags() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Flags = types.StringValue(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetFlags())
								}
								// resource specToModelNested StringField --> Service
								if naptrRecordOptions.NaptrRecord.GetValues()[iii].GetService() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Service = types.StringValue(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetService())
								}
								// resource specToModelNested StringField --> Regexp
								if naptrRecordOptions.NaptrRecord.GetValues()[iii].GetRegexp() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Regexp = types.StringValue(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetRegexp())
								}
								// resource specToModelNested StringField --> Replacement
								if naptrRecordOptions.NaptrRecord.GetValues()[iii].GetReplacement() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NaptrRecord.Values[iii].Replacement = types.StringValue(naptrRecordOptions.NaptrRecord.GetValues()[iii].GetReplacement())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> SrvRecord
						srvRecordOptions, isSrvRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_SrvRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", srvRecordOptions))
						if isSrvRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord = &RRSetSrvRecord{}

							// resource specToModelNested RequiredStringField --> Name
							if srvRecordOptions.SrvRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Name = types.StringValue(srvRecordOptions.SrvRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range srvRecordOptions.SrvRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested UintField --> Priority
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Priority = types.Int32Value(int32(srvRecordOptions.SrvRecord.GetValues()[iii].GetPriority()))
								// resource specToModelNested UintField --> Weight
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Weight = types.Int32Value(int32(srvRecordOptions.SrvRecord.GetValues()[iii].GetWeight()))
								// resource specToModelNested UintField --> Port
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Port = types.Int32Value(int32(srvRecordOptions.SrvRecord.GetValues()[iii].GetPort()))
								// resource specToModelNested StringField --> Target
								if srvRecordOptions.SrvRecord.GetValues()[iii].GetTarget() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SrvRecord.Values[iii].Target = types.StringValue(srvRecordOptions.SrvRecord.GetValues()[iii].GetTarget())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> TlsaRecord
						tlsaRecordOptions, isTlsaRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_TlsaRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", tlsaRecordOptions))
						if isTlsaRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord = &RRSetTlsaRecord{}

							// resource specToModelNested StringField --> Name
							if tlsaRecordOptions.TlsaRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Name = types.StringValue(tlsaRecordOptions.TlsaRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range tlsaRecordOptions.TlsaRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredEnumField --> CertificateUsage
								if tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetCertificateUsage().String() != "" {
									stringValue := types.StringValue(tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetCertificateUsage().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].CertificateUsage = listValue
								}
								// resource specToModelNested RequiredEnumField --> Selector
								if tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetSelector().String() != "" {
									stringValue := types.StringValue(tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetSelector().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].Selector = listValue
								}
								// resource specToModelNested RequiredEnumField --> MatchingType
								if tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetMatchingType().String() != "" {
									stringValue := types.StringValue(tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetMatchingType().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].MatchingType = listValue
								}
								// resource specToModelNested RequiredStringField --> CertificateAssociationData
								if tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetCertificateAssociationData() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TlsaRecord.Values[iii].CertificateAssociationData = types.StringValue(tlsaRecordOptions.TlsaRecord.GetValues()[iii].GetCertificateAssociationData())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> Eui48Record
						eui48RecordOptions, isEui48Record := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_Eui48Record)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", eui48RecordOptions))
						if isEui48Record {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record = &RRSetEui48Record{}

							// resource specToModelNested RequiredStringField --> Value
							if eui48RecordOptions.Eui48Record.GetValue() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record.Value = types.StringValue(eui48RecordOptions.Eui48Record.GetValue())
							}
							// resource specToModelNested StringField --> Name
							if eui48RecordOptions.Eui48Record.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui48Record.Name = types.StringValue(eui48RecordOptions.Eui48Record.GetName())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> Eui64Record
						eui64RecordOptions, isEui64Record := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_Eui64Record)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", eui64RecordOptions))
						if isEui64Record {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record = &RRSetEui64Record{}

							// resource specToModelNested StringField --> Name
							if eui64RecordOptions.Eui64Record.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record.Name = types.StringValue(eui64RecordOptions.Eui64Record.GetName())
							}
							// resource specToModelNested RequiredStringField --> Value
							if eui64RecordOptions.Eui64Record.GetValue() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.Eui64Record.Value = types.StringValue(eui64RecordOptions.Eui64Record.GetValue())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> CdsRecord
						cdsRecordOptions, isCdsRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_CdsRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", cdsRecordOptions))
						if isCdsRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord = &RRSetCdsRecord{}

							// resource specToModelNested StringField --> Name
							if cdsRecordOptions.CdsRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Name = types.StringValue(cdsRecordOptions.CdsRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range cdsRecordOptions.CdsRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredUintField --> KeyTag
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].KeyTag = types.Int32Value(int32(cdsRecordOptions.CdsRecord.GetValues()[iii].GetKeyTag()))
								// resource specToModelNested RequiredEnumField --> DsKeyAlgorithm
								if cdsRecordOptions.CdsRecord.GetValues()[iii].GetDsKeyAlgorithm().String() != "" {
									stringValue := types.StringValue(cdsRecordOptions.CdsRecord.GetValues()[iii].GetDsKeyAlgorithm().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DsKeyAlgorithm = listValue
								}
								// resource specToModelNested RequiredOneOfField --> DigestChoice
								digestChoiceChoices := cdsRecordOptions.CdsRecord.GetValues()[iii].GetDigestChoice()
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice = &DSRecordValueDigestChoice{}

								// resource specToModelNested PointerOfOneOfField --> Sha1Digest
								sha1DigestOptions, isSha1Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha1Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha1DigestOptions))
								if isSha1Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha1Digest = &DSRecordValueSha1Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha1DigestOptions.Sha1Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha1Digest.Digest = types.StringValue(sha1DigestOptions.Sha1Digest.GetDigest())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha256Digest
								sha256DigestOptions, isSha256Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha256Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha256DigestOptions))
								if isSha256Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha256Digest = &DSRecordValueSha256Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha256DigestOptions.Sha256Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha256Digest.Digest = types.StringValue(sha256DigestOptions.Sha256Digest.GetDigest())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha384Digest
								sha384DigestOptions, isSha384Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha384Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha384DigestOptions))
								if isSha384Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha384Digest = &DSRecordValueSha384Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha384DigestOptions.Sha384Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CdsRecord.Values[iii].DigestChoice.Sha384Digest.Digest = types.StringValue(sha384DigestOptions.Sha384Digest.GetDigest())
									}
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> CaaRecord
						caaRecordOptions, isCaaRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_CaaRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", caaRecordOptions))
						if isCaaRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord = &RRSetCaaRecord{}

							// resource specToModelNested StringField --> Name
							if caaRecordOptions.CaaRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Name = types.StringValue(caaRecordOptions.CaaRecord.GetName())
							}
							// resource specToModelNested ArrayOfPointerField --> Values
							if len(caaRecordOptions.CaaRecord.GetValues()) > 0 {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values = make([]DnsZoneCertificationAuthorityAuthorization, len(caaRecordOptions.CaaRecord.GetValues()))
								for iii := range caaRecordOptions.CaaRecord.GetValues() {

									// resource specToModelNested StringField --> Tag
									if caaRecordOptions.CaaRecord.GetValues()[iii].GetTag() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Tag = types.StringValue(caaRecordOptions.CaaRecord.GetValues()[iii].GetTag())
									}
									// resource specToModelNested StringField --> Value
									if caaRecordOptions.CaaRecord.GetValues()[iii].GetValue() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Value = types.StringValue(caaRecordOptions.CaaRecord.GetValues()[iii].GetValue())
									}
									// resource specToModelNested UintField --> Flags
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CaaRecord.Values[iii].Flags = types.Int32Value(int32(caaRecordOptions.CaaRecord.GetValues()[iii].GetFlags()))
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> NsRecord
						nsRecordOptions, isNsRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_NsRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", nsRecordOptions))
						if isNsRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord = &RRSetNsRecord{}

							// resource specToModelNested StringField --> Name
							if nsRecordOptions.NsRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord.Name = types.StringValue(nsRecordOptions.NsRecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(nsRecordOptions.NsRecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, nsRecordOptions.NsRecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.NsRecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> AfsdbRecord
						afsdbRecordOptions, isAfsdbRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_AfsdbRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", afsdbRecordOptions))
						if isAfsdbRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord = &RRSetAfsdbRecord{}

							// resource specToModelNested StringField --> Name
							if afsdbRecordOptions.AfsdbRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Name = types.StringValue(afsdbRecordOptions.AfsdbRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range afsdbRecordOptions.AfsdbRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredEnumField --> Subtype
								if afsdbRecordOptions.AfsdbRecord.GetValues()[iii].GetSubtype().String() != "" {
									stringValue := types.StringValue(afsdbRecordOptions.AfsdbRecord.GetValues()[iii].GetSubtype().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values[iii].Subtype = listValue
								}
								// resource specToModelNested RequiredStringField --> Hostname
								if afsdbRecordOptions.AfsdbRecord.GetValues()[iii].GetHostname() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AfsdbRecord.Values[iii].Hostname = types.StringValue(afsdbRecordOptions.AfsdbRecord.GetValues()[iii].GetHostname())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> TxtRecord
						txtRecordOptions, isTxtRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_TxtRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", txtRecordOptions))
						if isTxtRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord = &RRSetTxtRecord{}

							// resource specToModelNested StringField --> Name
							if txtRecordOptions.TxtRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord.Name = types.StringValue(txtRecordOptions.TxtRecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(txtRecordOptions.TxtRecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, txtRecordOptions.TxtRecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.TxtRecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> ARecord
						aRecordOptions, isARecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_ARecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", aRecordOptions))
						if isARecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord = &RRSetARecord{}

							// resource specToModelNested StringField --> Name
							if aRecordOptions.ARecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord.Name = types.StringValue(aRecordOptions.ARecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(aRecordOptions.ARecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, aRecordOptions.ARecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.ARecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> AliasRecord
						aliasRecordOptions, isAliasRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_AliasRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", aliasRecordOptions))
						if isAliasRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AliasRecord = &RRSetAliasRecord{}

							// resource specToModelNested StringField --> Value
							if aliasRecordOptions.AliasRecord.GetValue() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AliasRecord.Value = types.StringValue(aliasRecordOptions.AliasRecord.GetValue())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> LbRecord
						lbRecordOptions, isLbRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_LbRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", lbRecordOptions))
						if isLbRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord = &RRSetLbRecord{}

							// resource specToModelNested StringField --> Name
							if lbRecordOptions.LbRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Name = types.StringValue(lbRecordOptions.LbRecord.GetName())
							}
							// resource specToModelNested PointerField --> Value
							if lbRecordOptions.LbRecord.GetValue() != nil {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value = &ObjectRefTypeModel{}

								// resource specToModelNested StringField --> Tenant
								if lbRecordOptions.LbRecord.GetValue().GetTenant() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Tenant = types.StringValue(lbRecordOptions.LbRecord.GetValue().GetTenant())
								}
								// resource specToModelNested StringField --> Namespace
								if lbRecordOptions.LbRecord.GetValue().GetNamespace() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Namespace = types.StringValue(lbRecordOptions.LbRecord.GetValue().GetNamespace())
								}
								// resource specToModelNested StringField --> Name
								if lbRecordOptions.LbRecord.GetValue().GetName() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LbRecord.Value.Name = types.StringValue(lbRecordOptions.LbRecord.GetValue().GetName())
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> SshfpRecord
						sshfpRecordOptions, isSshfpRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_SshfpRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sshfpRecordOptions))
						if isSshfpRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord = &RRSetSshfpRecord{}

							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range sshfpRecordOptions.SshfpRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredEnumField --> Algorithm
								if sshfpRecordOptions.SshfpRecord.GetValues()[iii].GetAlgorithm().String() != "" {
									stringValue := types.StringValue(sshfpRecordOptions.SshfpRecord.GetValues()[iii].GetAlgorithm().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].Algorithm = listValue
								}
								// resource specToModelNested RequiredOneOfField --> FingerprintType
								fingerprintTypeChoiceChoices := sshfpRecordOptions.SshfpRecord.GetValues()[iii].GetFingerprintType()
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice = &SSHFPRecordValueFingerprintType{}

								// resource specToModelNested PointerOfOneOfField --> Sha1Fingerprint
								sha1FingerprintOptions, isSha1Fingerprint := fingerprintTypeChoiceChoices.(*ves_io_schema_dns_zone.SSHFPRecordValue_Sha1Fingerprint)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha1FingerprintOptions))
								if isSha1Fingerprint {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha1Fingerprint = &SSHFPRecordValueSha1Fingerprint{}

									// resource specToModelNested RequiredStringField --> Fingerprint
									if sha1FingerprintOptions.Sha1Fingerprint.GetFingerprint() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha1Fingerprint.Fingerprint = types.StringValue(sha1FingerprintOptions.Sha1Fingerprint.GetFingerprint())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha256Fingerprint
								sha256FingerprintOptions, isSha256Fingerprint := fingerprintTypeChoiceChoices.(*ves_io_schema_dns_zone.SSHFPRecordValue_Sha256Fingerprint)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha256FingerprintOptions))
								if isSha256Fingerprint {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha256Fingerprint = &SSHFPRecordValueSha256Fingerprint{}

									// resource specToModelNested RequiredStringField --> Fingerprint
									if sha256FingerprintOptions.Sha256Fingerprint.GetFingerprint() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Values[iii].FingerprintTypeChoice.Sha256Fingerprint.Fingerprint = types.StringValue(sha256FingerprintOptions.Sha256Fingerprint.GetFingerprint())
									}
								}
							}
							// resource specToModelNested StringField --> Name
							if sshfpRecordOptions.SshfpRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.SshfpRecord.Name = types.StringValue(sshfpRecordOptions.SshfpRecord.GetName())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> CertRecord
						certRecordOptions, isCertRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_CertRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", certRecordOptions))
						if isCertRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord = &RRSetCertRecord{}

							// resource specToModelNested StringField --> Name
							if certRecordOptions.CertRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Name = types.StringValue(certRecordOptions.CertRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range certRecordOptions.CertRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredUintField --> CertKeyTag
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].CertKeyTag = types.Int32Value(int32(certRecordOptions.CertRecord.GetValues()[iii].GetCertKeyTag()))
								// resource specToModelNested RequiredEnumField --> Algorithm
								if certRecordOptions.CertRecord.GetValues()[iii].GetAlgorithm().String() != "" {
									stringValue := types.StringValue(certRecordOptions.CertRecord.GetValues()[iii].GetAlgorithm().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].Algorithm = listValue
								}
								// resource specToModelNested RequiredStringField --> Certificate
								if certRecordOptions.CertRecord.GetValues()[iii].GetCertificate() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].Certificate = types.StringValue(certRecordOptions.CertRecord.GetValues()[iii].GetCertificate())
								}
								// resource specToModelNested RequiredEnumField --> CertType
								if certRecordOptions.CertRecord.GetValues()[iii].GetCertType().String() != "" {
									stringValue := types.StringValue(certRecordOptions.CertRecord.GetValues()[iii].GetCertType().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CertRecord.Values[iii].CertType = listValue
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> CnameRecord
						cnameRecordOptions, isCnameRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_CnameRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", cnameRecordOptions))
						if isCnameRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord = &RRSetCnameRecord{}

							// resource specToModelNested StringField --> Value
							if cnameRecordOptions.CnameRecord.GetValue() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord.Value = types.StringValue(cnameRecordOptions.CnameRecord.GetValue())
							}
							// resource specToModelNested RequiredStringField --> Name
							if cnameRecordOptions.CnameRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.CnameRecord.Name = types.StringValue(cnameRecordOptions.CnameRecord.GetName())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> PtrRecord
						ptrRecordOptions, isPtrRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_PtrRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", ptrRecordOptions))
						if isPtrRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord = &RRSetPtrRecord{}

							// resource specToModelNested StringField --> Name
							if ptrRecordOptions.PtrRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord.Name = types.StringValue(ptrRecordOptions.PtrRecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(ptrRecordOptions.PtrRecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, ptrRecordOptions.PtrRecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.PtrRecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> AaaaRecord
						aaaaRecordOptions, isAaaaRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_AaaaRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", aaaaRecordOptions))
						if isAaaaRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord = &RRSetAaaaRecord{}

							// resource specToModelNested StringField --> Name
							if aaaaRecordOptions.AaaaRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord.Name = types.StringValue(aaaaRecordOptions.AaaaRecord.GetName())
							} // resource specToModelNested RequiredArrayOfStringField --> Values
							if len(aaaaRecordOptions.AaaaRecord.GetValues()) > 0 {
								listValue, diags := types.ListValueFrom(ctx, types.StringType, aaaaRecordOptions.AaaaRecord.GetValues())
								response.Diagnostics.Append(diags...)
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.AaaaRecord.Values = listValue
							}
						}
						// resource specToModelNested PointerOfOneOfField --> LocRecord
						locRecordOptions, isLocRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_LocRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", locRecordOptions))
						if isLocRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord = &RRSetLocRecord{}

							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range locRecordOptions.LocRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested FloatField --> LatitudeSecond
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeSecond = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeSecond()))
								// resource specToModelNested RequiredIntField --> LongitudeDegree
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeDegree = types.Int32Value(locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeDegree())
								// resource specToModelNested IntField --> LongitudeMinute
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeMinute = types.Int32Value(locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeMinute())
								// resource specToModelNested FloatField --> LongitudeSecond
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeSecond = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeSecond()))
								// resource specToModelNested FloatField --> HorizontalPrecision
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].HorizontalPrecision = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetHorizontalPrecision()))
								// resource specToModelNested RequiredIntField --> LatitudeDegree
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeDegree = types.Int32Value(locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeDegree())
								// resource specToModelNested RequiredEnumField --> LatitudeHemisphere
								if locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeHemisphere().String() != "" {
									stringValue := types.StringValue(locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeHemisphere().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeHemisphere = listValue
								}
								// resource specToModelNested RequiredEnumField --> LongitudeHemisphere
								if locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeHemisphere().String() != "" {
									stringValue := types.StringValue(locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeHemisphere().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeHemisphere = listValue
								}
								// resource specToModelNested RequiredFloatField --> Altitude
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].Altitude = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetAltitude()))
								// resource specToModelNested FloatField --> LocationDiameter
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LocationDiameter = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetLocationDiameter()))
								// resource specToModelNested FloatField --> VerticalPrecision
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].VerticalPrecision = types.Float32Value(float32(locRecordOptions.LocRecord.GetValues()[iii].GetVerticalPrecision()))
								// resource specToModelNested IntField --> LatitudeMinute
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LatitudeMinute = types.Int32Value(locRecordOptions.LocRecord.GetValues()[iii].GetLatitudeMinute())
							}
							// resource specToModelNested StringField --> Name
							if locRecordOptions.LocRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Name = types.StringValue(locRecordOptions.LocRecord.GetName())
							}
						}
						// resource specToModelNested PointerOfOneOfField --> DsRecord
						dsRecordOptions, isDsRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_DsRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", dsRecordOptions))
						if isDsRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord = &RRSetDsRecord{}

							// resource specToModelNested StringField --> Name
							if dsRecordOptions.DsRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Name = types.StringValue(dsRecordOptions.DsRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range dsRecordOptions.DsRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested RequiredUintField --> KeyTag
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].KeyTag = types.Int32Value(int32(dsRecordOptions.DsRecord.GetValues()[iii].GetKeyTag()))
								// resource specToModelNested RequiredEnumField --> DsKeyAlgorithm
								if dsRecordOptions.DsRecord.GetValues()[iii].GetDsKeyAlgorithm().String() != "" {
									stringValue := types.StringValue(dsRecordOptions.DsRecord.GetValues()[iii].GetDsKeyAlgorithm().String())
									stringSlice := []types.String{stringValue}
									listValue, diags := types.ListValueFrom(ctx, types.StringType, stringSlice)
									response.Diagnostics.Append(diags...)
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DsKeyAlgorithm = listValue
								}
								// resource specToModelNested RequiredOneOfField --> DigestChoice
								digestChoiceChoices := dsRecordOptions.DsRecord.GetValues()[iii].GetDigestChoice()
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice = &DSRecordValueDigestChoice{}

								// resource specToModelNested PointerOfOneOfField --> Sha1Digest
								sha1DigestOptions, isSha1Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha1Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha1DigestOptions))
								if isSha1Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha1Digest = &DSRecordValueSha1Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha1DigestOptions.Sha1Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha1Digest.Digest = types.StringValue(sha1DigestOptions.Sha1Digest.GetDigest())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha256Digest
								sha256DigestOptions, isSha256Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha256Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha256DigestOptions))
								if isSha256Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha256Digest = &DSRecordValueSha256Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha256DigestOptions.Sha256Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha256Digest.Digest = types.StringValue(sha256DigestOptions.Sha256Digest.GetDigest())
									}
								}
								// resource specToModelNested PointerOfOneOfField --> Sha384Digest
								sha384DigestOptions, isSha384Digest := digestChoiceChoices.(*ves_io_schema_dns_zone.DSRecordValue_Sha384Digest)
								tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", sha384DigestOptions))
								if isSha384Digest {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha384Digest = &DSRecordValueSha384Digest{}

									// resource specToModelNested RequiredStringField --> Digest
									if sha384DigestOptions.Sha384Digest.GetDigest() != "" {
										data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.DsRecord.Values[iii].DigestChoice.Sha384Digest.Digest = types.StringValue(sha384DigestOptions.Sha384Digest.GetDigest())
									}
								}
							}
						}
						// resource specToModelNested PointerOfOneOfField --> MxRecord
						mxRecordOptions, isMxRecord := typeRecordSetChoiceChoices.(*ves_io_schema_dns_zone.RRSet_MxRecord)
						tflog.Info(ctx, fmt.Sprintf("PointerOfOneOfField: %v", mxRecordOptions))
						if isMxRecord {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord = &RRSetMxRecord{}

							// resource specToModelNested StringField --> Name
							if mxRecordOptions.MxRecord.GetName() != "" {
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Name = types.StringValue(mxRecordOptions.MxRecord.GetName())
							}
							// resource specToModelNested RequiredArrayOfPointerField --> Values
							for iii := range mxRecordOptions.MxRecord.GetValues() {
								tflog.Info(ctx, fmt.Sprintf("%d", iii))

								// resource specToModelNested StringField --> Domain
								if mxRecordOptions.MxRecord.GetValues()[iii].GetDomain() != "" {
									data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values[iii].Domain = types.StringValue(mxRecordOptions.MxRecord.GetValues()[iii].GetDomain())
								}
								// resource specToModelNested UintField --> Priority
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.MxRecord.Values[iii].Priority = types.Int32Value(int32(mxRecordOptions.MxRecord.GetValues()[iii].GetPriority()))
							}
						}
						// resource specToModelNested StringField --> Description
						if primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()[ii].GetDescription() != "" {
							data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].Description = types.StringValue(primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet()[ii].GetDescription())
						}
					}
				}
			}
		}
		// resource specToModelNested UintField --> Serial
		data.DnsTypeChoice.Primary.Serial = types.Int32Value(int32(primaryOptions.Primary.GetSerial()))
	}
	// resource SpecToModelRoot StringField --> SubscriptionId
	if specData.GetSubscriptionId() != "" {
		data.SubscriptionId = types.StringValue(specData.GetSubscriptionId())
	}

	// Save updated data into Terraform state
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (r *ResourceF5XCDnsZone) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var data ResourceF5XCDnsZoneModel

	// Read Terraform prior state data into the model
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	opts := []vesapi.CallOpt{
		vesapi.WithFailIfReferred(),
	}

	err := r.client.DeleteObject(ctx, ves_io_schema_dns_zone.ObjectType, data.Namespace.ValueString(), data.Name.ValueString(), opts...)
	if err != nil {
		response.Diagnostics.AddError(
			"Client Error", fmt.Sprintf("Unable to delete %s, got error: %s", "dns_zone", err),
		)
		return
	}
}

func (r *ResourceF5XCDnsZone) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
