import 'package:bip39/bip39.dart' as bip39;

/// Wallet lifecycle primitives that are safe to use before an official
/// IndoChain BIP39 derivation path is frozen.
class WalletLifecycle {
  String generateMnemonic() => bip39.generateMnemonic(strength: 128);

  bool validateMnemonic(String mnemonic) =>
      bip39.validateMnemonic(mnemonic.trim());

  String mnemonicToSeedHex(String mnemonic) {
    if (!validateMnemonic(mnemonic)) {
      throw ArgumentError('invalid mnemonic');
    }
    return bip39.mnemonicToSeedHex(mnemonic.trim());
  }

  /// Returns a normalized, user-supplied private key for local signing.
  ///
  /// This is intentionally separate from the BIP39 flow: the repository does
  /// not currently define an IndoChain mnemonic derivation path, so no wallet
  /// implementation may invent one.
  String normalizePrivateKey(String privateKeyHex) {
    final value = privateKeyHex.trim().toLowerCase();
    final normalized = value.startsWith('0x') ? value.substring(2) : value;
    if (!RegExp(r'^[0-9a-f]{64}$').hasMatch(normalized)) {
      throw ArgumentError('private key must be exactly 32-byte hex');
    }
    return normalized;
  }

  /// Signals that mnemonic -> private-key derivation cannot be performed yet.
  Never derivePrivateKeyFromMnemonic(String mnemonic) {
    if (!validateMnemonic(mnemonic)) {
      throw ArgumentError('invalid mnemonic');
    }
    throw StateError(
      'IndoChain mnemonic derivation path is not defined yet',
    );
  }
}
