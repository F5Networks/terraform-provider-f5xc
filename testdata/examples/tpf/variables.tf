##############################################################################
# Provider settings
##############################################################################
variable "f5xc_api_url" {
  description = "F5XC API URL"
  type        = string
}

variable "f5xc_api_p12_file" {
  description = "Path to the F5XC API P12 credential file"
  type        = string
}

variable "f5xc_api_token" {
  description = "F5XC API token"
  type        = string
}

##############################################################################
# Master toggle – set to true to enable ALL resources at once.
##############################################################################
variable "f5xc_all_resources" {
  description = "Enable ALL resources at once (overrides individual resource flags)"
  type        = bool
  default     = false
}

##############################################################################
# Resource creation control flags
# Set to true to create the resource, false to skip it.
##############################################################################
variable "f5xc_route" {
  description = "Create f5xc_route resource"
  type        = bool
  default     = false
}

variable "f5xc_token" {
  description = "Create f5xc_token resource"
  type        = bool
  default     = false
}

variable "f5xc_namespace" {
  description = "Create f5xc_namespace resource"
  type        = bool
  default     = false
}

variable "f5xc_nat_policy" {
  description = "Create f5xc_nat_policy resource"
  type        = bool
  default     = false
}

variable "f5xc_app_setting" {
  description = "Create f5xc_app_setting resource"
  type        = bool
  default     = false
}

variable "f5xc_healthcheck" {
  description = "Create f5xc_healthcheck resource"
  type        = bool
  default     = false
}

variable "f5xc_virtual_k8s" {
  description = "Create f5xc_virtual_k8s resource"
  type        = bool
  default     = false
}

variable "f5xc_origin_pool" {
  description = "Create f5xc_origin_pool resource"
  type        = bool
  default     = false
}

variable "f5xc_alert_policy" {
  description = "Create f5xc_alert_policy resource"
  type        = bool
  default     = false
}

variable "f5xc_virtual_site" {
  description = "Create f5xc_virtual_site resource"
  type        = bool
  default     = false
}

variable "f5xc_app_firewall" {
  description = "Create f5xc_app_firewall resource"
  type        = bool
  default     = false
}

variable "f5xc_service_policy" {
  description = "Create f5xc_service_policy resource"
  type        = bool
  default     = false
}

variable "f5xc_v1_dns_monitor" {
  description = "Create f5xc_v1_dns_monitor resource"
  type        = bool
  default     = false
}

variable "f5xc_voltstack_site" {
  description = "Create f5xc_voltstack_site resource"
  type        = bool
  default     = false
}

variable "f5xc_alert_receiver" {
  description = "Create f5xc_alert_receiver resource"
  type        = bool
  default     = false
}

variable "f5xc_virtual_network" {
  description = "Create f5xc_virtual_network resource"
  type        = bool
  default     = false
}

variable "f5xc_site_mesh_group" {
  description = "Create f5xc_site_mesh_group resource"
  type        = bool
  default     = false
}

variable "f5xc_network_firewall" {
  description = "Create f5xc_network_firewall resource"
  type        = bool
  default     = false
}

variable "f5xc_http_loadbalancer" {
  description = "Create f5xc_http_loadbalancer resource"
  type        = bool
  default     = false
}

variable "f5xc_network_policy_rule" {
  description = "Create f5xc_network_policy_rule resource"
  type        = bool
  default     = false
}

variable "f5xc_forward_proxy_policy" {
  description = "Create f5xc_forward_proxy_policy resource"
  type        = bool
  default     = false
}

variable "f5xc_api_definition_uploader" {
  description = "Create f5xc_api_definition_uploader resource"
  type        = bool
  default     = false
}

variable "f5xc_enhanced_firewall_policy" {
  description = "Create f5xc_enhanced_firewall_policy resource"
  type        = bool
  default     = false
}

##############################################################################
# Resource output control flags
# Set to true to include the resource in the "resource" output block.
##############################################################################
variable "f5xc_route_output" {
  description = "Include f5xc_route in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_token_output" {
  description = "Include f5xc_token in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_namespace_output" {
  description = "Include f5xc_namespace in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_nat_policy_output" {
  description = "Include f5xc_nat_policy in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_app_setting_output" {
  description = "Include f5xc_app_setting in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_healthcheck_output" {
  description = "Include f5xc_healthcheck in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_virtual_k8s_output" {
  description = "Include f5xc_virtual_k8s in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_origin_pool_output" {
  description = "Include f5xc_origin_pool in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_alert_policy_output" {
  description = "Include f5xc_alert_policy in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_virtual_site_output" {
  description = "Include f5xc_virtual_site in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_app_firewall_output" {
  description = "Include f5xc_app_firewall in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_service_policy_output" {
  description = "Include f5xc_service_policy in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_v1_dns_monitor_output" {
  description = "Include f5xc_v1_dns_monitor in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_voltstack_site_output" {
  description = "Include f5xc_voltstack_site in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_alert_receiver_output" {
  description = "Include f5xc_alert_receiver in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_virtual_network_output" {
  description = "Include f5xc_virtual_network in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_site_mesh_group_output" {
  description = "Include f5xc_site_mesh_group in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_network_firewall_output" {
  description = "Include f5xc_network_firewall in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_http_loadbalancer_output" {
  description = "Include f5xc_http_loadbalancer in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_network_policy_rule_output" {
  description = "Include f5xc_network_policy_rule in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_forward_proxy_policy_output" {
  description = "Include f5xc_forward_proxy_policy in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_api_definition_uploader_output" {
  description = "Include f5xc_api_definition_uploader in the resource output"
  type        = bool
  default     = false
}

variable "f5xc_enhanced_firewall_policy_output" {
  description = "Include f5xc_enhanced_firewall_policy in the resource output"
  type        = bool
  default     = false
}

##############################################################################
# Data source control flags
# Set to true to query the data source for the corresponding resource.
# Note: route, nat_policy, app_setting, virtual_site and app_firewall have
#       no data source and are therefore not listed here.
##############################################################################
variable "f5xc_token_data" {
  description = "Query data source for f5xc_token"
  type        = bool
  default     = false
}

variable "f5xc_namespace_data" {
  description = "Query data source for f5xc_namespace"
  type        = bool
  default     = false
}

variable "f5xc_healthcheck_data" {
  description = "Query data source for f5xc_healthcheck"
  type        = bool
  default     = false
}

variable "f5xc_virtual_k8s_data" {
  description = "Query data source for f5xc_virtual_k8s"
  type        = bool
  default     = false
}

variable "f5xc_origin_pool_data" {
  description = "Query data source for f5xc_origin_pool"
  type        = bool
  default     = false
}

variable "f5xc_alert_policy_data" {
  description = "Query data source for f5xc_alert_policy"
  type        = bool
  default     = false
}

variable "f5xc_service_policy_data" {
  description = "Query data source for f5xc_service_policy"
  type        = bool
  default     = false
}

variable "f5xc_v1_dns_monitor_data" {
  description = "Query data source for f5xc_v1_dns_monitor"
  type        = bool
  default     = false
}

variable "f5xc_voltstack_site_data" {
  description = "Query data source for f5xc_voltstack_site"
  type        = bool
  default     = false
}

variable "f5xc_alert_receiver_data" {
  description = "Query data source for f5xc_alert_receiver"
  type        = bool
  default     = false
}

variable "f5xc_virtual_network_data" {
  description = "Query data source for f5xc_virtual_network"
  type        = bool
  default     = false
}

variable "f5xc_site_mesh_group_data" {
  description = "Query data source for f5xc_site_mesh_group"
  type        = bool
  default     = false
}

variable "f5xc_network_firewall_data" {
  description = "Query data source for f5xc_network_firewall"
  type        = bool
  default     = false
}

variable "f5xc_http_loadbalancer_data" {
  description = "Query data source for f5xc_http_loadbalancer"
  type        = bool
  default     = false
}

variable "f5xc_tcp_loadbalancer_data" {
  description = "Query data source for f5xc_tcp_loadbalancer"
  type        = bool
  default     = false
}

variable "f5xc_network_policy_rule_data" {
  description = "Query data source for f5xc_network_policy_rule"
  type        = bool
  default     = false
}

variable "f5xc_forward_proxy_policy_data" {
  description = "Query data source for f5xc_forward_proxy_policy"
  type        = bool
  default     = false
}

variable "f5xc_enhanced_firewall_policy_data" {
  description = "Query data source for f5xc_enhanced_firewall_policy"
  type        = bool
  default     = false
}

variable "f5xc_securemesh_site_v2_data" {
  description = "Query data source for f5xc_securemesh_site_v2"
  type        = bool
  default     = false
}

##############################################################################
# Data source output control flags
# Set to true to include the data source in the "data" output block.
##############################################################################
variable "f5xc_token_data_output" {
  description = "Include f5xc_token data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_namespace_data_output" {
  description = "Include f5xc_namespace data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_healthcheck_data_output" {
  description = "Include f5xc_healthcheck data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_virtual_k8s_data_output" {
  description = "Include f5xc_virtual_k8s data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_origin_pool_data_output" {
  description = "Include f5xc_origin_pool data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_alert_policy_data_output" {
  description = "Include f5xc_alert_policy data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_service_policy_data_output" {
  description = "Include f5xc_service_policy data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_v1_dns_monitor_data_output" {
  description = "Include f5xc_v1_dns_monitor data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_voltstack_site_data_output" {
  description = "Include f5xc_voltstack_site data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_alert_receiver_data_output" {
  description = "Include f5xc_alert_receiver data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_virtual_network_data_output" {
  description = "Include f5xc_virtual_network data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_site_mesh_group_data_output" {
  description = "Include f5xc_site_mesh_group data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_network_firewall_data_output" {
  description = "Include f5xc_network_firewall data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_http_loadbalancer_data_output" {
  description = "Include f5xc_http_loadbalancer data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_tcp_loadbalancer_data_output" {
  description = "Include f5xc_tcp_loadbalancer data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_network_policy_rule_data_output" {
  description = "Include f5xc_network_policy_rule data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_forward_proxy_policy_data_output" {
  description = "Include f5xc_forward_proxy_policy data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_enhanced_firewall_policy_data_output" {
  description = "Include f5xc_enhanced_firewall_policy data source in the data output"
  type        = bool
  default     = false
}

variable "f5xc_securemesh_site_v2_data_output" {
  description = "Include f5xc_securemesh_site_v2 data source in the data output"
  type        = bool
  default     = false
}
