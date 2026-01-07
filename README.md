# Huegio

!Työmaa!

Sometimes I can't be asked to pick up my phone to dim or switch off the lights,
so I tried Go and made an app for desktop.

## Building & running

Depends on [Gio](https://gioui.org/doc/install) + some other dependencies

```bash
go init
go build .
go run .
```

To use the app you must be connected to the same WiFi as your hue bridge.
Upon first application start the user is asked to press the hue bridge
connect button to authenticate to the app and fetch all of the lights.
