class NetworkInfo {
  final String name;
  final String networkId;
  final int chainId;
  final String? explorerBaseUrl;
  const NetworkInfo({required this.name, required this.networkId, required this.chainId, this.explorerBaseUrl});
  factory NetworkInfo.fromJson(Map<String, dynamic> json) => NetworkInfo(
    name: '${json['network'] ?? json['name'] ?? 'unknown'}',
    networkId: '${json['network_id'] ?? ''}',
    chainId: int.tryParse('${json['chain_id'] ?? 0}') ?? 0,
    explorerBaseUrl: json['explorer_base_url']?.toString(),
  );
}

class WalletTransaction {
  final String id;
  final String from;
  final String to;
  final int amount;
  final int fee;
  final int nonce;
  final String status;
  final String? blockHash;
  final int? blockHeight;
  const WalletTransaction({required this.id, required this.from, required this.to, required this.amount, required this.fee, required this.nonce, required this.status, this.blockHash, this.blockHeight});
  factory WalletTransaction.fromJson(Map<String, dynamic> json) => WalletTransaction(
    id: '${json['id'] ?? ''}',
    from: '${json['from'] ?? ''}',
    to: '${json['to'] ?? ''}',
    amount: int.tryParse('${json['amount'] ?? 0}') ?? 0,
    fee: int.tryParse('${json['fee'] ?? 0}') ?? 0,
    nonce: int.tryParse('${json['nonce'] ?? 0}') ?? 0,
    status: '${json['status'] ?? 'unknown'}',
    blockHash: json['block_hash']?.toString(),
    blockHeight: int.tryParse('${json['block_height'] ?? ''}'),
  );
}
