import 'package:flutter_test/flutter_test.dart';
import 'package:indochainwallet/services/transaction_guard.dart';

void main() {
  test('rejects network mismatch', () {
    expect(
      TransactionGuard.validateSend(
        selectedNetworkId: 'ind-testnet-1',
        selectedChainId: 777101,
        activeNetworkId: 'ind-mainnet-1',
        activeChainId: 777001,
        sender: 'a',
        recipient: 'b',
        amount: 1,
        fee: 1,
      ),
      'Network mismatch',
    );
  });

  test('accepts valid local review input', () {
    expect(
      TransactionGuard.validateSend(
        selectedNetworkId: 'ind-testnet-1',
        selectedChainId: 777101,
        activeNetworkId: 'ind-testnet-1',
        activeChainId: 777101,
        sender: 'A',
        recipient: 'B',
        amount: 10,
        fee: 1,
      ),
      isNull,
    );
  });
}
