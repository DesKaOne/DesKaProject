import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

void main() => runApp(const IndoChainWalletApp());

class IndoChainWalletApp extends StatelessWidget {
  const IndoChainWalletApp({super.key});
  @override
  Widget build(BuildContext context) => MaterialApp(
    title: 'IndoChainWallet',
    theme: ThemeData(useMaterial3: true, colorSchemeSeed: Colors.indigo),
    home: const WalletHomePage(),
  );
}

class WalletHomePage extends StatefulWidget {
  const WalletHomePage({super.key});
  @override
  State<WalletHomePage> createState() => _WalletHomePageState();
}

class _WalletHomePageState extends State<WalletHomePage> {
  final backend = TextEditingController(text: 'http://127.0.0.1:8787');
  final address = TextEditingController();
  String network = 'Not connected';
  String balance = '—';
  bool busy = false;

  Future<Map<String, dynamic>> getJSON(String path) async {
    final response = await http.get(Uri.parse(backend.text.trim() + path));
    if (response.statusCode >= 400) throw Exception(response.body);
    return jsonDecode(response.body) as Map<String, dynamic>;
  }

  Future<void> refreshWallet() async {
    setState(() => busy = true);
    try {
      final n = await getJSON('/v1/network');
      var b = '—';
      final a = address.text.trim();
      if (a.isNotEmpty) {
        final data = await getJSON('/v1/wallet/balance?address=' + Uri.encodeQueryComponent(a));
        b = jsonEncode(data);
      }
      if (!mounted) return;
      setState(() {
        network = n['network_id'].toString() + ' / chain ' + n['chain_id'].toString();
        balance = b;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() { network = 'Backend unavailable'; balance = e.toString(); });
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('IndoChainWallet')),
    body: ListView(
      padding: const EdgeInsets.all(20),
      children: [
        Text('Network', style: Theme.of(context).textTheme.titleMedium),
        Text(network),
        const SizedBox(height: 16),
        TextField(controller: backend, decoration: const InputDecoration(labelText: 'Wallet backend URL')),
        TextField(controller: address, decoration: const InputDecoration(labelText: 'IndoChain address')),
        const SizedBox(height: 16),
        FilledButton.icon(onPressed: busy ? null : refreshWallet, icon: const Icon(Icons.sync), label: const Text('Refresh')),
        const SizedBox(height: 20),
        Text('Balance', style: Theme.of(context).textTheme.titleMedium),
        SelectableText(balance),
      ],
    ),
  );
}
