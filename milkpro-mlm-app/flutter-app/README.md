# Flutter App - MilkPro MLM

This directory contains the Flutter mobile app for MilkPro MLM.

## Features

- Firebase OTP authentication
- User profile with KYC upload and verification
- Investment management
- Milk transactions (buy/sell)
- Referral system with MLM tree and commissions
- Support ticket system with chat-style replies
- PDF export of profile, receipts, and investment summaries
- Dark mode and language toggle (Urdu/English)

## Setup

1. Install Flutter SDK: https://flutter.dev/docs/get-started/install
2. Configure Firebase project and add `google-services.json` (Android) and `GoogleService-Info.plist` (iOS)
3. Run `flutter pub get` to install dependencies
4. Run the app on an emulator or device

## Firebase Integration

- Firebase Authentication (Phone OTP)
- Firebase Storage for KYC documents and images
- Firebase Cloud Messaging for push notifications
