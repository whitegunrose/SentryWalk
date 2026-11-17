import 'package:flutter_nfc_kit/flutter_nfc_kit.dart';
import 'package:ndef/ndef.dart' as ndef;

/// What we pull off a tag in one tap.
class TagRead {
  final String tagId; // UID as uppercase hex, no separators
  final String? ndefUrl; // first URL record on the tag, if any
  final String tagType;

  TagRead({required this.tagId, required this.ndefUrl, required this.tagType});
}

class NfcReader {
  Future<bool> isAvailable() async {
    final availability = await FlutterNfcKit.nfcAvailability;
    return availability == NFCAvailability.available;
  }

  /// Waits for a tag, reads its UID and any URL record, then closes the
  /// session. On iPhone this shows the system "Ready to Scan" sheet.
  Future<TagRead> readTag() async {
    try {
      final tag = await FlutterNfcKit.poll(
        timeout: const Duration(seconds: 20),
        iosAlertMessage: 'Hold your phone near the checkpoint tag',
      );

      String? url;
      if (tag.ndefAvailable == true) {
        try {
          final records = await FlutterNfcKit.readNDEFRecords(cached: false);
          for (final r in records) {
            if (r is ndef.UriRecord && r.uriString != null) {
              url = r.uriString;
              break;
            }
          }
        } catch (_) {
          // Blank or unformatted tags have no NDEF data; the UID is enough.
        }
      }

      await FlutterNfcKit.finish(iosAlertMessage: 'Checkpoint scanned');
      return TagRead(
        tagId: _normalizeId(tag.id),
        ndefUrl: url,
        tagType: tag.type.name,
      );
    } catch (e) {
      // Always close the session, or the next scan won't start.
      try {
        await FlutterNfcKit.finish(iosErrorMessage: 'Scan failed');
      } catch (_) {}
      rethrow;
    }
  }

  /// "04:a1:b2..." / "04 A1 B2" / "04a1b2" -> "04A1B2"
  static String _normalizeId(String id) =>
      id.replaceAll(RegExp(r'[^0-9a-fA-F]'), '').toUpperCase();
}
