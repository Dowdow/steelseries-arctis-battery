# SteelSeries Arctis Battery

This is a simple Go application that allows you to view the battery level of your SteelSeries Arctis gaming headset on your SteelSeries keyboard with a screen.

On Windows, it scans HID devices through `setupapi.dll` and reads battery information from the headset through `hid.dll`. On Linux, it uses the kernel `hidraw` interface.

Two interchangeable output backends can display the battery level on the keyboard screen, each of them can be toggled from the tray menu:

- **GameSense** — sends the level to the SteelSeriesGG app through the [SteelSeries GameSense™ SDK](https://github.com/SteelSeries/gamesense-sdk). Windows/macOS only, since SteelSeriesGG does not exist on Linux.
- **Direct HID** — draws the frame (name, percentage, progress bar) straight on the OLED screen of SteelSeries Apex keyboards, without SteelSeriesGG. This is the default on Linux. On Windows, enable it only if SteelSeriesGG is not running, otherwise both will fight over the screen. The screen is cleared (blanked) whenever this backend is turned off or the app quits — there is no known command to hand the screen back to the keyboard's own firmware display, so a blank screen is the closest available reset.

The app is visible and manageable from the system tray, using the [fyne-io/systray](https://github.com/fyne-io/systray) library. The tray icon itself is generated at runtime: the plain black headphone silhouette gets a level-colored circle badge (red/orange/green) bottom-right, a mirrored light blue circle badge bottom-left while charging, and the percentage stamped across the middle in a blocky "digital readout" font, matching the font used on the OLED screen. All flat fills, no outlines — the tooltip mirrors the same info as plain text.

## Screenshots

|Scanning|Progress|Systray|Systray Click|SteelSeriesGG|
|--------|--------|-------|-------------|-------------|
|<img src="https://github.com/Dowdow/steelseries-arctis-battery/blob/main/screenshots/scanning.jpg?raw=true" alt="Scanning example" height="150" />|<img src="https://github.com/Dowdow/steelseries-arctis-battery/blob/main/screenshots/progress.jpg?raw=true" alt="Progress example" height="150" />|<img src="https://github.com/Dowdow/steelseries-arctis-battery/blob/main/screenshots/tray.png?raw=true" alt="Systray example" />|<img src="https://github.com/Dowdow/steelseries-arctis-battery/blob/main/screenshots/tray-click.png?raw=true" alt="Systray click example" />|<img src="https://github.com/Dowdow/steelseries-arctis-battery/blob/main/screenshots/sse.png?raw=true" alt="SteelSeriesGG example" />|

## Support

The app currently only supports the [Arctis 7 (2019)](https://steelseries.com/gaming-headsets/arctis-7) headset. The headset table in `internal/headset/headset.go` allows you to add support for other models — pull requests are welcome, as long as they don't break existing platforms.

Charging status is also detected for the Arctis 7, shown as a bolt icon next to the percentage on the keyboard screen and a `+` suffix in the GameSense text. The headset's own HID report never carries this information (confirmed against [HeadsetControl](https://github.com/Sapd/HeadsetControl)'s implementation) — instead, an extra "Arctis 7 Bootloader" USB interface appears on the bus only while charging over USB, and its mere presence is used as the signal.

The direct HID backend knows the OLED screen of the Apex Pro, Apex 7, Apex Pro TKL and Apex 7 TKL keyboards (table in `internal/oled/oled.go`). The protocol is reverse-engineered (report `0x61` carrying a 128x40 1-bit frame), only the Apex 7 has been tested.

| Platform | Battery scan | Tray icon | Screen via GameSense | Screen via direct HID |
|----------|--------------|-----------|----------------------|-----------------------|
| Windows  | ✅ | ✅ | ✅ | ⚠️ conflicts with SteelSeriesGG |
| Linux    | ✅ | ✅ | ❌ no SteelSeriesGG on Linux | ✅ |
| macOS    | untested | untested | untested | untested |

If you have any questions, feel free to open an issue.

### Adding a new headset

1. Find the vendor/product IDs of the wireless dongle and the HID command that returns the battery level. [HeadsetControl](https://github.com/Sapd/HeadsetControl) is a good protocol reference for most SteelSeries models.
2. Add an entry to the `headsets` table in `internal/headset/headset.go`. If the device doesn't answer with a plain percentage (status byte, 0-4 scale...), set the optional `parseLevel` function.
3. If you want to try wiring up charging detection: on the Arctis 7 (2019), the battery HID report carries no charging status at all — instead, an extra HID interface (`0x1038:0x12AE`, "Arctis 7 Bootloader") only shows up on the bus while charging over USB. Check for a similar "extra interface appears while charging" tell with `lsusb`/`dmesg` before and after plugging your headset's cable, and wire the product ID into the optional `chargingProductId` field. Only ever check its *presence*, never open or write to it — it's a firmware-update interface.
4. On Linux, add a udev line for the product ID in `contrib/70-steelseries-arctis-battery.rules`.
5. Validate with the headset connected: `go test -v -run TestLiveBatteryRead ./internal/headset/`

### Adding a new keyboard

1. Add an entry to the `keyboards` table in `internal/oled/oled.go`. The backend assumes a 128x40 OLED screen addressed by a flat framebuffer in output report `0x61`, which is the case for the Apex Pro, Apex 7, Apex 7 TKL, Apex Pro TKL and Apex 5. The right HID interface (usage page `0xFFC0`) is selected automatically.

   **Gen3 keyboards use a different protocol and are not supported by this backend.** [apex-tux](https://github.com/not-jan/apex-tux) — the reference implementation this protocol was reverse-engineered from — shows that Apex Pro Gen3 (product ID `0x1640`) uses an SSD1306 page-major encoding, and the wireless Apex Pro TKL Gen3 (`0x1646`/`0x1644` dongle) uses chunked feature reports. Adding one of those product IDs here as-is will silently draw garbage instead of the intended frame; they'd need their own `screen` implementation and write path.
2. On Linux, add a udev line for the product ID in `contrib/70-steelseries-arctis-battery.rules`.
3. Validate with the keyboard connected: `go test -v -run TestLiveOLEDWrite ./internal/oled/` — the test frame must appear on the screen, then be cleared right after (the test also exercises the cleanup path).

## Linux

Linux support is experimental. The tray icon shows the battery level (colorized headphone + percentage) and the keyboard screen is driven directly over HID.

By default, `/dev/hidraw*` devices are only accessible by root. Install the provided udev rules (headset dongle + keyboards) to grant access to the logged-in user:

```bash
sudo cp contrib/70-steelseries-arctis-battery.rules /etc/udev/rules.d/
sudo udevadm control --reload-rules && sudo udevadm trigger
```

## Development

Clone and download the dependencies:

```bash
git clone https://github.com/Dowdow/steelseries-arctis-battery.git
go mod download
```

### Windows

The `build.bat` script requires the `rsrc` package to assign an icon to the `.exe` file:

```bash
go install github.com/akavel/rsrc@latest
.\build.bat
```

It will create a `dist\build.exe` file that you can immediately run.

### Linux

```bash
./build.sh
```

It will create a `dist/steelseries-arctis-battery` binary.

### Project layout

- `internal/hid` — minimal cross-platform HID access (`setupapi.dll`/`hid.dll` on Windows, `hidraw` on Linux)
- `internal/headset` — headset definitions, scanning and battery polling
- `internal/display` — the battery message and the `Display` interface implemented by every output backend
- `internal/sse` — GameSense backend (registration, heartbeat, battery events)
- `internal/oled` — direct HID backend (frame rendering and writing to the Apex OLED screen)
- `internal/icon` — tray icon rendering: composites the level/charging badges and battery percentage onto the base headphone silhouette at runtime (ICO on Windows, PNG elsewhere)
