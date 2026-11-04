# Example: Api Definition - Basic

resource "f5xc_api_definition" "example" {
  name      = "example-api-definition"
  namespace = "shared"

  api_inventory_inclusion_list = [
    {
      method = ["GET"]
      path   = "/v1/orders/{order_id}"
    }
  ]
  schema_updates_strategy_choice = {
    strict_schema_origin = true
  }
}
