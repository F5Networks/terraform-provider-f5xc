# Example: Azure Vnet Site - Full

data "f5xc_azure_vnet_site" "example" {
  name      = f5xc_azure_vnet_site.example.name
  namespace = f5xc_azure_vnet_site.example.namespace
}
