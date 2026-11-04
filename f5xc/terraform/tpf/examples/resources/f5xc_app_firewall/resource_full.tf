# Example: App Firewall - Full

resource "f5xc_app_firewall" "example" {
  name        = "example-app-firewall"
  namespace   = "default"
  description = "App firewall example"

  labels = {
    env  = "example"
    team = "security"
  }

  annotations = {
    owner = "platform"
  }

  enforcement_mode_choice = {
    blocking = true
  }

  detection_setting_choice = {
    detection_settings = {
      signatures_staging_settings_choice = {
        stage_new_signatures = {
          staging_period = 7
        }
      }
      bot_protection_choice = {
        bot_protection_setting = {
          good_bot_action       = ["IGNORE"]
          malicious_bot_action  = ["BLOCK"]
          suspicious_bot_action = ["REPORT"]
        }
      }
      violations_view = []
      signature_selection_setting = {
        signature_selection_by_accuracy_choice = {
          high_medium_accuracy_signatures = true
        }
        attack_type_setting_choice = {
          attack_type_settings = {
            disabled_attack_types = ["ATTACK_TYPE_NON_BROWSER_CLIENT"]
          }
        }
      }
      false_positive_suppression_choice = {
        enable_suppression = true
      }
      threat_campaign_choice = {
        enable_threat_campaigns = true
      }
    }
  }

  allowed_response_codes_choice = {
    allowed_response_codes = {
      response_code = [200, 201, 204, 301, 302, 400, 404]
    }
  }

  anonymization_setting_choice = {
    custom_anonymization = {
      anonymization_config = [
        {
          anonymization_choice = {
            http_header = {
              header_name = "x-api-key"
            }
          }
        },
        {
          anonymization_choice = {
            query_parameter = {
              query_param_name = "password"
            }
          }
        }
      ]
    }
  }

  blocking_page_choice = {
    blocking_page = {
      blocking_page = "https://example.com/f5xc-request-blocked"
      response_code = ["Forbidden"]
    }
  }

  enhance_with_ai_choice = {
    enable_ai_enhancements = {
      risk_score_action_choice = {
        mitigate_high_risk_action = true
      }
    }
  }
}
