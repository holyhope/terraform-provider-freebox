# `freebox_api_version` (Data Source)

Discovery of the Freebox over HTTP(S)

## Example

```terraform
data "freebox_api_version" "example" {}

output "box_model" {
  value = data.freebox_api_version.example.box_model
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `api_base_url` (String) The API root path on the HTTP server
- `api_domain` (String) The domain to use in place of hardcoded Freebox IP
- `api_version` (String) The current API version on the Freebox
- `box_model` (String) Box model
- `box_model_name` (String) Box model display name
- `https_available` (Boolean) Tells if HTTPS has been configured on the Freebox
- `https_port` (Number) Port to use for remote https access to the Freebox API
- `uid` (String) The device unique id
