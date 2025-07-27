input "file" "default" {
  path = "/data/dnstap.fstrm"
  forward_to = [
    "output.unix.default",
    "output.stdout.default",
    "output.tcp.default",
    "output.nats.default",
    ]
  labels = {
    "input_plugin" = "input.file.default"
  }
}

output "unix" "default" {
  path = "/run/unix.sock"
}


output "tcp" "default" {
  host = "dtap2"
  port = 12345
}

output "nats" "default" {
  hosts = ["nats://nats:4222"]
  subject = "dtap"
}

output "stdout" "default" {
  format = "go-template"
	template = "{{ .Message.Timestamp }} in:{{ index .Labels \"input_plugin\" }} {{ .Message.Type }} {{ .Message.Qclass }} {{ .Message.Qtype }} {{ .Message.Qname }}"
}