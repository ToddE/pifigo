# /etc/pifigo/netplan.tpl

network:
  version: 2
  renderer: networkd
  wifis:
    # This is a template variable that will be replaced by the
    # 'wireless_interface' value from your config.yaml
    {{.WirelessInterface}}:
      dhcp4: true
      access-points:
        # These are template variables that will be replaced by the
        # user-submitted SSID and Password from the web form.
        "{{.SSID}}":
          password: "{{.Password}}"