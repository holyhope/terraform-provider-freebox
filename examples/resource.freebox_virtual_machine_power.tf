resource "freebox_virtual_machine" "example" {
  name      = "example-vm"
  vcpus     = 1
  memory    = 2048
  disk_path = "/Freebox/VMs/ubuntu.qcow2"
  disk_type = "qcow2"
  status    = "stopped"
}

resource "freebox_dhcp_static_lease" "example" {
  mac        = freebox_virtual_machine.example.mac
  ip         = "192.168.1.32"
  hostname   = "example-vm"
  depends_on = [freebox_virtual_machine.example]
}

resource "freebox_virtual_machine_power" "example" {
  vm_id       = freebox_virtual_machine.example.id
  power_state = "running"

  timeouts = {
    kill = "30s"
  }

  depends_on = [freebox_dhcp_static_lease.example]
}