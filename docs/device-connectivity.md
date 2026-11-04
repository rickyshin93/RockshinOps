# Device connectivity

The **Devices / 设备连接** sidebar page is separate from **Connections**, which inspects process network sockets. This page is implemented on macOS. Windows and Linux explicitly report it as unsupported.

## Sources and behavior

| Area | Source | Behavior and limits |
| --- | --- | --- |
| Wi-Fi | CoreWLAN shared `CWWiFiClient` | Current interface, connected state, SSID when available, RSSI/noise, negotiated link rate, channel/band/width, PHY and security. Connected state does not depend on a potentially redacted SSID. Link rate is not measured network throughput. |
| Nearby Wi-Fi | `CWInterface.scanForNetworksWithSSID` | Explicit scan only, off the main thread. Requires Location access. One strongest result per SSID/band; unnamed networks are omitted. Results retain their scan time. Joining networks remains in System Settings. |
| Bluetooth | IOBluetooth paired devices | Lists the actual paired-device names and connection states. Connect/disconnect rechecks membership in the paired list. `openConnection` establishes a Bluetooth link; successful completion does not promise a usable audio profile. The UI follows the next system state. Power and pairing stay in System Settings. |
| Bluetooth battery | IORegistry `AppleDeviceManagementHIDEventService` / `BatteryPercent` | Only valid 0–100 values joined by exact normalized `DeviceAddress`. No name matching or private preference files. Many accessories, including some headphones, do not expose a compatible reading. A missing value is not zero. Registry class/property coverage varies by device and macOS release. |
| Audio | CoreAudio output stream configuration and device UID | Lists devices with output channels and selects the default output. Volume uses `kAudioHardwareServiceDeviceProperty_VirtualMainVolume`; fixed-volume outputs remain selectable but have a disabled slider. Selection affects other apps. |

Volume is committed on release or keyboard change. The displayed device UID is retained during a drag, and the native boundary verifies that it is still the default output before writing. Rapid keyboard changes coalesce to the latest user-requested value. Responses from earlier adjustments preserve newer in-progress slider edits and their original output UID. A route change causes an explicit error instead of changing the new route's volume.

The Go device monitor starts on first visit and samples independently every `deviceRefreshSeconds` from root `rockshinops.json` (default 5 seconds). Discovery never scans, connects devices or requests permission. A failed read is explicit. The page keeps scan results separate from live readings, and live updates preserve disclosure and slider interaction state.

## Permissions

Location and Bluetooth authorization are requested only from their page buttons. Denied, restricted or disabled access routes to the relevant System Settings pane. No location coordinates are requested or recorded. The packaged app declares `NSLocationUsageDescription`, `NSLocationWhenInUseUsageDescription` and `NSBluetoothAlwaysUsageDescription`.

Apple documents [Location authorization and required usage descriptions](https://developer.apple.com/documentation/corelocation/requesting-authorization-to-use-location-services) and [SSID access restrictions](https://developer.apple.com/forums/thread/759044). CoreWLAN [scan calls block their calling thread](https://developer.apple.com/documentation/corewlan/cwinterface/scanfornetworks(withssid:)); device scans therefore run in Go's native service, outside AppKit's main thread. Bluetooth controls use the public [IOBluetoothDevice](https://developer.apple.com/documentation/iobluetooth/iobluetoothdevice) API.

Native acceptance must distinguish discovery from real control operations and permission prompts. Current implementation progress and validation limits live in `STATUS.md`.

## Plain-language Wi-Fi overview

The overview has a shared white card. Wi-Fi shows strong/fair/weak signal inline beside its title, with practical guidance in Technical details. It does not claim internet reachability, download speed, latency or connection stability. Disconnected or powered-off Wi-Fi does not reuse a previous signal grade; a missing reading stays unknown. Raw readings, the negotiated link-rate explanation and source are available under the default-open Technical details disclosure. Nearby-network results use the same signal labels. Signal icons light three green bars for strong, two blue for fair, and one amber for weak; missing or disconnected readings light no bars. Text remains alongside the icons. Both disclosures retain the user’s open or closed state during live updates, including permission or redacted-name notices appearing and disappearing.

Root `rockshinops.json` owns `wifiSignalThresholds`: strong at or above −67 dBm, fair from −70 to below −67 dBm, weak below −70 dBm. These are approximate presentation bands based on [Cisco’s RSSI guidance](https://www.cisco.com/c/en/us/support/docs/smb/wireless/CB-Wireless-Mesh/1902-tz-Troubleshooting-Traditional-Cisco-Wireless-Network.html), not an OS-provided quality verdict or a speed test. The thresholds appear in Technical details and are bundled into the frontend.
