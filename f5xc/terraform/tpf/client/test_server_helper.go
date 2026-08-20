// Package client
// Copyright (c) 2025 F5 Inc. All rights reserved.

package client

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/types"
	"gopkg.volterra.us/stdlib/client/vesapi"
	"gopkg.volterra.us/stdlib/db"
	"gopkg.volterra.us/stdlib/server"
	"gopkg.volterra.us/stdlib/svcfw"
	"gopkg.volterra.us/stdlib/testutil/fixtures/vesenv"

	ves_io_schema_api_credential "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/api_credential"
	ves_io_schema_dns_zone_rrset "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/dns_zone/rrset"
	ves_io_schema_known_label "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/known_label"
	ves_io_schema_known_label_key "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/known_label_key"
	ves_io_schema_ns "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/namespace"
	ves_io_schema_site "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/site"
	"gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views"
	ves_io_schema_aws_tgw_site "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views/aws_tgw_site"
	ves_io_schema_aws_vpc_site "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views/aws_vpc_site"
	ves_io_schema_azure_vnet_site "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views/azure_vnet_site"
	ves_io_schema_gcp_vpc_site "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views/gcp_vpc_site"
	ves_io_schema_tf_params "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/views/terraform_parameters"
	ves_io_schema_vh "gopkg.volterra.us/terraform-provider-f5xc/pbgo/extschema/schema/virtual_host"
)

var (
	tdRoot = "../testdata"
)

// APIStimulus is a type holding all RPC stimulus keyed by RPC Name
type APIStimulus map[string]*RPCStimulus

// RPCStimulus holds the APIOperator s to exercise the RPC and associated documentation
type RPCStimulus struct {
	APIOps []vesenv.APIOperator
}

// apiCredentialCustomAPIServer implements ves.io.schema.api_credential.CustomAPIServer interface
type apiCredentialCustomAPIServer struct {
	//sf svcfw.Service
}

func (s *apiCredentialCustomAPIServer) CreateKubeConfig(_ context.Context, _ *ves_io_schema_api_credential.CreateKubeConfigRequest) (*ves_io_schema_api_credential.CreateResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) CreateKubeConfigServiceCredentials(_ context.Context, _ *ves_io_schema_api_credential.CreateKubeConfigServiceCredentialsRequest) (*ves_io_schema_api_credential.CreateResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) CreateSiteGlobalKubeConfig(_ context.Context, _ *ves_io_schema_api_credential.CreateSiteGlobalKubeConfigRequest) (*ves_io_schema_api_credential.CreateResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) GetKubeConfig(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.GetResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) GetKubeConfigServiceCredentials(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.GetServiceCredentialsResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) ListKubeConfig(_ context.Context, _ *ves_io_schema_api_credential.ListRequest) (*ves_io_schema_api_credential.ListResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) ListKubeConfigServiceCredentials(_ context.Context, _ *ves_io_schema_api_credential.ListRequest) (*ves_io_schema_api_credential.ListServiceCredentialsResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) RevokeKubeConfig(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.StatusResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) RevokeKubeConfigServiceCredentials(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.StatusResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) ActivateKubeConfig(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.StatusResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) ActivateKubeConfigServiceCredentials(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.StatusResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) GetSiteGlobalKubeConfig(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.GetServiceCredentialsResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) ListSiteGlobalKubeConfig(_ context.Context, _ *ves_io_schema_api_credential.ListRequest) (*ves_io_schema_api_credential.ListServiceCredentialsResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) RevokeSiteGlobalKubeConfig(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.StatusResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) ActivateSiteGlobalKubeConfig(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.StatusResponse, error) {
	panic("implement me")
}

func (s *apiCredentialCustomAPIServer) Create(_ context.Context, _ *ves_io_schema_api_credential.CreateRequest) (*ves_io_schema_api_credential.CreateResponse, error) {
	return &rspCreateResponse, nil
}

func (s *apiCredentialCustomAPIServer) Get(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.GetResponse, error) {
	timestamp := &types.Timestamp{
		Seconds: time.Now().Unix(),
		Nanos:   int32(time.Now().Nanosecond()),
	}
	return &ves_io_schema_api_credential.GetResponse{
		Object: &ves_io_schema_api_credential.Object{
			Spec: &ves_io_schema_api_credential.SpecType{
				GcSpec: &ves_io_schema_api_credential.GlobalSpecType{
					ExpirationTimestamp: timestamp,
				},
			},
		},
	}, nil
}

func (s *apiCredentialCustomAPIServer) List(_ context.Context, _ *ves_io_schema_api_credential.ListRequest) (*ves_io_schema_api_credential.ListResponse, error) {
	return &ves_io_schema_api_credential.ListResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) Revoke(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.StatusResponse, error) {
	return &ves_io_schema_api_credential.StatusResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) Activate(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.StatusResponse, error) {
	return &ves_io_schema_api_credential.StatusResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) Renew(_ context.Context, _ *ves_io_schema_api_credential.RenewRequest) (*ves_io_schema_api_credential.StatusResponse, error) {
	return &ves_io_schema_api_credential.StatusResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) CreateServiceCredentials(_ context.Context, _ *ves_io_schema_api_credential.CreateServiceCredentialsRequest) (*ves_io_schema_api_credential.CreateResponse, error) {
	return &ves_io_schema_api_credential.CreateResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) ListServiceCredentials(_ context.Context, _ *ves_io_schema_api_credential.ListRequest) (*ves_io_schema_api_credential.ListServiceCredentialsResponse, error) {
	return &ves_io_schema_api_credential.ListServiceCredentialsResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) ActivateServiceCredentials(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.StatusResponse, error) {
	return &ves_io_schema_api_credential.StatusResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) RenewServiceCredentials(_ context.Context, _ *ves_io_schema_api_credential.RenewRequest) (*ves_io_schema_api_credential.StatusResponse, error) {
	return &ves_io_schema_api_credential.StatusResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) RevokeServiceCredentials(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.StatusResponse, error) {
	return &ves_io_schema_api_credential.StatusResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) RecreateScimToken(_ context.Context, _ *ves_io_schema_api_credential.RecreateScimTokenRequest) (*ves_io_schema_api_credential.CreateResponse, error) {
	return &ves_io_schema_api_credential.CreateResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) RevokeScimToken(_ context.Context, _ *ves_io_schema_api_credential.ScimTokenRequest) (*ves_io_schema_api_credential.StatusResponse, error) {
	return &ves_io_schema_api_credential.StatusResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) GetScimToken(_ context.Context, _ *ves_io_schema_api_credential.ScimTokenRequest) (*ves_io_schema_api_credential.GetResponse, error) {
	return &ves_io_schema_api_credential.GetResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) GetServiceCredentials(_ context.Context, _ *ves_io_schema_api_credential.GetRequest) (*ves_io_schema_api_credential.GetServiceCredentialsResponse, error) {
	return &ves_io_schema_api_credential.GetServiceCredentialsResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) ReplaceServiceCredentials(context.Context, *ves_io_schema_api_credential.ReplaceServiceCredentialsRequest) (*ves_io_schema_api_credential.ReplaceServiceCredentialsResponse, error) {
	return &ves_io_schema_api_credential.ReplaceServiceCredentialsResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) BulkRevoke(_ context.Context, _ *ves_io_schema_api_credential.BulkRevokeRequest) (*ves_io_schema_api_credential.BulkRevokeResponse, error) {
	return &ves_io_schema_api_credential.BulkRevokeResponse{}, nil
}

func (s *apiCredentialCustomAPIServer) BulkRevokeServiceCredentials(_ context.Context, _ *ves_io_schema_api_credential.BulkRevokeRequest) (*ves_io_schema_api_credential.BulkRevokeResponse, error) {
	return &ves_io_schema_api_credential.BulkRevokeResponse{}, nil
}

var _ ves_io_schema_api_credential.CustomAPIServer = &apiCredentialCustomAPIServer{}

var (
	rspCreateResponse = ves_io_schema_api_credential.CreateResponse{}
)

// ves.io.schema.virtual_host.CustomAPI handling - start

type vhCustomAPIServer struct {
	//sf svcfw.Service
}

// GetDnsInfo custom api implementation
func (s *vhCustomAPIServer) GetDnsInfo(_ context.Context,
	_ *ves_io_schema_vh.GetDnsInfoRequest) (*ves_io_schema_vh.GetDnsInfoResponse, error) {
	return &ves_io_schema_vh.GetDnsInfoResponse{}, nil
}

func (s *vhCustomAPIServer) AssignAPIDefinition(context.Context, *ves_io_schema_vh.AssignAPIDefinitionReq) (*ves_io_schema_vh.AssignAPIDefinitionResp, error) {
	return &ves_io_schema_vh.AssignAPIDefinitionResp{}, nil
}

var _ ves_io_schema_vh.CustomAPIServer = &vhCustomAPIServer{}

// ves.io.schema.virtual_host.CustomAPI handling - end

// ves.io.schema.site.CustomAPI handling - start
type siteCustomAPIServer struct {
	//sf svcfw.Service
}

func (s *siteCustomAPIServer) SetState(_ context.Context,
	_ *ves_io_schema_site.SetStateReq) (*ves_io_schema_site.SetStateResp, error) {
	return &ves_io_schema_site.SetStateResp{}, nil
}

var _ ves_io_schema_site.CustomStateAPIServer = &siteCustomAPIServer{}

// ves.io.schema.site.CustomAPI handling - end

// ves.io.schema.namespace.CustomAPI handling - start
type nsCustomAPIServer struct {
	//sf svcfw.Service
}

func (n *nsCustomAPIServer) CascadeDelete(context.Context,
	*ves_io_schema_ns.CascadeDeleteRequest) (*ves_io_schema_ns.CascadeDeleteResponse, error) {
	return &ves_io_schema_ns.CascadeDeleteResponse{}, nil
}

func (n *nsCustomAPIServer) EvaluateAPIAccess(context.Context,
	*ves_io_schema_ns.EvaluateAPIAccessReq) (*ves_io_schema_ns.EvaluateAPIAccessResp, error) {
	return &ves_io_schema_ns.EvaluateAPIAccessResp{}, nil
}

func (n *nsCustomAPIServer) EvaluateBatchAPIAccess(context.Context, *ves_io_schema_ns.EvaluateBatchAPIAccessReq) (*ves_io_schema_ns.EvaluateBatchAPIAccessResp, error) {
	return &ves_io_schema_ns.EvaluateBatchAPIAccessResp{}, nil
}

func (n *nsCustomAPIServer) SuggestValues(context.Context,
	*ves_io_schema_ns.SuggestValuesReq) (*ves_io_schema_ns.SuggestValuesResp, error) {
	return &ves_io_schema_ns.SuggestValuesResp{}, nil
}

func (n *nsCustomAPIServer) LookupUserRoles(context.Context,
	*ves_io_schema_ns.LookupUserRolesReq) (*ves_io_schema_ns.LookupUserRolesResp, error) {
	return &ves_io_schema_ns.LookupUserRolesResp{}, nil
}

var _ ves_io_schema_ns.CustomAPIServer = &nsCustomAPIServer{}

// ves.io.schema.namespace.CustomAPI handling - end

// ves.io.schema.views.aws_vpc_site.CustomAPI handling - start
type vpcCustomAPIServer struct {
	//sf svcfw.Service
}

func (t *vpcCustomAPIServer) SetVPCK8SHostnames(context.Context,
	*ves_io_schema_aws_vpc_site.SetVPCK8SHostnamesRequest) (*ves_io_schema_aws_vpc_site.SetVPCK8SHostnamesResponse, error) {
	return &ves_io_schema_aws_vpc_site.SetVPCK8SHostnamesResponse{}, nil

}

func (t *vpcCustomAPIServer) SetVIPInfo(context.Context,
	*ves_io_schema_aws_vpc_site.SetVIPInfoRequest) (*ves_io_schema_aws_vpc_site.SetVIPInfoResponse, error) {
	return &ves_io_schema_aws_vpc_site.SetVIPInfoResponse{}, nil

}

func (t *vpcCustomAPIServer) SetCloudSiteInfo(context.Context,
	*ves_io_schema_aws_vpc_site.SetCloudSiteInfoRequest) (*ves_io_schema_aws_vpc_site.SetCloudSiteInfoResponse, error) {
	return &ves_io_schema_aws_vpc_site.SetCloudSiteInfoResponse{}, nil

}

func (t *vpcCustomAPIServer) ValidateConfig(context.Context, *views.ValidateConfigRequest) (*views.ValidateConfigResponse, error) {
	return &views.ValidateConfigResponse{}, nil
}

var _ ves_io_schema_aws_vpc_site.CustomAPIServer = &vpcCustomAPIServer{}

// ves.io.schema.views.azure_vnet_site.CustomAPI handling - start
type vnetCustomAPIServer struct {
	//sf svcfw.Service
}

func (t *vnetCustomAPIServer) SetVIPInfo(context.Context,
	*ves_io_schema_azure_vnet_site.SetVIPInfoRequest) (*ves_io_schema_azure_vnet_site.SetVIPInfoResponse, error) {
	return &ves_io_schema_azure_vnet_site.SetVIPInfoResponse{}, nil

}

func (t *vnetCustomAPIServer) SetCloudSiteInfo(context.Context,
	*ves_io_schema_azure_vnet_site.SetCloudSiteInfoRequest) (*ves_io_schema_azure_vnet_site.SetCloudSiteInfoResponse, error) {
	return &ves_io_schema_azure_vnet_site.SetCloudSiteInfoResponse{}, nil

}

func (t *vnetCustomAPIServer) ValidateConfig(context.Context, *views.ValidateConfigRequest) (*views.ValidateConfigResponse, error) {
	return &views.ValidateConfigResponse{}, nil
}

var _ ves_io_schema_azure_vnet_site.CustomAPIServer = &vnetCustomAPIServer{}

// ves.io.schema.views.azure_vnet_site.CustomAPI handling - end

// ves.io.schema.views.aws_tgw_site.CustomAPI handling - start
type tgwCustomAPIServer struct {
	sf svcfw.Service
}

func (t *tgwCustomAPIServer) SetVPCIpPrefixes(ctx context.Context,
	r *ves_io_schema_aws_tgw_site.SetVPCIpPrefixesRequest) (*ves_io_schema_aws_tgw_site.SetVPCIpPrefixesResponse, error) {
	tenant := server.TenantFromContext(ctx)
	if tenant == "" {
		return nil, fmt.Errorf("tenant missing in request")
	}

	e, ok, err := t.sf.FindEntry(ctx, ves_io_schema_aws_tgw_site.ObjectDefTblName, "", db.WithFindUsingNSIndex(tenant, "system", r.Name))
	if err != nil {
		return nil, fmt.Errorf("cannot find aws_tgw_site entry %s, err: %s", r.Name, err)
	}
	if !ok {
		return nil, fmt.Errorf("aws_tgw_site %s does not exist", r.Name)
	}
	obj := e.(*ves_io_schema_aws_tgw_site.DBObject)
	obj.Spec.GcSpec.VpcIpPrefixes = r.GetVpcIpPrefixes()
	if _, err := t.sf.ChgEntry(ctx, obj.ToEntry()); err != nil {
		return nil, fmt.Errorf("could not update aws_tgw_site %s", err)
	}
	return &ves_io_schema_aws_tgw_site.SetVPCIpPrefixesResponse{}, nil

}
func (t *tgwCustomAPIServer) SetVPNTunnels(context.Context,
	*ves_io_schema_aws_tgw_site.SetVPNTunnelsRequest) (*ves_io_schema_aws_tgw_site.SetVPNTunnelsResponse, error) {
	return &ves_io_schema_aws_tgw_site.SetVPNTunnelsResponse{}, nil
}

func (t *tgwCustomAPIServer) SetTGWInfo(context.Context,
	*ves_io_schema_aws_tgw_site.SetTGWInfoRequest) (*ves_io_schema_aws_tgw_site.SetTGWInfoResponse, error) {
	return &ves_io_schema_aws_tgw_site.SetTGWInfoResponse{}, nil
}

func (t *tgwCustomAPIServer) SetVIPInfo(context.Context,
	*ves_io_schema_aws_tgw_site.SetVIPInfoRequest) (*ves_io_schema_aws_tgw_site.SetVIPInfoResponse, error) {
	return &ves_io_schema_aws_tgw_site.SetVIPInfoResponse{}, nil
}

func (t *tgwCustomAPIServer) ValidateConfig(context.Context, *views.ValidateConfigRequest) (*views.ValidateConfigResponse, error) {
	return &views.ValidateConfigResponse{}, nil
}

var _ ves_io_schema_aws_tgw_site.CustomAPIServer = &tgwCustomAPIServer{}

// ves.io.schema.views.aws_tgw_site.CustomAPI handling - end

// ves.io.schema.views.terraform_parameters.CustomAPI handling - start
type tfCustomAPIServer struct {
	//sf svcfw.Service
}

func (t *tfCustomAPIServer) Get(_ context.Context, _ *ves_io_schema_tf_params.GetRequest) (*ves_io_schema_tf_params.GetResponse, error) {
	return &ves_io_schema_tf_params.GetResponse{}, nil

}
func (t *tfCustomAPIServer) GetStatus(_ context.Context, _ *ves_io_schema_tf_params.GetRequest) (*ves_io_schema_tf_params.GetStatusResponse, error) {
	return &ves_io_schema_tf_params.GetStatusResponse{}, nil
}

var _ ves_io_schema_tf_params.CustomAPIServer = &tfCustomAPIServer{}

// ves.io.schema.views.terraform_parameters.CustomAPI handling - end

// ves.io.schema.views.terraform_parameters.CustomActionAPI handling - start
type tfCustomActionAPIServer struct {
	//sf svcfw.Service
}

func (t *tfCustomActionAPIServer) Run(_ context.Context, _ *ves_io_schema_tf_params.RunRequest) (*ves_io_schema_tf_params.RunResponse, error) {
	return &ves_io_schema_tf_params.RunResponse{}, nil

}
func (t *tfCustomActionAPIServer) ForceDelete(_ context.Context, _ *ves_io_schema_tf_params.ForceDeleteRequest) (*ves_io_schema_tf_params.ForceDeleteResponse, error) {
	return &ves_io_schema_tf_params.ForceDeleteResponse{}, nil
}

var _ ves_io_schema_tf_params.CustomActionAPIServer = &tfCustomActionAPIServer{}

// ves.io.schema.views.terraform_parameters.CustomAPI handling - end

// ves.io.schema.namespace.NamespaceCustomAPI handling - start
type namespaceCustomAPIServer struct {
	//sf svcfw.Service
}

func (n *namespaceCustomAPIServer) DynamicData(_ context.Context, _ *ves_io_schema_ns.DynamicDataReq) (*ves_io_schema_ns.DynamicDataResp, error) {
	panic("implement me")
}

func (n *namespaceCustomAPIServer) SetFastACLsForInternetVIPs(_ context.Context, _ *ves_io_schema_ns.SetFastACLsForInternetVIPsRequest) (*ves_io_schema_ns.SetFastACLsForInternetVIPsResponse, error) {
	return &ves_io_schema_ns.SetFastACLsForInternetVIPsResponse{}, nil

}
func (n *namespaceCustomAPIServer) GetFastACLsForInternetVIPs(_ context.Context, _ *ves_io_schema_ns.GetFastACLsForInternetVIPsRequest) (*ves_io_schema_ns.GetFastACLsForInternetVIPsResponse, error) {
	return &ves_io_schema_ns.GetFastACLsForInternetVIPsResponse{}, nil
}

func (n *namespaceCustomAPIServer) SetActiveServicePolicies(_ context.Context, _ *ves_io_schema_ns.SetActiveServicePoliciesRequest) (*ves_io_schema_ns.SetActiveServicePoliciesResponse, error) {
	return &ves_io_schema_ns.SetActiveServicePoliciesResponse{}, nil

}
func (n *namespaceCustomAPIServer) GetActiveServicePolicies(_ context.Context, _ *ves_io_schema_ns.GetActiveServicePoliciesRequest) (*ves_io_schema_ns.GetActiveServicePoliciesResponse, error) {
	return &ves_io_schema_ns.GetActiveServicePoliciesResponse{}, nil
}

func (n *namespaceCustomAPIServer) SetActiveNetworkPolicies(_ context.Context, _ *ves_io_schema_ns.SetActiveNetworkPoliciesRequest) (*ves_io_schema_ns.SetActiveNetworkPoliciesResponse, error) {
	return &ves_io_schema_ns.SetActiveNetworkPoliciesResponse{}, nil

}
func (n *namespaceCustomAPIServer) GetActiveNetworkPolicies(_ context.Context, _ *ves_io_schema_ns.GetActiveNetworkPoliciesRequest) (*ves_io_schema_ns.GetActiveNetworkPoliciesResponse, error) {
	return &ves_io_schema_ns.GetActiveNetworkPoliciesResponse{}, nil
}

func (n *namespaceCustomAPIServer) SetActiveAlertPolicies(_ context.Context, _ *ves_io_schema_ns.SetActiveAlertPoliciesRequest) (*ves_io_schema_ns.SetActiveAlertPoliciesResponse, error) {
	return &ves_io_schema_ns.SetActiveAlertPoliciesResponse{}, nil

}
func (n *namespaceCustomAPIServer) GetActiveAlertPolicies(_ context.Context, _ *ves_io_schema_ns.GetActiveAlertPoliciesRequest) (*ves_io_schema_ns.GetActiveAlertPoliciesResponse, error) {
	return &ves_io_schema_ns.GetActiveAlertPoliciesResponse{}, nil
}

func (n *namespaceCustomAPIServer) CascadeDelete(_ context.Context, _ *ves_io_schema_ns.CascadeDeleteRequest) (*ves_io_schema_ns.CascadeDeleteResponse, error) {
	return &ves_io_schema_ns.CascadeDeleteResponse{}, nil
}

func (n *namespaceCustomAPIServer) SuggestValues(_ context.Context, _ *ves_io_schema_ns.SuggestValuesReq) (*ves_io_schema_ns.SuggestValuesResp, error) {
	return &ves_io_schema_ns.SuggestValuesResp{}, nil
}

func (n *namespaceCustomAPIServer) ValidateRules(_ context.Context, _ *ves_io_schema_ns.ValidateRulesReq) (*ves_io_schema_ns.ValidateRulesResponse, error) {
	return &ves_io_schema_ns.ValidateRulesResponse{}, nil
}

func (n *namespaceCustomAPIServer) UpdateAllowAdvertiseOnPublic(context.Context,
	*ves_io_schema_ns.UpdateAllowAdvertiseOnPublicReq) (*ves_io_schema_ns.UpdateAllowAdvertiseOnPublicResp, error) {
	return &ves_io_schema_ns.UpdateAllowAdvertiseOnPublicResp{}, nil
}

func (n *namespaceCustomAPIServer) NetworkingInventory(context.Context,
	*ves_io_schema_ns.NetworkingInventoryRequest) (*ves_io_schema_ns.NetworkingInventoryResponse, error) {
	return &ves_io_schema_ns.NetworkingInventoryResponse{}, nil
}

func (n *namespaceCustomAPIServer) ApplicationInventory(context.Context,
	*ves_io_schema_ns.ApplicationInventoryRequest) (*ves_io_schema_ns.ApplicationInventoryResponse, error) {
	return &ves_io_schema_ns.ApplicationInventoryResponse{}, nil
}

func (n *namespaceCustomAPIServer) AllApplicationInventory(context.Context,
	*ves_io_schema_ns.AllApplicationInventoryRequest) (*ves_io_schema_ns.ApplicationInventoryResponse, error) {
	return &ves_io_schema_ns.ApplicationInventoryResponse{}, nil
}

func (n *namespaceCustomAPIServer) AllApplicationInventoryWaf(context.Context,
	*ves_io_schema_ns.AllApplicationInventoryWafFilterRequest) (*ves_io_schema_ns.AllApplicationInventoryWafFilterResponse, error) {
	return &ves_io_schema_ns.AllApplicationInventoryWafFilterResponse{}, nil
}

func (n *namespaceCustomAPIServer) OIDCOAuthDiscovery(context.Context,
	*ves_io_schema_ns.OIDCOAuthDiscoveryReq) (*ves_io_schema_ns.OIDCOAuthDiscoveryResp, error) {
	return &ves_io_schema_ns.OIDCOAuthDiscoveryResp{}, nil
}

var _ ves_io_schema_ns.NamespaceCustomAPIServer = &namespaceCustomAPIServer{}

// ves.io.schema.views.gcp_vpc_site.CustomAPI handling - start
type gcpVpcCustomAPIServer struct {
	//sf svcfw.Service
}

func (t *gcpVpcCustomAPIServer) SetCloudSiteInfo(context.Context,
	*ves_io_schema_gcp_vpc_site.SetCloudSiteInfoRequest) (*ves_io_schema_gcp_vpc_site.SetCloudSiteInfoResponse, error) {
	return &ves_io_schema_gcp_vpc_site.SetCloudSiteInfoResponse{}, nil

}

func (t *gcpVpcCustomAPIServer) ValidateConfig(context.Context, *views.ValidateConfigRequest) (*views.ValidateConfigResponse, error) {
	return &views.ValidateConfigResponse{}, nil
}

var _ ves_io_schema_gcp_vpc_site.CustomAPIServer = &gcpVpcCustomAPIServer{}

// ves.io.schema.namespace.NamespaceCustomAPI handling - end

func getTestClientOpts(url, tenant string) ([]vesapi.ConfigOpt, error) {

	clKey := fmt.Sprintf("file:///%s/tls/client.examplesvc.ves.io.key", tdRoot)
	clCert := fmt.Sprintf("file:///%s/tls/client.examplesvc.ves.io.crt", tdRoot)
	clCACert := fmt.Sprintf("file:///%s/tls/client.ca.crt", tdRoot)

	policy, err := resolveThrottlingPolicy(nil)
	if err != nil {
		return nil, err
	}
	remapperFn := uriRemapperFn(tenant, policy, newResourceThrottleRegistry())

	clOpts := []vesapi.ConfigOpt{
		vesapi.WithCfgKey(clKey),
		vesapi.WithCfgCert(clCert),
		vesapi.WithCfgCACert(clCACert),
		vesapi.WithCfgRoundTripperFn(remapperFn),
	}
	vesapi.ServerCN = "examplesvc.ves.io"

	return clOpts, nil
}

// ves.io.schema.known_label.CustomAPI handling - start
type dnsZoneRecordCustomAPIServer struct {
	//sf svcfw.Service
}

func (s *dnsZoneRecordCustomAPIServer) Create(_ context.Context, _ *ves_io_schema_dns_zone_rrset.CreateRequest) (*ves_io_schema_dns_zone_rrset.Response, error) {
	return &ves_io_schema_dns_zone_rrset.Response{}, nil
}

func (s *dnsZoneRecordCustomAPIServer) Get(_ context.Context, _ *ves_io_schema_dns_zone_rrset.GetRequest) (*ves_io_schema_dns_zone_rrset.Response, error) {
	return &ves_io_schema_dns_zone_rrset.Response{}, nil
}

func (s *dnsZoneRecordCustomAPIServer) Replace(_ context.Context, _ *ves_io_schema_dns_zone_rrset.ReplaceRequest) (*ves_io_schema_dns_zone_rrset.Response, error) {
	return &ves_io_schema_dns_zone_rrset.Response{}, nil
}

func (s *dnsZoneRecordCustomAPIServer) Delete(_ context.Context, _ *ves_io_schema_dns_zone_rrset.DeleteRequest) (*ves_io_schema_dns_zone_rrset.Response, error) {
	return &ves_io_schema_dns_zone_rrset.Response{}, nil
}

var _ ves_io_schema_dns_zone_rrset.CustomAPIServer = &dnsZoneRecordCustomAPIServer{}

// ves.io.schema.known_label.CustomAPI handling - start
type knownLabelCustomAPIServer struct {
	//sf svcfw.Service
}

func (s *knownLabelCustomAPIServer) Create(_ context.Context, _ *ves_io_schema_known_label.CreateRequest) (*ves_io_schema_known_label.CreateResponse, error) {
	return &ves_io_schema_known_label.CreateResponse{}, nil
}

func (s *knownLabelCustomAPIServer) Get(_ context.Context, req *ves_io_schema_known_label.GetRequest) (*ves_io_schema_known_label.GetResponse, error) {
	return &ves_io_schema_known_label.GetResponse{
		Label: []*ves_io_schema_known_label.LabelType{
			{
				Key:   req.Key,
				Value: req.Value,
			},
		},
	}, nil
}

func (s *knownLabelCustomAPIServer) Delete(_ context.Context, _ *ves_io_schema_known_label.DeleteRequest) (*ves_io_schema_known_label.DeleteResponse, error) {
	return &ves_io_schema_known_label.DeleteResponse{}, nil
}

var _ ves_io_schema_known_label.CustomAPIServer = &knownLabelCustomAPIServer{}

// ves.io.schema.known_label.CustomAPI handling - start
type knownLabelKeyCustomAPIServer struct {
	//sf svcfw.Service
}

func (s *knownLabelKeyCustomAPIServer) Create(_ context.Context, _ *ves_io_schema_known_label_key.CreateRequest) (*ves_io_schema_known_label_key.CreateResponse, error) {
	return &ves_io_schema_known_label_key.CreateResponse{}, nil
}

func (s *knownLabelKeyCustomAPIServer) Get(_ context.Context, req *ves_io_schema_known_label_key.GetRequest) (*ves_io_schema_known_label_key.GetResponse, error) {
	return &ves_io_schema_known_label_key.GetResponse{
		LabelKey: []*ves_io_schema_known_label_key.LabelKeyType{
			{
				Key: req.Key,
			},
		},
	}, nil
}

func (s *knownLabelKeyCustomAPIServer) Delete(_ context.Context, _ *ves_io_schema_known_label_key.DeleteRequest) (*ves_io_schema_known_label_key.DeleteResponse, error) {
	return &ves_io_schema_known_label_key.DeleteResponse{}, nil
}

var _ ves_io_schema_known_label_key.CustomAPIServer = &knownLabelKeyCustomAPIServer{}
