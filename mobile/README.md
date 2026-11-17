# SentryWalk mobile app (Flutter)

Scans an NFC checkpoint tag and POSTs it to the server in `../server`.
Works on Android and iPhone. Scans made with no connection are queued on the phone and sent later.

| File | What it does |
|---|---|
| `lib/nfc_reader.dart` | Reads the tag UID and any URL stored on it (`flutter_nfc_kit`) |
| `lib/scan_service.dart` | POSTs to the server; offline queue in `shared_preferences` |
| `lib/scan.dart` | Scan data model + JSON |
| `lib/main.dart` | UI: scan button, history, settings |

## Setup

Only `lib/`, `test/` and `pubspec.yaml` are included. Generate the Android/iOS project folders around them first:

```bash
cd mobile
flutter create --org com.sentrywalk --platforms=android,ios .
flutter pub get
```

(`flutter create .` doesn't overwrite existing files, so it keeps the included code.)

**Only one person needs to do this.** Commit the generated `android/` and `ios/` folders, with the edits below, so teammates can just `flutter pub get` and `flutter run`.

Then make the platform edits below. **NFC won't work without them.**

### Android - `android/app/src/main/AndroidManifest.xml`

Inside `<manifest>`, above `<application>`:

```xml
<uses-permission android:name="android.permission.NFC" />
<uses-permission android:name="android.permission.INTERNET" />
<uses-feature android:name="android.hardware.nfc" android:required="false" />
```

On the `<application ...>` tag, add (dev only - allows plain `http://` to your Mac):

```xml
android:usesCleartextTraffic="true"
```

Run: plug in the phone with USB debugging on, then `flutter run`.

### iPhone

1. Open `ios/Runner.xcworkspace` in Xcode → Runner target → **Signing & Capabilities** → **+ Capability** → **Near Field Communication Tag Reading**.
   You'll likely need a paid Apple Developer account for this capability; free personal teams don't usually support it.
2. Add to `ios/Runner/Info.plist` inside the top `<dict>`:

```xml
<key>NFCReaderUsageDescription</key>
<string>Scan checkpoint tags during patrols.</string>

<!-- Needed to read Type 4 tags like NTAG 424 DNA -->
<key>com.apple.developer.nfc.readersession.iso7816.select-identifiers</key>
<array>
  <string>D2760000850101</string>
  <string>D2760000850100</string>
</array>

<key>NSLocalNetworkUsageDescription</key>
<string>Send scans to the patrol server on this network.</string>

<!-- DEV ONLY: allows plain http:// to your Mac. Remove once the server uses https. -->
<key>NSAppTransportSecurity</key>
<dict>
  <key>NSAllowsArbitraryLoads</key>
  <true/>
</dict>
```

3. Make sure the iOS deployment target is **13.0 or higher** (Runner target → General → Minimum Deployments, and `platform :ios, '13.0'` in `ios/Podfile`).
4. Run on a real iPhone (`flutter run`). The simulator has no NFC.

---

## Using the app

1. Tap the gear icon → set **Server URL** (e.g. `http://192.168.1.20:8000`) and **Guard ID**.
2. Tap **Scan checkpoint** and hold the phone's top back edge (iPhone: the top edge) against the tag.
3. Results show as:
   - **Green**: the checkpoint name.
   - **Amber**: unregistered tag.
   - **Orange**: saved offline. Queued scans sync on the next successful scan, or tap **N pending** in the top bar.
