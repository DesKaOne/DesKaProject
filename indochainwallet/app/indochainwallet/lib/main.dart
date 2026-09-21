import 'package:flutter/material.dart';
import 'api/indochain_api.dart';
import 'models/wallet_models.dart';
import 'services/transaction_guard.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const IndoChainWalletApp());
}

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
  final recipient = TextEditingController();
  final amount = TextEditingController();
  NetworkInfo? networkInfo;
  List<WalletTransaction> history = const [];
  String balance = '—';
  String nonce = '—';
  String fee = '—';
  String status = 'Ready';
  bool busy = false;

  Future<void> refreshWallet() async {
    final addr = address.text.trim();
    if (addr.isEmpty) { setState(() => status = 'Enter an IndoChain address'); return; }
    setState(() => busy = true);
    try {
      final api = IndoChainApi(backend.text.trim());
      final n = await api.network();
      final b = await api.balance(addr);
      final a = await api.addressInfo(addr);
      final h = await api.history(addr);
      final bd = (b['data'] ?? b) as Map<String, dynamic>;
      final ad = (a['data'] ?? a) as Map<String, dynamic>;
      if (!mounted) return;
      setState(() {
        networkInfo = n;
        balance = '${bd['spendable_balance'] ?? bd['balance'] ?? '—'} ${bd['ticker'] ?? 'dIDR'}';
        nonce = '${ad['confirmed_nonce'] ?? '—'}';
        history = h;
        status = 'Connected';
      });
    } catch (e) {
      if (mounted) setState(() => status = 'Backend unavailable: $e');
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  Future<void> reviewTransaction() async {
    final n = networkInfo;
    final sender = address.text.trim();
    final target = recipient.text.trim();
    final units = int.tryParse(amount.text.trim()) ?? 0;
    if (n == null) { setState(() => status = 'Connect to IndoChain first'); return; }
    final error = TransactionGuard.validateSend(
      selectedNetworkId: n.networkId,
      selectedChainId: n.chainId,
      activeNetworkId: n.networkId,
      activeChainId: n.chainId,
      sender: sender, recipient: target, amount: units, fee: 0,
    );
    if (error != null) { setState(() => status = error); return; }
    try {
      final policy = await IndoChainApi(backend.text.trim()).feePolicy();
      final pd = (policy['data'] ?? policy) as Map<String, dynamic>;
      final minFee = pd['min_fee'] ?? 0;
      if (!mounted) return;
      await showDialog<void>(
        context: context,
        builder: (_) => AlertDialog(
          title: const Text('Transaction confirmation'),
          content: Text('Network: ${n.networkId}\nChain ID: ${n.chainId}\nFrom: $sender\nTo: $target\nAmount: $units base units\nMinimum fee: $minFee\n\nSigning must occur locally before relay.'),
          actions: [TextButton(onPressed: () => Navigator.pop(context), child: const Text('Close'))],
        ),
      );
      if (mounted) setState(() => fee = 'Minimum fee: $minFee');
    } catch (e) {
      if (mounted) setState(() => status = 'Fee policy unavailable: $e');
    }
  }

  @override
  void dispose() { backend.dispose(); address.dispose(); recipient.dispose(); amount.dispose(); super.dispose(); }

  @override
  Widget build(BuildContext context) {
    final n = networkInfo;
    return Scaffold(
      appBar: AppBar(title: const Text('IndoChainWallet')),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          Text('Network', style: Theme.of(context).textTheme.titleMedium),
          Text(n == null ? 'Not connected' : '${n.networkId} / chain ${n.chainId}'),
          const SizedBox(height: 8),
          Text('Status: $status'),
          TextField(controller: backend, decoration: const InputDecoration(labelText: 'Wallet backend URL')),
          TextField(controller: address, decoration: const InputDecoration(labelText: 'IndoChain address')),
          FilledButton.icon(onPressed: busy ? null : refreshWallet, icon: const Icon(Icons.sync), label: const Text('Refresh wallet')),
          Card(child: ListTile(title: const Text('Spendable balance'), subtitle: Text(balance))),
          Card(child: ListTile(title: const Text('Confirmed nonce'), subtitle: Text(nonce))),
          Card(child: ListTile(title: const Text('Fee'), subtitle: Text(fee))),
          const SizedBox(height: 12),
          Text('Send review', style: Theme.of(context).textTheme.titleMedium),
          TextField(controller: recipient, decoration: const InputDecoration(labelText: 'Recipient')),
          TextField(controller: amount, keyboardType: TextInputType.number, decoration: const InputDecoration(labelText: 'Amount (base units)')),
          FilledButton.icon(onPressed: busy ? null : reviewTransaction, icon: const Icon(Icons.verified_user), label: const Text('Review transaction')),
          const SizedBox(height: 12),
          Text('History', style: Theme.of(context).textTheme.titleMedium),
          if (history.isEmpty) const Text('No transaction history loaded.'),
          for (final tx in history) ListTile(
            title: Text(tx.id.isEmpty ? '(unknown tx)' : tx.id),
            subtitle: Text('${tx.status} · nonce ${tx.nonce}\n${tx.from} → ${tx.to}'),
            isThreeLine: true,
          ),
        ],
      ),
    );
  }
}