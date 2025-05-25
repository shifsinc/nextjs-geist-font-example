# Backend API - MilkPro MLM

This directory contains the Go backend API for MilkPro MLM.

## Features

- RESTful API endpoints for user management, KYC, investments, transactions, referrals, support tickets, notifications, and payments
- Database integration with PostgreSQL or SQLite
- Authentication and authorization middleware
- Admin panel backend support

## Setup

1. Install Go: https://golang.org/doc/install
2. Setup PostgreSQL or SQLite database
3. Configure environment variables for database connection and Firebase credentials
4. Run `go mod tidy` to install dependencies
5. Run the server with `go run main.go`

## Database Schema

- users
- kyc_documents
- products
- transactions
- projects
- investments
- referrals
- support_tickets
- ticket_messages
- notifications
