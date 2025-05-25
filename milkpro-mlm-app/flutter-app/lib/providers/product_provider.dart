import 'package:flutter/material.dart';
import 'package:cloud_firestore/cloud_firestore.dart';
import '../models/product.dart';

class ProductProvider extends ChangeNotifier {
  final FirebaseFirestore _firestore = FirebaseFirestore.instance;
  List<Product> _products = [];
  bool _loading = false;
  String? _error;

  List<Product> get products => _products;
  bool get loading => _loading;
  String? get error => _error;

  Future<void> loadProducts() async {
    try {
      _loading = true;
      _error = null;
      notifyListeners();

      final snapshot = await _firestore.collection('products').get();
      _products = snapshot.docs.map((doc) {
        return Product.fromJson({
          'id': int.parse(doc.id),
          ...doc.data(),
        });
      }).toList();

      // Sort products by type
      _products.sort((a, b) => a.type.compareTo(b.type));
    } catch (e) {
      _error = 'Failed to load products: $e';
      debugPrint(_error);
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  List<Product> getProductsByType(String type) {
    return _products.where((product) => product.type == type).toList();
  }

  Future<void> purchaseProduct(Product product, int quantity, String userId) async {
    try {
      _loading = true;
      notifyListeners();

      // Create transaction record
      await _firestore.collection('transactions').add({
        'user_id': userId,
        'product_id': product.id,
        'quantity': quantity,
        'total_amount': product.price * quantity,
        'status': 'pending',
        'created_at': FieldValue.serverTimestamp(),
      });

      // Update user's transaction history
      await _firestore.collection('users').doc(userId).update({
        'transactions': FieldValue.arrayUnion([
          {
            'description': '${product.name} x $quantity',
            'amount': -(product.price * quantity),
            'date': DateTime.now().toIso8601String(),
            'type': 'purchase',
          }
        ])
      });

    } catch (e) {
      _error = 'Failed to purchase product: $e';
      debugPrint(_error);
      rethrow;
    } finally {
      _loading = false;
      notifyListeners();
    }
  }
}
