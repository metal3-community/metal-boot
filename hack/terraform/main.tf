data "redfish_system_boot" "system_boot" {

  redfish_server {
    endpoint = "http://localhost:8080"
    user = "admin"
    password = "admin"
  }

  system_id = "d8:3a:dd:61:4d:15"
}

output "system_boot" {
  value = {
    boot_order = data.redfish_system_boot.system_boot.boot_order
  }
}