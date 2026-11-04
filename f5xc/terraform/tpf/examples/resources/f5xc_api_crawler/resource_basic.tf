# Example: Api Crawler - Basic

resource "f5xc_api_crawler" "example" {
  name        = "example-api-crawler"
  namespace   = "default"
  description = "API crawler example"
  domains = [
    {
      domain = "example.com"
    }
  ]
}
