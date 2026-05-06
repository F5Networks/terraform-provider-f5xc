output "resource" {
  sensitive = false
  value = {
    route                    = var.f5xc_route_output                    ? f5xc_route.example[0]                   : null
    token                    = var.f5xc_token_output                    ? f5xc_token.t1[0]                        : null
    namespace                = var.f5xc_namespace_output                ? f5xc_namespace.ns1[0]                   : null
    nat_policy               = var.f5xc_nat_policy_output               ? f5xc_nat_policy.natp1[0]                : null
    app_setting              = var.f5xc_app_setting_output              ? f5xc_app_setting.apps-1[0]              : null
    healthcheck              = var.f5xc_healthcheck_output              ? f5xc_healthcheck.hc1[0]                 : null
    virtual_k8s              = var.f5xc_virtual_k8s_output              ? f5xc_virtual_k8s.vk8s1[0]               : null
    origin_pool              = var.f5xc_origin_pool_output              ? f5xc_origin_pool.op[0]                  : null
    alert_policy             = var.f5xc_alert_policy_output             ? f5xc_alert_policy.ap1[0]                : null
    virtual_site             = var.f5xc_virtual_site_output             ? f5xc_virtual_site.vs1[0]                : null
    app_firewall             = var.f5xc_app_firewall_output             ? f5xc_app_firewall.app-fw1[0]            : null
    service_policy           = var.f5xc_service_policy_output           ? f5xc_service_policy.sp1[0]              : null
    v1_dns_monitor           = var.f5xc_v1_dns_monitor_output           ? f5xc_v1_dns_monitor.v1_dns_mon1[0]      : null
    voltstack_site           = var.f5xc_voltstack_site_output           ? f5xc_voltstack_site.vss1[0]             : null
    alert_receiver           = var.f5xc_alert_receiver_output           ? f5xc_alert_receiver.ar1[0]              : null
    virtual_network          = var.f5xc_virtual_network_output          ? f5xc_virtual_network.vn1[0]             : null
    site_mesh_group          = var.f5xc_site_mesh_group_output          ? f5xc_site_mesh_group.smg1[0]            : null
    network_firewall         = var.f5xc_network_firewall_output         ? f5xc_network_firewall.nfw1[0]           : null
    http_loadbalancer        = var.f5xc_http_loadbalancer_output        ? f5xc_http_loadbalancer.lb[0]            : null
    network_policy_rule      = var.f5xc_network_policy_rule_output      ? f5xc_network_policy_rule.nfwr1[0]       : null
    forward_proxy_policy     = var.f5xc_forward_proxy_policy_output     ? f5xc_forward_proxy_policy.fpp1[0]       : null
    api_definition_uploader  = var.f5xc_api_definition_uploader_output  ? f5xc_api_definition_uploader.swagger[0] : null
    enhanced_firewall_policy = var.f5xc_enhanced_firewall_policy_output ? f5xc_enhanced_firewall_policy.efp1[0]   : null
  }
}

output "data" {
  sensitive = false
  value = {
    token                    = var.f5xc_token_data_output                    ? data.f5xc_token.t1[0]                      : null
    namespace                = var.f5xc_namespace_data_output                ? data.f5xc_namespace.ns1[0]                 : null
    healthcheck              = var.f5xc_healthcheck_data_output              ? data.f5xc_healthcheck.hc1[0]               : null
    virtual_k8s              = var.f5xc_virtual_k8s_data_output              ? data.f5xc_virtual_k8s.vk8s1[0]             : null
    origin_pool              = var.f5xc_origin_pool_data_output              ? data.f5xc_origin_pool.op1[0]               : null
    alert_policy             = var.f5xc_alert_policy_data_output             ? data.f5xc_alert_policy.ap1[0]              : null
    service_policy           = var.f5xc_service_policy_data_output           ? data.f5xc_service_policy.sp1[0]            : null
    v1_dns_monitor           = var.f5xc_v1_dns_monitor_data_output           ? data.f5xc_v1_dns_monitor.v1_dns_mon1[0]    : null
    voltstack_site           = var.f5xc_voltstack_site_data_output           ? data.f5xc_voltstack_site.vss1[0]           : null
    alert_receiver           = var.f5xc_alert_receiver_data_output           ? data.f5xc_alert_receiver.ar1[0]            : null
    virtual_network          = var.f5xc_virtual_network_data_output          ? data.f5xc_virtual_network.vn1[0]           : null
    site_mesh_group          = var.f5xc_site_mesh_group_data_output          ? data.f5xc_site_mesh_group.smg1[0]          : null
    network_firewall         = var.f5xc_network_firewall_data_output         ? data.f5xc_network_firewall.nfw1[0]         : null
    http_loadbalancer        = var.f5xc_http_loadbalancer_data_output        ? data.f5xc_http_loadbalancer.lb[0]          : null
    tcp_loadbalancer         = var.f5xc_tcp_loadbalancer_data_output         ? data.f5xc_tcp_loadbalancer.tcp-lb1[0]      : null
    network_policy_rule      = var.f5xc_network_policy_rule_data_output      ? data.f5xc_network_policy_rule.npr1[0]      : null
    forward_proxy_policy     = var.f5xc_forward_proxy_policy_data_output     ? data.f5xc_forward_proxy_policy.fpp1[0]     : null
    enhanced_firewall_policy = var.f5xc_enhanced_firewall_policy_data_output ? data.f5xc_enhanced_firewall_policy.efp1[0] : null
    securemesh_site_v2       = var.f5xc_securemesh_site_v2_data_output       ? data.f5xc_securemesh_site_v2.smv2[0]       : null
  }
}
