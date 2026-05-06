##############################################################################
# ENV
##############################################################################
# These resources have been tested against playground staging env
# If another env is used one has to make sure certain values / refs are not
# available.

##############################################################################
# Provider settings
##############################################################################
#f5xc_api_url      = "https://your-tenant.console.ves.volterra.io"
#f5xc_api_p12_file = "/path/to/your/api-credential.p12"
#f5xc_api_token    = ""

##############################################################################
# Master toggle – set to true to enable ALL resources at once.
# Individual resource flags below are ignored when this is true.
##############################################################################
f5xc_all_resources = false

##############################################################################
# Resource creation control flags
# Set to true to create the resource, false to skip it.
##############################################################################
f5xc_route                    = true
f5xc_token                    = true
f5xc_namespace                = true
f5xc_nat_policy               = false
f5xc_app_setting              = false
f5xc_healthcheck              = false
f5xc_virtual_k8s              = false
f5xc_origin_pool              = false
f5xc_alert_policy             = false
f5xc_virtual_site             = false
f5xc_app_firewall             = false
f5xc_service_policy           = false
f5xc_v1_dns_monitor           = false
f5xc_voltstack_site           = false
f5xc_alert_receiver           = false
f5xc_virtual_network          = false
f5xc_site_mesh_group          = false
f5xc_network_firewall         = true
f5xc_http_loadbalancer        = true
f5xc_network_policy_rule      = false
f5xc_forward_proxy_policy     = false
f5xc_api_definition_uploader  = false
f5xc_enhanced_firewall_policy = false

##############################################################################
# Resource output control flags
# Set to true to include the resource in the "resource" output block.
##############################################################################
f5xc_route_output                    = false
f5xc_token_output                    = false
f5xc_namespace_output                = false
f5xc_nat_policy_output               = false
f5xc_app_setting_output              = false
f5xc_healthcheck_output              = false
f5xc_virtual_k8s_output              = false
f5xc_origin_pool_output              = false
f5xc_alert_policy_output             = false
f5xc_virtual_site_output             = false
f5xc_app_firewall_output             = false
f5xc_service_policy_output           = false
f5xc_v1_dns_monitor_output           = false
f5xc_voltstack_site_output           = false
f5xc_alert_receiver_output           = false
f5xc_virtual_network_output          = false
f5xc_site_mesh_group_output          = false
f5xc_network_firewall_output         = false
f5xc_http_loadbalancer_output        = false
f5xc_network_policy_rule_output      = false
f5xc_forward_proxy_policy_output     = false
f5xc_api_definition_uploader_output  = false
f5xc_enhanced_firewall_policy_output = false

##############################################################################
# Data source control flags
# Set to true to query the data source for the corresponding resource.
# Note: route, nat_policy, app_setting, virtual_site and app_firewall have
#       no data source and are therefore not listed here.
##############################################################################
f5xc_token_data                    = true
f5xc_namespace_data                = false
f5xc_healthcheck_data              = false
f5xc_virtual_k8s_data              = false
f5xc_origin_pool_data              = false
f5xc_alert_policy_data             = false
f5xc_service_policy_data           = false
f5xc_v1_dns_monitor_data           = false
f5xc_voltstack_site_data           = false
f5xc_alert_receiver_data           = false
f5xc_virtual_network_data          = false
f5xc_site_mesh_group_data          = false
f5xc_network_firewall_data         = false
f5xc_http_loadbalancer_data        = false
f5xc_tcp_loadbalancer_data         = false
f5xc_network_policy_rule_data      = false
f5xc_forward_proxy_policy_data     = false
f5xc_enhanced_firewall_policy_data = false
f5xc_securemesh_site_v2_data       = false

##############################################################################
# Data source output control flags
# Set to true to include the data source in the "data" output block.
# Note: route, nat_policy, app_setting, virtual_site and app_firewall have
#       no data source and are therefore not listed here.
##############################################################################
f5xc_token_data_output                    = true
f5xc_namespace_data_output                = false
f5xc_healthcheck_data_output              = false
f5xc_virtual_k8s_data_output              = false
f5xc_origin_pool_data_output              = false
f5xc_alert_policy_data_output             = false
f5xc_service_policy_data_output           = false
f5xc_v1_dns_monitor_data_output           = false
f5xc_voltstack_site_data_output           = false
f5xc_alert_receiver_data_output           = false
f5xc_virtual_network_data_output          = false
f5xc_site_mesh_group_data_output          = false
f5xc_network_firewall_data_output         = false
f5xc_http_loadbalancer_data_output        = false
f5xc_tcp_loadbalancer_data_output         = false
f5xc_network_policy_rule_data_output      = false
f5xc_forward_proxy_policy_data_output     = false
f5xc_enhanced_firewall_policy_data_output = false
f5xc_securemesh_site_v2_data_output       = false
