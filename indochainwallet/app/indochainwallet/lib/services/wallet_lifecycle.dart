import 'package:bip39/bip39.dart' as bip39;

class WalletLifecycle {
  String generateMnemonic() => bip39.generateMnemonic(strength: 128);

  bool validateMnemonic(String mnemonic) => bip39.validateMnemonic(mnemonic.trim());

  String mnemonicToSeedHex(String mnemonic) {
    if (!validateMnemonic(mnemonic)) {
      throw ArgumentError('invalid mnemonic');
    }
    return bip39.mnemonicToSeedHex(mnemonic.trim());
  }

  // IndoChain must define its derivation path before a BIP39 seed can become
  // an IndoChain secp256k1 signing key. This method intentionally does not
  // invent a derivation rule.
  Never derivePrivateKeyFromMnemonic(String mnemonic) {
    throw StateError('IndoChain mnemonic derivation path is not defined yet');
  }
}
