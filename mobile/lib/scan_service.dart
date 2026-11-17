import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

import 'scan.dart';

/// Result of trying to deliver a scan.
class SendResult {
  final bool delivered; // false = saved in the offline queue
  final String? checkpoint; // name from the server's checkpoints.json
  final bool knownTag;

  SendResult({required this.delivered, this.checkpoint, this.knownTag = false});
}

/// Sends scans to the server. Scans that can't be sent (no signal, server
/// down) are kept on the phone and retried later, so nothing is lost on a
/// patrol through a dead zone.
class ScanService {
  static const _queueKey = 'pending_scans';
  static const _timeout = Duration(seconds: 8);

  final String serverUrl; // e.g. http://192.168.1.20:8000

  ScanService(this.serverUrl);

  Future<SendResult> submit(Scan scan) async {
    try {
      final body = await _post(scan);
      // Connection works again, so try to empty the queue too.
      unawaited(flushQueue());
      return SendResult(
        delivered: true,
        checkpoint: body['checkpoint'] as String?,
        knownTag: body['known_tag'] == true,
      );
    } catch (_) {
      await _enqueue(scan);
      return SendResult(delivered: false);
    }
  }

  /// Retries queued scans in order. Returns how many were sent.
  Future<int> flushQueue() async {
    final prefs = await SharedPreferences.getInstance();
    final pending = prefs.getStringList(_queueKey) ?? [];
    var sent = 0;
    for (final raw in pending) {
      try {
        await _post(Scan.fromJson(jsonDecode(raw) as Map<String, dynamic>));
        sent++;
      } catch (_) {
        break; // still offline; keep the rest for later
      }
    }
    // Re-read in case a new scan was queued while we were sending.
    final now = prefs.getStringList(_queueKey) ?? [];
    await prefs.setStringList(_queueKey, now.skip(sent).toList());
    return sent;
  }

  Future<int> pendingCount() async {
    final prefs = await SharedPreferences.getInstance();
    return (prefs.getStringList(_queueKey) ?? []).length;
  }

  Future<Map<String, dynamic>> _post(Scan scan) async {
    final res = await http
        .post(
          Uri.parse('$serverUrl/scans'),
          headers: {'Content-Type': 'application/json'},
          body: jsonEncode(scan.toJson()),
        )
        .timeout(_timeout);
    if (res.statusCode != 200) {
      throw Exception('Server returned ${res.statusCode}: ${res.body}');
    }
    return jsonDecode(res.body) as Map<String, dynamic>;
  }

  Future<void> _enqueue(Scan scan) async {
    final prefs = await SharedPreferences.getInstance();
    final pending = prefs.getStringList(_queueKey) ?? [];
    pending.add(jsonEncode(scan.toJson()));
    await prefs.setStringList(_queueKey, pending);
  }
}
