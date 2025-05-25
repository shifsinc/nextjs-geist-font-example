import 'package:flutter/material.dart';
import 'package:cloud_firestore/cloud_firestore.dart';
import 'package:firebase_auth/firebase_auth.dart';
import '../models/user.dart';

class UserProvider extends ChangeNotifier {
  MlmUser? _user;
  final FirebaseAuth _auth = FirebaseAuth.instance;
  final FirebaseFirestore _firestore = FirebaseFirestore.instance;
  bool _loading = false;

  MlmUser? get user => _user;
  bool get loading => _loading;

  Future<void> loadUser() async {
    if (_auth.currentUser == null) return;

    try {
      _loading = true;
      notifyListeners();

      final doc = await _firestore
          .collection('users')
          .doc(_auth.currentUser!.uid)
          .get();

      if (doc.exists) {
        _user = MlmUser.fromJson({
          'id': doc.id,
          ...doc.data()!,
        });
      } else {
        // Create new user document if it doesn't exist
        final newUser = MlmUser(
          id: _auth.currentUser!.uid,
          phone: _auth.currentUser!.phoneNumber!,
          referralCode: _generateReferralCode(),
        );

        await _firestore
            .collection('users')
            .doc(_auth.currentUser!.uid)
            .set(newUser.toJson());

        _user = newUser;
      }
    } catch (e) {
      debugPrint('Error loading user: $e');
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> updateProfile({String? name, String? email}) async {
    if (_user == null) return;

    try {
      _loading = true;
      notifyListeners();

      final updatedUser = _user!.copyWith(
        name: name,
        email: email,
      );

      await _firestore
          .collection('users')
          .doc(_user!.id)
          .update(updatedUser.toJson());

      _user = updatedUser;
    } catch (e) {
      debugPrint('Error updating profile: $e');
      rethrow;
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> submitKYC({
    required String fullName,
    required String idNumber,
  }) async {
    if (_user == null) return;

    try {
      _loading = true;
      notifyListeners();

      // Create KYC document
      await _firestore.collection('kyc_submissions').add({
        'user_id': _user!.id,
        'full_name': fullName,
        'id_number': idNumber,
        'status': 'pending',
        'submitted_at': FieldValue.serverTimestamp(),
      });

      // Update user's KYC status
      final updatedUser = _user!.copyWith(
        name: fullName,
        kycStatus: 'pending',
      );

      await _firestore
          .collection('users')
          .doc(_user!.id)
          .update(updatedUser.toJson());

      _user = updatedUser;
    } catch (e) {
      debugPrint('Error submitting KYC: $e');
      rethrow;
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  Future<void> applyReferralCode(String referralCode) async {
    if (_user == null || _user!.referredBy != null) return;

    try {
      _loading = true;
      notifyListeners();

      // Find user with this referral code
      final querySnapshot = await _firestore
          .collection('users')
          .where('referral_code', isEqualTo: referralCode)
          .limit(1)
          .get();

      if (querySnapshot.docs.isEmpty) {
        throw Exception('Invalid referral code');
      }

      final referrer = querySnapshot.docs.first;

      // Update current user
      final updatedUser = _user!.copyWith(
        referredBy: referrer.id,
      );

      await _firestore
          .collection('users')
          .doc(_user!.id)
          .update(updatedUser.toJson());

      // Update referrer's referrals list
      await _firestore.collection('users').doc(referrer.id).update({
        'referrals': FieldValue.arrayUnion([_user!.id]),
      });

      _user = updatedUser;
    } catch (e) {
      debugPrint('Error applying referral code: $e');
      rethrow;
    } finally {
      _loading = false;
      notifyListeners();
    }
  }

  String _generateReferralCode() {
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    final random = DateTime.now().millisecondsSinceEpoch.toString();
    final code = List.generate(6, (index) {
      final randomIndex = (random.hashCode + index) % chars.length;
      return chars[randomIndex];
    }).join();
    return code;
  }

  void signOut() {
    _auth.signOut();
    _user = null;
    notifyListeners();
  }
}
