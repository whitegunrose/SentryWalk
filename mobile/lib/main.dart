import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'nfc_reader.dart';
import 'scan.dart';
import 'scan_service.dart';

void main() => runApp(const SentryWalkApp());

class SentryWalkApp extends StatelessWidget {
  const SentryWalkApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'SentryWalk',
      theme: ThemeData(colorSchemeSeed: Colors.indigo, useMaterial3: true),
      darkTheme: ThemeData(
        colorSchemeSeed: Colors.indigo,
        brightness: Brightness.dark,
        useMaterial3: true,
      ),
      home: const ScanScreen(),
    );
  }
}

/// A row in the on-screen history.
class _LogEntry {
  final DateTime time;
  final String tagId;
  final String label;
  final IconData icon;
  final Color color;
  _LogEntry(this.time, this.tagId, this.label, this.icon, this.color);
}

class ScanScreen extends StatefulWidget {
  const ScanScreen({super.key});

  @override
  State<ScanScreen> createState() => _ScanScreenState();
}

class _ScanScreenState extends State<ScanScreen> {
  final _nfc = NfcReader();
  final _log = <_LogEntry>[];

  String _serverUrl = 'http://192.168.1.20:8000';
  String _guardId = 'G1';
  bool _nfcAvailable = true;
  bool _scanning = false;
  int _pending = 0;

  ScanService get _service => ScanService(_serverUrl);

  @override
  void initState() {
    super.initState();
    _init();
  }

  Future<void> _init() async {
    final prefs = await SharedPreferences.getInstance();
    final available = await _nfc.isAvailable();
    setState(() {
      _serverUrl = prefs.getString('server_url') ?? _serverUrl;
      _guardId = prefs.getString('guard_id') ?? _guardId;
      _nfcAvailable = available;
    });
    await _refreshPending();
  }

  Future<void> _refreshPending() async {
    final n = await _service.pendingCount();
    if (mounted) setState(() => _pending = n);
  }

  Future<void> _scan() async {
    setState(() => _scanning = true);
    try {
      final read = await _nfc.readTag();
      final scan = Scan.create(
        tagId: read.tagId,
        guardId: _guardId,
        ndefUrl: read.ndefUrl,
        tagType: read.tagType,
      );
      final result = await _service.submit(scan);

      final _LogEntry entry;
      if (!result.delivered) {
        entry = _LogEntry(scan.scannedAt, scan.tagId, 'Saved offline - will sync',
            Icons.cloud_off, Colors.orange);
      } else if (result.knownTag) {
        entry = _LogEntry(scan.scannedAt, scan.tagId, result.checkpoint!,
            Icons.check_circle, Colors.green);
      } else {
        entry = _LogEntry(scan.scannedAt, scan.tagId, 'Unregistered tag',
            Icons.help, Colors.amber);
      }
      setState(() => _log.insert(0, entry));
    } catch (e) {
      _toast('Scan failed: $e');
    } finally {
      if (mounted) setState(() => _scanning = false);
      await _refreshPending();
    }
  }

  Future<void> _sync() async {
    final sent = await _service.flushQueue();
    await _refreshPending();
    _toast(_pending == 0 ? 'Synced $sent scan(s)' : 'Sent $sent, $_pending still waiting');
  }

  void _toast(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  Future<void> _openSettings() async {
    final urlCtrl = TextEditingController(text: _serverUrl);
    final guardCtrl = TextEditingController(text: _guardId);
    final saved = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Settings'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: urlCtrl,
              decoration: const InputDecoration(labelText: 'Server URL'),
              keyboardType: TextInputType.url,
              autocorrect: false,
            ),
            TextField(
              controller: guardCtrl,
              decoration: const InputDecoration(labelText: 'Guard ID'),
              autocorrect: false,
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
          FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('Save')),
        ],
      ),
    );
    if (saved != true) return;

    final url = urlCtrl.text.trim().replaceAll(RegExp(r'/+$'), '');
    final guard = guardCtrl.text.trim();
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString('server_url', url);
    await prefs.setString('guard_id', guard);
    setState(() {
      _serverUrl = url;
      _guardId = guard;
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('SentryWalk'),
        actions: [
          if (_pending > 0)
            TextButton.icon(
              onPressed: _sync,
              icon: const Icon(Icons.sync),
              label: Text('$_pending pending'),
            ),
          IconButton(onPressed: _openSettings, icon: const Icon(Icons.settings)),
        ],
      ),
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                children: [
                  Text('Guard $_guardId', style: theme.textTheme.titleMedium),
                  const SizedBox(height: 16),
                  SizedBox(
                    width: double.infinity,
                    height: 72,
                    child: FilledButton.icon(
                      onPressed: (_nfcAvailable && !_scanning) ? _scan : null,
                      icon: _scanning
                          ? const SizedBox(
                              width: 24, height: 24,
                              child: CircularProgressIndicator(strokeWidth: 3))
                          : const Icon(Icons.nfc, size: 32),
                      label: Text(
                        _scanning ? 'Hold near tag...' : 'Scan checkpoint',
                        style: const TextStyle(fontSize: 20),
                      ),
                    ),
                  ),
                  if (!_nfcAvailable)
                    Padding(
                      padding: const EdgeInsets.only(top: 12),
                      child: Text(
                        'NFC is off or not supported on this phone.',
                        style: TextStyle(color: theme.colorScheme.error),
                      ),
                    ),
                ],
              ),
            ),
            const Divider(height: 1),
            Expanded(
              child: _log.isEmpty
                  ? const Center(child: Text('No scans yet this session'))
                  : ListView.separated(
                      itemCount: _log.length,
                      separatorBuilder: (_, __) => const Divider(height: 1),
                      itemBuilder: (_, i) {
                        final e = _log[i];
                        final t = e.time.toLocal();
                        final hh = t.hour.toString().padLeft(2, '0');
                        final mm = t.minute.toString().padLeft(2, '0');
                        return ListTile(
                          leading: Icon(e.icon, color: e.color),
                          title: Text(e.label),
                          subtitle: Text(e.tagId),
                          trailing: Text('$hh:$mm'),
                        );
                      },
                    ),
            ),
          ],
        ),
      ),
    );
  }
}
