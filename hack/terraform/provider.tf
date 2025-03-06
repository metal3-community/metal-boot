terraform {
  required_providers {
    redfish = {
      version = ">= 1.5.0"
      source  = "registry.terraform.io/dell/redfish"
    }
  }
}

provider "redfish" {
}