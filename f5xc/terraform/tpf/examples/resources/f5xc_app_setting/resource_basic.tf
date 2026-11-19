# Example: App Setting

# App Setting references an App Type defined in the shared namespace.
resource "f5xc_app_type" "example" {
  name        = "example-app-type"
  namespace   = "shared"
  description = "Application profile used by the App Setting example"

  features = [
    {
      type = ["BUSINESS_LOGIC_MARKUP"]
    },
    {
      type = ["PER_REQ_ANOMALY_DETECTION"]
    }
  ]
}

# Only one App Setting can be created in a namespace.
resource "f5xc_app_setting" "example" {
  name        = "example-app-setting"
  namespace   = "default"
  description = "Application monitoring and anomaly detection settings"

  labels = {
    environment = "example"
    team        = "security"
  }

  annotations = {
    owner = "platform-team"
  }

  app_type_settings = [
    {
      app_type_ref = [
        {
          name      = f5xc_app_type.example.name
          namespace = f5xc_app_type.example.namespace
        }
      ]

      business_logic_markup_setting = {
        learn_from_namespace_choice = {
          enable = true
        }
      }

      timeseries_analyses_setting = {
        metric_selectors = [
          {
            metrics_source = ["NODES"]
            metric         = ["REQUEST_RATE", "ERROR_RATE"]
          }
        ]
      }
    }
  ]
}
