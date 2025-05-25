class Product {
  final int id;
  final String name;
  final String type;
  final double price;

  Product({
    required this.id,
    required this.name,
    required this.type,
    required this.price,
  });

  factory Product.fromJson(Map<String, dynamic> json) {
    return Product(
      id: json['id'] as int,
      name: json['name'] as String,
      type: json['type'] as String,
      price: (json['price'] as num).toDouble(),
    );
  }

  Map<String, dynamic> toJson() => {
    'id': id,
    'name': name,
    'type': type,
    'price': price,
  };
}
