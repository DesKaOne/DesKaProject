import 'package:flutter_test/flutter_test.dart';
import 'package:indochainwallet/services/wallet_lifecycle.dart';

void main() {
  test('generates and validates a mnemonic', () {
    final lifecycle = WalletLifecycle();
    final mnemonic = lifecycle.generateMnemonic();
    expect(mnemonic.split(' '), hasLength(12));
    expect(lifecycle.validateMnemonic(mnemonic), isTrue);
    expect(lifecycle.validateMnemonic('invalid words only'), isFalse);
  });
}
