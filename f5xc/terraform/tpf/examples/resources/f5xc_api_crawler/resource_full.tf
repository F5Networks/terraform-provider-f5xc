# Example: Api Crawler - Full

resource "f5xc_api_crawler" "example" {
  name        = "example-api-crawler"
  namespace   = "default"
  description = "API crawler full example"

  labels = {
    env  = "example"
    tier = "api"
  }

  domains = [
    {
      domain = "api.example.com"
      simple_login = {
        user = "apiuser"
        password = {
          secret_info_choice = {
            clear_secret_info = {
              url = "string:///ZXhhbXBsZS1wYXNzd29yZA=="
            }
          }
        }
      }
    },
    {
      domain = "public.example.com"
    }
  ]
}
