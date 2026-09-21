class TransactionGuard {
  static String? validateSend({required String selectedNetworkId, required int selectedChainId, required String activeNetworkId, required int activeChainId, required String sender, required String recipient, required int amount, required int fee}) {
    if (selectedNetworkId != activeNetworkId || selectedChainId != activeChainId) return 'Network mismatch';
    if (sender.trim().isEmpty || recipient.trim().isEmpty) return 'Sender and recipient are required';
    if (sender == recipient) return 'Sender and recipient must differ';
    if (amount <= 0) return 'Amount must be greater than zero';
    if (fee < 0) return 'Fee cannot be negative';
    return null;
  }
}
