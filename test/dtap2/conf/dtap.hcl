
input "unix" "default" {
  path = "/run/unix.sock"
  forward_to = ["output.stdout.default"]
  labels = {
    "input_plugin" = "input.unix.default"
  }
}

input "tcp" "default" {
  address = "0.0.0.0"
  port = 12345
  forward_to = ["output.stdout.default"]
  labels = {
    "input_plugin" = "input.tcp.default"
  }
}

input "nats" "default" {
  hosts = ["nats://nats:4222"]
  subject = "dtap"
  forward_to = ["output.stdout.default"]
  labels = {
    "input_plugin" = "input.nats.default"
  }
}

output "stdout" "default" {
  format = "go-template"
	template = "{{ .Message.Timestamp }} in:{{ index .Labels \"input_plugin\" }} {{ .Message.Type }} {{ .Message.Qclass }} {{ .Message.Qtype }} {{ .Message.Qname }}"
}