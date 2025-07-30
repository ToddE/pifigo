# This template is used by the pifigo application to generate the client config.
network:
  version: 2
  renderer: NetworkManager
  wifis:
    {{.WirelessInterface}}:
    {{- if eq .ConnectionMode "dhcp" }}
      dhcp4: true
    {{- else }}
      dhcp4: no
      addresses:
        - {{.StaticIP}}
      routes:
        - to: default
          via: {{.Gateway}}
      nameservers:
        addresses:
        {{- range .DNSServers }}
          - {{.}}
        {{- end }}
    {{- end }}
      access-points:
        "{{.SSID}}":
          password: "{{.Password}}"
