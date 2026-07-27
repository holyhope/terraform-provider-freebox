data "freebox_profiles" "all" {}

data "freebox_network_control" "example" {
  profile_id = data.freebox_profiles.all.profiles[0].id
}

output "current_mode" {
  value = data.freebox_network_control.example.current_mode
}
