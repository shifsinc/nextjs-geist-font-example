class MlmUser {
  final String id;
  final String phone;
  final String? name;
  final String? email;
  final String? referralCode;
  final String? referredBy;
  final List<String> referrals;
  final double balance;
  final String kycStatus;
  final List<Map<String, dynamic>> transactions;

  MlmUser({
    required this.id,
    required this.phone,
    this.name,
    this.email,
    this.referralCode,
    this.referredBy,
    this.referrals = const [],
    this.balance = 0.0,
    this.kycStatus = 'pending',
    this.transactions = const [],
  });

  factory MlmUser.fromJson(Map<String, dynamic> json) {
    return MlmUser(
      id: json['id'] as String,
      phone: json['phone'] as String,
      name: json['name'] as String?,
      email: json['email'] as String?,
      referralCode: json['referral_code'] as String?,
      referredBy: json['referred_by'] as String?,
      referrals: List<String>.from(json['referrals'] ?? []),
      balance: (json['balance'] as num?)?.toDouble() ?? 0.0,
      kycStatus: json['kyc_status'] as String? ?? 'pending',
      transactions: List<Map<String, dynamic>>.from(json['transactions'] ?? []),
    );
  }

  Map<String, dynamic> toJson() => {
    'id': id,
    'phone': phone,
    'name': name,
    'email': email,
    'referral_code': referralCode,
    'referred_by': referredBy,
    'referrals': referrals,
    'balance': balance,
    'kyc_status': kycStatus,
    'transactions': transactions,
  };

  MlmUser copyWith({
    String? name,
    String? email,
    String? referralCode,
    String? referredBy,
    List<String>? referrals,
    double? balance,
    String? kycStatus,
    List<Map<String, dynamic>>? transactions,
  }) {
    return MlmUser(
      id: id,
      phone: phone,
      name: name ?? this.name,
      email: email ?? this.email,
      referralCode: referralCode ?? this.referralCode,
      referredBy: referredBy ?? this.referredBy,
      referrals: referrals ?? this.referrals,
      balance: balance ?? this.balance,
      kycStatus: kycStatus ?? this.kycStatus,
      transactions: transactions ?? this.transactions,
    );
  }
}
