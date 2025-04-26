# Steelseries Arctis Battery

This is a simple Go application that allows you to view the battery level of your Steelseries Arctis gaming headset on your Steelseries keyboard with a screen.

It uses `setupapi.dll` to scan devices and `hid.dll` to retrieve battery information from the headset. This data is then sent to the SteelSeriesGG app through the [SteelSeries GameSense™ SDK](https://github.com/SteelSeries/gamesense-sdk).

The app is visible and manageable from the system tray, using the [fyne-io/systray](https://github.com/fyne-io/systray) library.

## Screenshots

|Scanning|Progress|Systray|Systray Click|SteelseriesGG|
|--------|--------|-------|-------------|-------------|
|<img src="https://github.com/Dowdow/steelseries-arctis-battery/blob/main/screenshots/scanning.jpg?raw=true" alt="Scanning example" height="150" />|<img src="https://github.com/Dowdow/steelseries-arctis-battery/blob/main/screenshots/progress.jpg?raw=true" alt="Progress example" height="150" />|<img src="https://github.com/Dowdow/steelseries-arctis-battery/blob/main/screenshots/tray.png?raw=true" alt="Systray example" />|<img src="https://github.com/Dowdow/steelseries-arctis-battery/blob/main/screenshots/tray-click.png?raw=true" alt="Systray click example" />|<img src="https://github.com/Dowdow/steelseries-arctis-battery/blob/main/screenshots/sse.png?raw=true" alt="SteelseriesGG example" height="200" />|

## Support

The app currently only supports the [Arctis 7 (2019)](https://steelseries.com/gaming-headsets/arctis-7) headset, and there are no plans to extend support to other models. The code in `headset.go` allows you to add support for other headsets, but I won't do it myself, as I can't test them.

For now, the app is only built for Windows. I can't test on macOS, even though the SteelSeriesGG app is available there. You’re welcome to open a pull request for that, as long as it doesn't break Windows compatibility.

If you have any questions, feel free to open an issue.
