import 'dart:math';

/// One tap of a phone on a checkpoint tag.
class Scan {
  final String scanId;
  final String tagId;
  final String guardId;
  final DateTime scannedAt;
  final String? ndefUrl;
  final String? tagType;

  Scan({
    required this.scanId,
    required this.tagId,
    required this.guardId,
    required this.scannedAt,
    this.ndefUrl,
    this.tagType,
  });

  /// Creates a scan with a fresh unique ID. The server uses scan_id to drop
  /// duplicates when a queued scan is retried.
  factory Scan.create({
    required String tagId,
    required String guardId,
    String? ndefUrl,
    String? tagType,
  }) {
    final now = DateTime.now().toUtc();
    final rand = Random.secure().nextInt(1 << 32).toRadixString(16).padLeft(8, '0');
    return Scan(
      scanId: '${now.microsecondsSinceEpoch}-$rand',
      tagId: tagId,
      guardId: guardId,
      scannedAt: now,
      ndefUrl: ndefUrl,
      tagType: tagType,
    );
  }

  Map<String, dynamic> toJson() => {
        'scan_id': scanId,
        'tag_id': tagId,
        'guard_id': guardId,
        'scanned_at': scannedAt.toIso8601String(),
        'ndef_url': ndefUrl,
        'tag_type': tagType,
      };

  factory Scan.fromJson(Map<String, dynamic> j) => Scan(
        scanId: j['scan_id'] as String,
        tagId: j['tag_id'] as String,
        guardId: j['guard_id'] as String,
        scannedAt: DateTime.parse(j['scanned_at'] as String),
        ndefUrl: j['ndef_url'] as String?,
        tagType: j['tag_type'] as String?,
      );
}
