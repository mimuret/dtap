input "pcap" "default" {
  device = "en0"
  forward_to = ["output.stdout.default"]
}

output "stdout" "default" {}

