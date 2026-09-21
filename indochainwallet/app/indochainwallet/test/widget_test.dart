import 'package:flutter_test/flutter_test.dart';
import 'package:indochainwallet/main.dart';

void main() {
  testWidgets('renders wallet network controls', (tester) async {
    await tester.pumpWidget(const IndoChainWalletApp());
    expect(find.text('IndoChainWallet'), findsOneWidget);
    expect(find.text('Network'), findsOneWidget);
    expect(find.text('Wallet backend URL'), findsOneWidget);
    expect(find.text('IndoChain address'), findsOneWidget);
  });
}
