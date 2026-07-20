data "freebox_profiles" "all" {}

output "profiles" {
  value = data.freebox_profiles.all.profiles
}
