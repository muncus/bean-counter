# Create a custom service. this can only be done through the API for non-GKE services.

resource "google_monitoring_custom_service" "beancounter" {
    service_id = "beancounter-svc"
    display_name = "Baby Monitoring: bean counts"
}

resource "google_monitoring_metric_descriptor" "changes" {
  description = "Diaper changes"
  display_name = "changes"
  type = "custom.googleapis.com/beancounter/changes"
  # FIXME: this also does not work, deliberately, for custom metrics.
  # metric_kind = "DELTA"
  metric_kind = "GAUGE"
  value_type = "INT64"
}

resource "google_monitoring_metric_descriptor" "feedings" {
  description = "feeding schedule"
  display_name = "feedings"
  type = "custom.googleapis.com/beancounter/feedings"
  metric_kind = "GAUGE"
  value_type = "INT64"
}

resource "google_monitoring_metric_descriptor" "weight" {
  description = "baby weight"
  display_name = "weight"
  type = "custom.googleapis.com/beancounter/weight"
  metric_kind = "GAUGE"
  value_type = "DOUBLE"
}


# same purpose as string status, but uses a label and boolean value, instead of string.
resource "google_monitoring_metric_descriptor" "status-label" {
  description = "current conditions"
  display_name = "status"
  type = "custom.googleapis.com/beancounter/status-label"
  metric_kind = "GAUGE"
  value_type = "INT64"
  labels {
    key = "status"
    value_type = "STRING"
    description = "bean's mood"
  }
}

# A rudimentary dashboard to show some simple stats via bar graph.
# JSON content comes directly from the dashboard editing interface!
resource "google_monitoring_dashboard" "dash" {
  dashboard_json = <<EOF
  {
  "category": "CUSTOM",
  "displayName": "test dashboard",
  "mosaicLayout": {
    "columns": 12,
    "tiles": [
      {
        "height": 4,
        "widget": {
          "title": "changes in sliding 30m window",
          "xyChart": {
            "chartOptions": {
              "mode": "COLOR"
            },
            "dataSets": [
              {
                "plotType": "STACKED_BAR",
                "targetAxis": "Y1",
                "timeSeriesQuery": {
                  "timeSeriesQueryLanguage": "fetch global::custom.googleapis.com/beancounter/changes\n| group_by sliding(30m), [value_changes_sum: sum(value.changes)]"
                }
              }
            ],
            "timeshiftDuration": "0s",
            "yAxis": {
              "label": "y1Axis",
              "scale": "LINEAR"
            }
          }
        },
        "width": 6,
        "xPos": 0,
        "yPos": 0
      }
    ]
  }
}

EOF  
}

# TODO: make this work, by adding the required storage bucket params.
# resource "google_cloudfunctions_function" "functions-changes" {
#   name = "changes"
#   description = "Bean Counter: Changed"
#   runtime = "go116"
#   trigger_http = true
#   environment_variables  = {
#     GOOGLE_PROJECT_ID = data.google_client_config.current.project
#   }
# }


# FIXME: this one is broken, because custom string-type values are not supported :(
  # Error: Error creating MetricDescriptor: googleapi: Error 400: Field
  # metricDescriptor.valueType had an invalid value of "STRING": When creating
  # metric custom.googleapis.com/beancounter/status: the value type is not
  # supported for custom metrics.
# resource "google_monitoring_metric_descriptor" "status" {
#   description = "current conditions"
#   display_name = "status"
#   type = "custom.googleapis.com/beancounter/status"
#   metric_kind = "GAUGE"
#   value_type = "STRING"
# }