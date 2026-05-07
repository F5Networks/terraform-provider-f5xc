//
// Copyright (c) 2026 F5, Inc. All rights reserved.
//

package manual

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/client"
	"gopkg.volterra.us/terraform-provider-f5xc/f5xc/terraform/tpf/custom_types/timetypes"

	ves_io_schema "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema"
	ves_io_schema_dns_zone "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/dns_zone"
)

// DataSourceF5XCDnsZone defines the data source implementation.
type DataSourceF5XCDnsZone struct {
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

type NsRecordModel struct {
	Name   types.String `tfsdk:"name"`
	Values types.List   `tfsdk:"values"`
}

type DefaultRrSetGroupModel struct {
	Ttl         types.Int64  `tfsdk:"ttl"`
	NsRecord    types.Object `tfsdk:"ns_record"`
	Description types.String `tfsdk:"description"`
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

// DataSourceF5XCDnsZoneModel describes the data source data model.
type DataSourceF5XCDnsZoneModel struct {
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
var _ datasource.DataSource = &DataSourceF5XCDnsZone{}

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

func NewDataSourceF5XCDnsZone() datasource.DataSource {
	return &DataSourceF5XCDnsZone{}
}

func (d *DataSourceF5XCDnsZone) Metadata(ctx context.Context, request datasource.MetadataRequest, response *datasource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_dns_zone"
}

func (r *DataSourceF5XCDnsZone) Schema(ctx context.Context, request datasource.SchemaRequest, response *datasource.SchemaResponse) {
	response.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "DnsZone data source",
		Description:         "DnsZone data source",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "name This is the name of configuration object. It has to be unique within the namespace. It can only be specified during create API and cannot be changed during replace API. The value of name has to follow DNS-1035 format.",
				Required:    false,
				Optional:    true,
				Computed:    true,
			},
			"namespace": schema.StringAttribute{
				Description: "namespace This defines the workspace within which each the configuration object is to be created. Must be a DNS_LABEL format. For a namespace object itself, namespace value will be ''",
				Required:    false,
				Optional:    true,
				Computed:    true,
			},
			"uid": schema.StringAttribute{
				Description: "uid uid is the unique in time and space value for this object. Object create will fail if provided by the client and the value exists in the system. Typically generated by the server on successful creation of an object and is not allowed to change once populated. Shadowed by SystemObjectMeta's uid field.",
				Required:    false,
				Optional:    false,
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: "labels Map of string keys and values that can be used to organize and categorize (scope and select) objects as chosen by the user. Values specified here will be used by selector expression",
				Required:    false,
				Optional:    false,
				Computed:    true,
				ElementType: types.StringType,
			},
			"annotations": schema.MapAttribute{
				Description: "annotations Annotations is an unstructured key value map stored with a resource that may be set by external tools to store and retrieve arbitrary metadata. They are not queryable and should be preserved when modifying objects.",
				Required:    false,
				Optional:    false,
				Computed:    true,
				ElementType: types.StringType,
			},
			"description": schema.StringAttribute{
				Description: "description Human readable description for the object",
				Required:    false,
				Optional:    false,
				Computed:    true,
			},
			"service_instance_id": schema.StringAttribute{
				Description: "Service Instance ID Service Instance ID is an internal identifier for backend dns services",
				Required:    false,
				Optional:    false,
				Computed:    true,
			},
			"num_of_dns_records": schema.Int32Attribute{
				Description: "Number of DNS records Number of DNS records displays number of dns records for each zone",
				Required:    false,
				Optional:    false,
				Computed:    true,
			},
			"dns_type_choice": schema.SingleNestedAttribute{
				Description: "Choice to DNS zone type Select type of zone whether its primary or secondary",
				Required:    false,
				Optional:    false,
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"secondary": schema.SingleNestedAttribute{
						Description: "Secondary DNS Configuration Secondary DNS",
						Required:    false,
						Optional:    false,
						Computed:    true,
						Attributes: map[string]schema.Attribute{
							"tsig_key_name": schema.StringAttribute{
								Description: "TSIG Key Name TSIG key name as used in TSIG protocol extension",
								Required:    false,
								Optional:    false,
								Computed:    true,
							},
							"tsig_key_algorithm": schema.ListAttribute{
								Description: "TSIG Key Algorithm TSIG key value must be compatible with the specified algorithm",
								ElementType: types.StringType,
								Required:    false,
								Optional:    false,
								Computed:    true,
							},
							"tsig_key_value": schema.SingleNestedAttribute{
								Description: "TSIG Key value",
								Required:    false,
								Optional:    false,
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"secret_info_choice": schema.SingleNestedAttribute{
										Description: "SecretInfoOneof Secret can be one of the following types - * Blindfold Secret - Secret is managed by F5XC Secret Management System * Clear Secret - Secret is not encrypted",
										Required:    false,
										Optional:    false,
										Computed:    true,
										Attributes: map[string]schema.Attribute{
											"blindfold_secret_info": schema.SingleNestedAttribute{
												Description: "Blindfold Secret Blindfold Secret is used for the secrets managed by F5XC Secret Management Service",
												Required:    false,
												Optional:    false,
												Computed:    true,
												Attributes: map[string]schema.Attribute{
													"decryption_provider": schema.StringAttribute{
														Description: "Decryption Provider Name of the Secret Management Access object that contains information about the backend Secret Management service.",
														Required:    false,
														Optional:    false,
														Computed:    true,
													},
													"store_provider": schema.StringAttribute{
														Description: "Store Provider Name of the Secret Management Access object that contains information about the store to get encrypted bytes This field needs to be provided only if the url scheme is not string:///",
														Required:    false,
														Optional:    false,
														Computed:    true,
													},
													"location": schema.StringAttribute{
														Description: "Location Location is the uri_ref. It could be in url format for string:/// Or it could be a path if the store provider is an http/https location",
														Required:    false,
														Optional:    false,
														Computed:    true,
													},
												},
											},
											"clear_secret_info": schema.SingleNestedAttribute{
												Description: "Clear Secret Clear Secret is used for the secrets that are not encrypted",
												Required:    false,
												Optional:    false,
												Computed:    true,
												Attributes: map[string]schema.Attribute{
													"provider_dns_zone": schema.StringAttribute{
														Description: "Provider Name of the Secret Management Access object that contains information about the store to get encrypted bytes This field needs to be provided only if the url scheme is not string:///",
														Required:    false,
														Optional:    false,
														Computed:    true,
													},
													"url": schema.StringAttribute{
														Description: "URL URL of the secret. Currently supported URL schemes is string:///. For string:/// scheme, Secret needs to be encoded Base64 format. When asked for this secret, caller will get Secret bytes after Base64 decoding.",
														Required:    false,
														Optional:    false,
														Computed:    true,
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
								Required:    false,
								Optional:    false,
								Computed:    true,
							},
						},
					},
					"primary": schema.SingleNestedAttribute{
						Description: "Primary DNS Configuration Primary DNS",
						Required:    false,
						Optional:    false,
						Computed:    true,
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
								Required:    false,
								Optional:    false,
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"mode_choice": schema.SingleNestedAttribute{
										Description: "DNSSEC Mode DNSSEC Mode of operation",
										Required:    false,
										Optional:    false,
										Computed:    true,
										Attributes: map[string]schema.Attribute{
											"enable": schema.BoolAttribute{
												Description: "Enable DNSSEC enable",
												Required:    false,
												Optional:    false,
												Computed:    true,
											},
										},
									},
								},
							},
							"allow_http_lb_managed_records": schema.BoolAttribute{
								Description: "Allow HTTP Load Balancer Managed Records Option to allow user-created HTTP, TCP, and CDN load balancer related resource records to be automatically managed in a protected RRset.",
								Required:    false,
								Optional:    false,
								Computed:    true,
							},
							"soa_record_parameters_choice": schema.SingleNestedAttribute{
								Description: "SOA Record Parameters Start Of Authority (SOA) Record Properties",
								Required:    false,
								Optional:    false,
								Computed:    true,
								Attributes: map[string]schema.Attribute{
									"soa_parameters": schema.SingleNestedAttribute{
										Description: "SOARecordParameterConfig Configure SOA Record parameters",
										Required:    false,
										Optional:    false,
										Computed:    true,
										Attributes: map[string]schema.Attribute{
											"refresh": schema.Int32Attribute{
												Description: "Refresh (in seconds) refresh value indicates when secondary nameservers should query for the SOA record to detect zone changes",
												Required:    false,
												Optional:    false,
												Computed:    true,
											},
											"retry": schema.Int32Attribute{
												Description: "Retry (in seconds) retry value indicates when secondary nameservers should retry to request the serial number if primary does not respond",
												Required:    false,
												Optional:    false,
												Computed:    true,
											},
											"expire": schema.Int32Attribute{
												Description: "Expire (in seconds) expire value indicates when secondary nameservers should stop answering request for this zone if primary does not respond",
												Required:    false,
												Optional:    false,
												Computed:    true,
											},
											"negative_ttl": schema.Int32Attribute{
												Description: "Negative TTL (in seconds) negative ttl value indicates how long to cache non-existent resource record for this zone",
												Required:    false,
												Optional:    false,
												Computed:    true,
											},
											"ttl": schema.Int32Attribute{
												Description: "TTL SOA record time to live (in seconds)",
												Required:    false,
												Optional:    false,
												Computed:    true,
											},
										},
									},
									"default_soa_parameters": schema.BoolAttribute{
										Description: "Default parameters Use default parameters",
										Required:    false,
										Optional:    false,
										Computed:    true,
									},
								},
							},
							"rr_set_group": schema.ListNestedAttribute{
								Description: "Record Sets Group",
								Required:    false,
								Optional:    false,
								Computed:    true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"metadata": schema.SingleNestedAttribute{
											Description: "metadata",
											Required:    false,
											Optional:    false,
											Computed:    true,
											Attributes: map[string]schema.Attribute{
												"name": schema.StringAttribute{
													Description: "name This is the name of the message. The value of name has to follow DNS-1035 format.",
													Required:    false,
													Optional:    false,
													Computed:    true,
												},
												"description": schema.StringAttribute{
													Description: "description Human readable description.",
													Required:    false,
													Optional:    false,
													Computed:    true,
												},
											},
										},
										"rr_set": schema.ListNestedAttribute{
											Description: "Record Sets Collection of DNS resource record sets",
											Required:    false,
											Optional:    false,
											Computed:    true,
											NestedObject: schema.NestedAttributeObject{
												Attributes: map[string]schema.Attribute{
													"ttl": schema.Int32Attribute{
														Description: "TTL",
														Required:    false,
														Optional:    false,
														Computed:    true,
													},
													"type_record_set_choice": schema.SingleNestedAttribute{
														Description: "Record Set",
														Required:    false,
														Optional:    false,
														Computed:    true,
														Attributes: map[string]schema.Attribute{
															"naptr_record": schema.SingleNestedAttribute{
																Description: "NAPTR Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name NAPTR Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "NAPTR Values",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"order": schema.Int32Attribute{
																					Description: "Order Order in which the NAPTR records must be processed. A lower number indicates a higher preference.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"preference": schema.Int32Attribute{
																					Description: "Preference Preference when records have the same order. A lower number indicates a higher preference.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"flags": schema.StringAttribute{
																					Description: "Flags Flag to control aspects of the rewriting and interpretation of the fields in the record. At this time only four flags, S/A/U/P, are defined.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"service": schema.StringAttribute{
																					Description: "Protocol Resolution Service Specifies the service(s) available down this rewrite path.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"regexp": schema.StringAttribute{
																					Description: "Regular Expression Regular expression to construct the next domain name to lookup.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"replacement": schema.StringAttribute{
																					Description: "Replacement The next NAME to query for NAPTR, SRV, or address records depending on the value of the flags field.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																			},
																		},
																	},
																},
															},
															"srv_record": schema.SingleNestedAttribute{
																Description: "SRV Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name SRV Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "SRV Value",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"priority": schema.Int32Attribute{
																					Description: "Priority Priority of the target. A lower number indicates a higher preference.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"weight": schema.Int32Attribute{
																					Description: "Weight Weight of the target. A higher number indicates a higher preference.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"port": schema.Int32Attribute{
																					Description: "Port Port on which the service can be found",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"target": schema.StringAttribute{
																					Description: "Target Hostname of the machine providing the service",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																			},
																		},
																	},
																},
															},
															"tlsa_record": schema.SingleNestedAttribute{
																Description: "TLSA Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name TLSA Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "TLSA Value",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"certificate_usage": schema.ListAttribute{
																					Description: "Certificate Usage TLSA Record Certificate Usage.",
																					ElementType: types.StringType,
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"selector": schema.ListAttribute{
																					Description: "Selector TLSA Record Selector.",
																					ElementType: types.StringType,
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"matching_type": schema.ListAttribute{
																					Description: "Matching Type TLSA Record Matching Type.",
																					ElementType: types.StringType,
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"certificate_association_data": schema.StringAttribute{
																					Description: "Certificate Association Data The actual data to be matched given the settings of the other fields.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																			},
																		},
																	},
																},
															},
															"eui48_record": schema.SingleNestedAttribute{
																Description: "EUI48 Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"value": schema.StringAttribute{
																		Description: "EUI48 Identifier A valid eui48 identifier, for example: 01-23-45-67-89-ab",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"name": schema.StringAttribute{
																		Description: "Record Name EUI48 Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																},
															},
															"eui64_record": schema.SingleNestedAttribute{
																Description: "EUI64 Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name EUI64 Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"value": schema.StringAttribute{
																		Description: "EUI64 Identifier A valid EUI64 identifier, for example: 01-23-45-67-89-ab-cd-ef",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																},
															},
															"cds_record": schema.SingleNestedAttribute{
																Description: "CDS Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name CDS Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "DS Value",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"key_tag": schema.Int32Attribute{
																					Description: "Key Tag A short numeric value which can help quickly identify the referenced DNSKEY-record.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"ds_key_algorithm": schema.ListAttribute{
																					Description: "DS Key Algorithm DS key value must be compatible with the specified algorithm.",
																					ElementType: types.StringType,
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"digest_choice": schema.SingleNestedAttribute{
																					Description: "Digest",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																					Attributes: map[string]schema.Attribute{
																						"sha1_digest": schema.SingleNestedAttribute{
																							Description: "SHA1 Digest",
																							Required:    false,
																							Optional:    false,
																							Computed:    true,
																							Attributes: map[string]schema.Attribute{
																								"digest": schema.StringAttribute{
																									Description: "Digest The 'digest' is the DS key and the actual contents of the DS record.",
																									Required:    false,
																									Optional:    false,
																									Computed:    true,
																								},
																							},
																						},
																						"sha256_digest": schema.SingleNestedAttribute{
																							Description: "SHA256 Digest",
																							Required:    false,
																							Optional:    false,
																							Computed:    true,
																							Attributes: map[string]schema.Attribute{
																								"digest": schema.StringAttribute{
																									Description: "Digest The 'digest' is the DS key and the actual contents of the DS record.",
																									Required:    false,
																									Optional:    false,
																									Computed:    true,
																								},
																							},
																						},
																						"sha384_digest": schema.SingleNestedAttribute{
																							Description: "SHA384 Digest",
																							Required:    false,
																							Optional:    false,
																							Computed:    true,
																							Attributes: map[string]schema.Attribute{
																								"digest": schema.StringAttribute{
																									Description: "Digest The 'digest' is the DS key and the actual contents of the DS record.",
																									Required:    false,
																									Optional:    false,
																									Computed:    true,
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
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name CAA Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "CAA Record Value",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"tag": schema.StringAttribute{
																					Description: "Certification authority authorization tag",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"value": schema.StringAttribute{
																					Description: "Certification authority authorization value",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"flags": schema.Int32Attribute{
																					Description: "Certification authority authorization flags This flag should be an integer between 0 and 255.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																			},
																		},
																	},
																},
															},
															"ns_record": schema.SingleNestedAttribute{
																Description: "NS Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name NS Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"values": schema.ListAttribute{
																		Description: "Name-servers",
																		ElementType: types.StringType,
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																},
															},
															"afsdb_record": schema.SingleNestedAttribute{
																Description: "AFSDB Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name AFSDB Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "AFSDB Value",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"subtype": schema.ListAttribute{
																					Description: "Subtype AFSDB Record Subtype.",
																					ElementType: types.StringType,
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"hostname": schema.StringAttribute{
																					Description: "Hostname Server name of the AFS cell database server or the DCE name server.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																			},
																		},
																	},
																},
															},
															"txt_record": schema.SingleNestedAttribute{
																Description: "TXT Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name TXT Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"values": schema.ListAttribute{
																		Description: "Text",
																		ElementType: types.StringType,
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																},
															},
															"a_record": schema.SingleNestedAttribute{
																Description: "A Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name A Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"values": schema.ListAttribute{
																		Description: "IPv4 addresses A valid IPv4 address, for example: 1.1.1.1",
																		ElementType: types.StringType,
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																},
															},
															"alias_record": schema.SingleNestedAttribute{
																Description: "ALIAS Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"value": schema.StringAttribute{
																		Description: "Domain A valid domain name, for example: example.com",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																},
															},
															"lb_record": schema.SingleNestedAttribute{
																Description: "DNS LB Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name Load Balancer record name (except for SRV DNS Load balancer record) should be a simple record name and not a subdomain of a subdomain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"value": schema.SingleNestedAttribute{
																		Description: "DNS LB Record",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																		Attributes: map[string]schema.Attribute{
																			"tenant": schema.StringAttribute{
																				Description: "",
																				Required:    false,
																				Optional:    false,
																				Computed:    true,
																			},
																			"namespace": schema.StringAttribute{
																				Description: "",
																				Required:    false,
																				Optional:    false,
																				Computed:    true,
																			},
																			"name": schema.StringAttribute{
																				Description: "",
																				Required:    false,
																				Optional:    false,
																				Computed:    true,
																			},
																		},
																	},
																},
															},
															"sshfp_record": schema.SingleNestedAttribute{
																Description: "SSHFP Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"values": schema.ListNestedAttribute{
																		Description: "SSHFP Value",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"algorithm": schema.ListAttribute{
																					Description: "SSHFP Algorithm Algorithm of the public key.",
																					ElementType: types.StringType,
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"fingerprint_type_choice": schema.SingleNestedAttribute{
																					Description: "Fingerprint Type Algorithm used to calculate the fingerprint of the public key and the hexadecimal representation of the hash result of the SSH key as text.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																					Attributes: map[string]schema.Attribute{
																						"sha1_fingerprint": schema.SingleNestedAttribute{
																							Description: "SHA1 Fingerprint",
																							Required:    false,
																							Optional:    false,
																							Computed:    true,
																							Attributes: map[string]schema.Attribute{
																								"fingerprint": schema.StringAttribute{
																									Description: "Fingerprint The 'fingerprint' is the DS key and the actual contents of the DS record.",
																									Required:    false,
																									Optional:    false,
																									Computed:    true,
																								},
																							},
																						},
																						"sha256_fingerprint": schema.SingleNestedAttribute{
																							Description: "SHA256 Digest",
																							Required:    false,
																							Optional:    false,
																							Computed:    true,
																							Attributes: map[string]schema.Attribute{
																								"fingerprint": schema.StringAttribute{
																									Description: "Fingerprint The 'fingerprint' is the DS key and the actual contents of the DS record.",
																									Required:    false,
																									Optional:    false,
																									Computed:    true,
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
																		Optional:    false,
																		Computed:    true,
																	},
																},
															},
															"cert_record": schema.SingleNestedAttribute{
																Description: "CERT Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name CERT Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "CERT Value",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"cert_key_tag": schema.Int32Attribute{
																					Description: "Key Tag",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"algorithm": schema.ListAttribute{
																					Description: "Algorithm",
																					ElementType: types.StringType,
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"certificate": schema.StringAttribute{
																					Description: "Certificate Certificate in base 64 format.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"cert_type": schema.ListAttribute{
																					Description: "Type",
																					ElementType: types.StringType,
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																			},
																		},
																	},
																},
															},
															"cname_record": schema.SingleNestedAttribute{
																Description: "CNAME Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"value": schema.StringAttribute{
																		Description: "Domain",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"name": schema.StringAttribute{
																		Description: "Record Name CName Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																},
															},
															"ptr_record": schema.SingleNestedAttribute{
																Description: "PTR Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name PTR Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"values": schema.ListAttribute{
																		Description: "Domain names",
																		ElementType: types.StringType,
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																},
															},
															"aaaa_record": schema.SingleNestedAttribute{
																Description: "AAAA Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name AAAA Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"values": schema.ListAttribute{
																		Description: "IPv6 addresses A valid IPv6 address, for example: 2001:0db8:85a3:0000:0000:8a2e:0370:7334",
																		ElementType: types.StringType,
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																},
															},
															"loc_record": schema.SingleNestedAttribute{
																Description: "LOC Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"values": schema.ListNestedAttribute{
																		Description: "LOC Values",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"latitude_second": schema.Float64Attribute{
																					Description: "Latitude second Latitude second, an decimal between 0 and 59.999, including 0 and 59.999",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"longitude_degree": schema.Int32Attribute{
																					Description: "Longitude degree Longitude degree, an integer between 0 and 180, including 0 and 180",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"longitude_minute": schema.Int32Attribute{
																					Description: "Longitude minute Longitude minute, an integer between 0 and 59, including 0 and 59",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"longitude_second": schema.Float64Attribute{
																					Description: "Longitude second Longitude second, an decimal between 0 and 59.999, including 0 and 59.999",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"horizontal_precision": schema.Float64Attribute{
																					Description: "Horizontal Precision Horizontal Precision in meters",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"latitude_degree": schema.Int32Attribute{
																					Description: "Latitude degree Latitude degree, an integer between 0 and 90, including 0 and 90",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"latitude_hemisphere": schema.ListAttribute{
																					Description: "Latitude hemisphere Latitude hemisphere, N or S",
																					ElementType: types.StringType,
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"longitude_hemisphere": schema.ListAttribute{
																					Description: "Longitude hemisphere Longitude hemisphere, E or W",
																					ElementType: types.StringType,
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
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
																					Optional:    false,
																					Computed:    true,
																				},
																				"vertical_precision": schema.Float64Attribute{
																					Description: "Vertical Precision Vertical Precision in meters",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"latitude_minute": schema.Int32Attribute{
																					Description: "Latitude minute Latitude minute, an integer between 0 and 59, including 0 and 59",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																			},
																		},
																	},
																	"name": schema.StringAttribute{
																		Description: "Record Name LOC Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																},
															},
															"ds_record": schema.SingleNestedAttribute{
																Description: "DS Record",
																Required:    false,
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name DS Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "DS Value",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"key_tag": schema.Int32Attribute{
																					Description: "Key Tag A short numeric value which can help quickly identify the referenced DNSKEY-record.",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"ds_key_algorithm": schema.ListAttribute{
																					Description: "DS Key Algorithm DS key value must be compatible with the specified algorithm.",
																					ElementType: types.StringType,
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"digest_choice": schema.SingleNestedAttribute{
																					Description: "Digest",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																					Attributes: map[string]schema.Attribute{
																						"sha1_digest": schema.SingleNestedAttribute{
																							Description: "SHA1 Digest",
																							Required:    false,
																							Optional:    false,
																							Computed:    true,
																							Attributes: map[string]schema.Attribute{
																								"digest": schema.StringAttribute{
																									Description: "Digest The 'digest' is the DS key and the actual contents of the DS record.",
																									Required:    false,
																									Optional:    false,
																									Computed:    true,
																								},
																							},
																						},
																						"sha256_digest": schema.SingleNestedAttribute{
																							Description: "SHA256 Digest",
																							Required:    false,
																							Optional:    false,
																							Computed:    true,
																							Attributes: map[string]schema.Attribute{
																								"digest": schema.StringAttribute{
																									Description: "Digest The 'digest' is the DS key and the actual contents of the DS record.",
																									Required:    false,
																									Optional:    false,
																									Computed:    true,
																								},
																							},
																						},
																						"sha384_digest": schema.SingleNestedAttribute{
																							Description: "SHA384 Digest",
																							Required:    false,
																							Optional:    false,
																							Computed:    true,
																							Attributes: map[string]schema.Attribute{
																								"digest": schema.StringAttribute{
																									Description: "Digest The 'digest' is the DS key and the actual contents of the DS record.",
																									Required:    false,
																									Optional:    false,
																									Computed:    true,
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
																Optional:    false,
																Computed:    true,
																Attributes: map[string]schema.Attribute{
																	"name": schema.StringAttribute{
																		Description: "Record Name MX Record name, please provide only the specific subdomain or record name without the base domain.",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																	},
																	"values": schema.ListNestedAttribute{
																		Description: "Value",
																		Required:    false,
																		Optional:    false,
																		Computed:    true,
																		NestedObject: schema.NestedAttributeObject{
																			Attributes: map[string]schema.Attribute{
																				"domain": schema.StringAttribute{
																					Description: "Domain Mail exchanger domain name, please provide the full hostname, for example: mail.example.com",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
																				},
																				"priority": schema.Int32Attribute{
																					Description: "Priority Mail exchanger priority code",
																					Required:    false,
																					Optional:    false,
																					Computed:    true,
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
														Optional:    false,
														Computed:    true,
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
				Optional:    false,
				Computed:    true,
			},
		},
	}
}

func (d *DataSourceF5XCDnsZone) Configure(ctx context.Context, request datasource.ConfigureRequest, response *datasource.ConfigureResponse) {
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

	d.client = client
}

func (d *DataSourceF5XCDnsZone) Read(ctx context.Context, request datasource.ReadRequest, response *datasource.ReadResponse) {
	var data DataSourceF5XCDnsZoneModel

	// Read Terraform configuration data into the model
	response.Diagnostics.Append(request.Config.Get(ctx, &data)...)

	if response.Diagnostics.HasError() {
		return
	}

	httpResp, err := d.client.GetObject(ctx, ves_io_schema_dns_zone.ObjectType, data.Namespace.ValueString(), data.Name.ValueString())
	if err != nil {
		response.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read data, got error: %s", err))
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
			for i := range primaryOptions.Primary.GetRrSetGroup() {
				//TODO: initialize ArrayOfPointerField e.g. data.DnsTypeChoice.Primary.RrSetGroup = []DnsZoneRRSetGroup{}

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
					for ii := range primaryOptions.Primary.GetRrSetGroup()[i].GetRrSet() {
						//TODO: initialize ArrayOfPointerField e.g. data.DnsTypeChoice.Primary.RrSetGroup = []DnsZoneRRSetGroup{}

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
								// TODO: implementation is missing for Values in dns_zone parent item is locRecord --> PointerOfOneOfField

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
								// TODO: implementation is missing for Values in dns_zone parent item is locRecord --> PointerOfOneOfField

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
								// TODO: implementation is missing for Values in dns_zone parent item is locRecord --> PointerOfOneOfField

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
								// TODO: implementation is missing for Values in dns_zone parent item is locRecord --> PointerOfOneOfField

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
								for iii := range caaRecordOptions.CaaRecord.GetValues() {
									//TODO: initialize ArrayOfPointerField e.g. data.DnsTypeChoice.Primary.RrSetGroup = []DnsZoneRRSetGroup{}

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
								// TODO: implementation is missing for Values in dns_zone parent item is locRecord --> PointerOfOneOfField

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
								// TODO: implementation is missing for Values in dns_zone parent item is locRecord --> PointerOfOneOfField

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
								// TODO: implementation is missing for Values in dns_zone parent item is locRecord --> PointerOfOneOfField

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
								// TODO: implementation is missing for Values in dns_zone parent item is locRecord --> PointerOfOneOfField

								// resource specToModelNested IntField --> LongitudeMinute
								data.DnsTypeChoice.Primary.RrSetGroup[i].RrSet[ii].TypeRecordSetChoice.LocRecord.Values[iii].LongitudeMinute = types.Int32Value(locRecordOptions.LocRecord.GetValues()[iii].GetLongitudeMinute())
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
								// TODO: implementation is missing for Values in dns_zone parent item is locRecord --> PointerOfOneOfField

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
								// TODO: implementation is missing for Values in dns_zone parent item is locRecord --> PointerOfOneOfField

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
