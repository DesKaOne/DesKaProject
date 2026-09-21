import 'package:flutter_test/flutter_test.dart';
import 'package:indochainwallet/models/wallet_models.dart';

void main() {
  test('parses network identity without guessing defaults', () {
    final n = NetworkInfo.fromJson({
      'network': 'testnet',
      'network_id': 'ind-testnet-1',
      'chain_id': 777101,
    });
    expect(n.name, 'testnet');
    expect(n.networkId, 'ind-testnet-1');
    expect(n.chainId, 777101);
  });

  test('parses transaction status and nonce', () {
    final tx = WalletTransaction.fromJson({
      'id': 'abc',
      'from': 'A',
      'to': 'B',
      'amount': 10,
      'fee': 1,
      'nonce': 4,
      'status': 'pending',
    });
    expect(tx.id, 'abc');
    expect(tx.status, 'pending');
    expect(tx.nonce, 4);
  });
}
