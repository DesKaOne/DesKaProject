import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/wallet_models.dart';

class IndoChainApi {
  final String baseUrl;
  final http.Client client;
  IndoChainApi(this.baseUrl, {http.Client? client}) : client = client ?? http.Client();

  Uri _uri(String path) => Uri.parse('${baseUrl.replaceFirst(RegExp(r'/$', multiLine: true), '')}$path');

  Future<Map<String, dynamic>> _get(String path) async {
    final response = await client.get(_uri(path));
    final data = jsonDecode(response.body) as Map<String, dynamic>;
    if (response.statusCode >= 400) throw Exception(data['error'] ?? response.body);
    return data;
  }

  Future<Map<String, dynamic>> _post(String path, Map<String, dynamic> body) async {
    final response = await client.post(_uri(path), headers: {'Content-Type': 'application/json'}, body: jsonEncode(body));
    final data = jsonDecode(response.body) as Map<String, dynamic>;
    if (response.statusCode >= 400) throw Exception(data['error'] ?? response.body);
    return data;
  }

  Future<NetworkInfo> network() async => NetworkInfo.fromJson(await _get('/v1/network'));

  Future<Map<String, dynamic>> balance(String address) async => _get('/v1/wallet/balance?address=${Uri.encodeQueryComponent(address)}');

  Future<List<WalletTransaction>> history(String address) async {
    final data = await _get('/v1/wallet/history?address=${Uri.encodeQueryComponent(address)}');
    final raw = (data['transactions'] ?? data['items'] ?? const []) as List<dynamic>;
    return raw.whereType<Map<String, dynamic>>().map(WalletTransaction.fromJson).toList();
  }

  Future<Map<String, dynamic>> send(Map<String, dynamic> signedTx) => _post('/v1/tx/send', signedTx);
  Future<Map<String, dynamic>> transaction(String id) => _get('/v1/tx/${Uri.encodeComponent(id)}');
  String explorerTxUrl(String base, String id) => '${base.replaceFirst(RegExp(r'/$', multiLine: true), '')}/explorer/tx/$id';
  String explorerAddressUrl(String base, String address) => '${base.replaceFirst(RegExp(r'/$', multiLine: true), '')}/explorer/address/$address';
}
