data "freebox_netshare_samba_configuration" "example" {}

output "workgroup" {
  value = data.freebox_netshare_samba_configuration.example.workgroup
}
