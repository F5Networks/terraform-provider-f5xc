# Example: Api Crawler - Simple Login User

resource "f5xc_api_crawler" "example" {
  name      = "example-api-crawler"
  namespace = "default"

  domains = [
    {
      domain = "app.example.com"
      simple_login = {
        user = "example-user"
      }
    }
  ]
}
