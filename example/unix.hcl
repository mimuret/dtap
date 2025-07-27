input "unix" "default" {
  path = "/var/run/unboud/dnstap.sock"
  forward_to = ["output.stdout.default"]
}

output "stdout" "default" {}

