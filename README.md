# go-vee

Control Govee lights through a REST API written in Go.

## Endpoints

### Devices

`GET api/v1/devices`
Get a list of devices and their capabilities from the Govee cloud API.

`GET api/v1/devices/lan`
Get a list of devices and their capabilities that are connected to your Local Area Network (LAN).

---

### Control

`POST api/v1/devices/control`

You can make a request to `GET api/v1/devices` to get a list of capabilities for your devices.

Alternatively, you can [look at this reference](https://developer.govee.com/reference/get-you-devices) if you do not wish to use the Govee API at all.

Switch the light on

```json
{
  "sku": "H6022",
  "device": "XX:XX:XX:XX:XX:XX:XX:XX",
  "capability": {
    "type": "devices.capabilities.on_off",
    "instance": "powerSwitch",
    "value": 1
  }
}
```

Set brightness to 50%

```json
{
  "sku": "H6022",
  "device": "XX:XX:XX:XX:XX:XX:XX:XX",
  "capability": {
    "type": "devices.capabilities.range",
    "instance": "brightness",
    "value": 50
  }
}
```

Set the color to red

```json
{
  "sku": "H6022",
  "device": "XX:XX:XX:XX:XX:XX:XX:XX",
  "capability": {
    "type": "devices.capabilities.color_setting",
    "instance": "colorRgb",
    "value": 16711680
  }
}
```
