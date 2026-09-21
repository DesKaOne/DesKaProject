import 'dart:convert';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class SecureWalletStore {
  static const _storage = FlutterSecureStorage();
  static const _privateKey = 'indochain.private_key';
  static const _mnemonic = 'indochain.mnemonic';
  static const _address = 'indochain.address';
  static const _networkId = 'indochain.network_id';
  static const _chainId = 'indochain.chain_id';

  Future<void> save({
    required String privateKey,
    required String address,
    required String networkId,
    required int chainId,
    String? mnemonic,
  }) async {
    if (privateKey.trim().isEmpty || address.trim().isEmpty) {
      throw ArgumentError('wallet credentials are required');
    }
    await _storage.write(key: _privateKey, value: privateKey.trim());
    await _storage.write(key: _address, value: address.trim());
    await _storage.write(key: _networkId, value: networkId.trim());
    await _storage.write(key: _chainId, value: '$chainId');
    if (mnemonic != null && mnemonic.trim().isNotEmpty) {
      await _storage.write(key: _mnemonic, value: mnemonic.trim());
    }
  }

  Future<Map<String, String>?> load() async {
    final privateKey = await _storage.read(key: _privateKey);
    final mnemonic = await _storage.read(key: _mnemonic);
    final address = await _storage.read(key: _address);
    final networkId = await _storage.read(key: _networkId);
    final chainId = await _storage.read(key: _chainId);
    if ([privateKey, address, networkId, chainId].any((v) => v == null || v.isEmpty)) {
      return null;
    }
    return {
      'private_key': privateKey!,
      if (mnemonic != null && mnemonic.isNotEmpty) 'mnemonic': mnemonic,
      'address': address!,
      'network_id': networkId!,
      'chain_id': chainId!,
    };
  }

  Future<void> clear() => _storage.deleteAll();

  String exportMetadata(Map<String, String> wallet) => jsonEncode({
        'address': wallet['address'],
        'network_id': wallet['network_id'],
        'chain_id': int.tryParse(wallet['chain_id'] ?? '0') ?? 0,
      });
}
