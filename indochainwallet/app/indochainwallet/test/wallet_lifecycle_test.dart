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

  test('converts a valid mnemonic into seed material', () {
    final lifecycle = WalletLifecycle();
    const mnemonic =
        'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about';
    final seed = lifecycle.mnemonicToSeedHex(mnemonic);
    expect(seed.length, 128);
    expect(seed, matches(RegExp(r'^[0-9a-f]+$')));
  });

  test('normalizes a valid private key', () {
    final lifecycle = WalletLifecycle();
    const key =
        '0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef';
    expect(
      lifecycle.normalizePrivateKey(key),
      '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef',
    );
  });

  test('rejects malformed private keys', () {
    final lifecycle = WalletLifecycle();
    expect(
      () => lifecycle.normalizePrivateKey('abcd'),
      throwsArgumentError,
    );
  });

  test('mnemonic derivation stays fail-closed until IndoChain path is defined',
      () {
    final lifecycle = WalletLifecycle();
    const mnemonic =
        'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about';
    expect(
      () => lifecycle.derivePrivateKeyFromMnemonic(mnemonic),
      throwsStateError,
    );
  });
}
