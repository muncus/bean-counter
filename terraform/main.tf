
provider "google" {
    project = "alert-impulse-331822"
    region = "us-west1"
    zone = "us-west1-b"
}  

data "google_client_config" "current" {}