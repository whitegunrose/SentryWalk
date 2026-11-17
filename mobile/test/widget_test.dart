import 'package:flutter_test/flutter_test.dart';
import 'package:sentrywalk/scan.dart';

void main() {
  test('Scan round-trips through JSON', () {
    final s = Scan.create(tagId: '04A1B2C3D4E5F6', guardId: 'G01');
    final back = Scan.fromJson(s.toJson());
    expect(back.scanId, s.scanId);
    expect(back.tagId, '04A1B2C3D4E5F6');
    expect(back.scannedAt, s.scannedAt);
  });

  test('Each scan gets a unique ID', () {
    final a = Scan.create(tagId: 'X', guardId: 'G01');
    final b = Scan.create(tagId: 'X', guardId: 'G01');
    expect(a.scanId, isNot(b.scanId));
  });
}
