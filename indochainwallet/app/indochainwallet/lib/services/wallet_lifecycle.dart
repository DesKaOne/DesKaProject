import 'package:bip39/bip39.dart' as bip39;
import 'package:secp256k1_ecdsa/secp256k1_ecdsa.dart';
import 'secure_wallet_store.dart';

class WalletLifecycle {
  final SecureWalletStore store;
  WalletLifecycle({SecureWalletStore? store}) : store = store ?? SecureWalletStore();

  String generateMnemonic() => bip39.generateMnemonic(strength: 128);

  bool validateMnemonic(String mnemonic) => bip39.validateMnemonic(mnemonic.trim());

  Future<Map<String, String>> restoreMetadata({
    required String mnemonic,
    required String address,
    required String networkId,
    required int chainId,
  }) async {
    if (!validateMnemonic(mnemonic)) {
      throw ArgumentError('invalid mnemonic');
    }
    final seedHex = bip39.mnemonicToSeedHex(mnemonic.trim());
    // BIP39 seed is not itself an IndoChain private key; derivation path must be
    // defined by the wallet protocol before converting it to a signing key.
    // Store the recovery material locally, but do not fabricate a derivation rule.
    final privateKeyPlaceholder = seedHex.substring(0, 64);
    await store.save(
      privateKey: privateKeyPlaceholder,
      mnemonic: mnemonic.trim(),
      address: address,
      networkId: networkId,
      chainId: chainId,
    );
    return {
      'address': address,
      'network_id': networkId,
      'chain_id': '$chainId',
    };
  }
}
