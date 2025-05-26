-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    phone VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100),
    email VARCHAR(100),
    profile_image_url TEXT,
    kyc_status VARCHAR(20) DEFAULT 'pending',
    is_admin BOOLEAN DEFAULT FALSE,
    balance DECIMAL(15,2) DEFAULT 0.0,
    referral_code VARCHAR(10) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Products table
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL, -- milk, dairy, feed
    price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Projects table (for investments)
CREATE TABLE projects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    lock_days INTEGER NOT NULL,
    profit_percent DECIMAL(5,2) NOT NULL,
    min_investment DECIMAL(10,2) NOT NULL,
    max_investment DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Investments table
CREATE TABLE investments (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    project_id INTEGER REFERENCES projects(id),
    amount DECIMAL(15,2) NOT NULL,
    lock_end_date TIMESTAMP NOT NULL,
    profit_percent DECIMAL(5,2) NOT NULL,
    reinvest BOOLEAN DEFAULT FALSE,
    status VARCHAR(20) DEFAULT 'active',
    invested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Transactions table
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    product_id INTEGER REFERENCES products(id),
    type VARCHAR(20) NOT NULL, -- buy or sell
    quantity DECIMAL(10,2) NOT NULL,
    unit VARCHAR(10) NOT NULL, -- kg or litre
    price DECIMAL(10,2) NOT NULL,
    transaction_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- KYC Documents table
CREATE TABLE kyc_documents (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    document_url TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Referrals table
CREATE TABLE referrals (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    referred_user_id INTEGER REFERENCES users(id),
    level INTEGER NOT NULL CHECK (level BETWEEN 1 AND 3),
    commission DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, referred_user_id)
);

-- Support staff table
CREATE TABLE support_staff (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'support', -- 'support' or 'admin'
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Support tickets table
CREATE TABLE support_tickets (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    subject VARCHAR(200) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'open', -- 'open', 'in_progress', 'resolved', 'closed'
    priority VARCHAR(20) NOT NULL DEFAULT 'medium', -- 'low', 'medium', 'high', 'urgent'
    assigned_to INTEGER REFERENCES support_staff(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Support ticket messages table
CREATE TABLE ticket_messages (
    id SERIAL PRIMARY KEY,
    ticket_id INTEGER REFERENCES support_tickets(id),
    sender_type VARCHAR(20) NOT NULL, -- 'user', 'staff'
    sender_id INTEGER NOT NULL, -- references users.id or support_staff.id
    message TEXT NOT NULL,
    attachment_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Live chat sessions table
CREATE TABLE chat_sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    staff_id INTEGER REFERENCES support_staff(id),
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- 'active', 'ended'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP
);

-- Live chat messages table
CREATE TABLE chat_messages (
    id SERIAL PRIMARY KEY,
    session_id INTEGER REFERENCES chat_sessions(id),
    sender_type VARCHAR(20) NOT NULL, -- 'user', 'staff'
    sender_id INTEGER NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for support system
CREATE INDEX idx_support_tickets_user_id ON support_tickets(user_id);
CREATE INDEX idx_support_tickets_assigned_to ON support_tickets(assigned_to);
CREATE INDEX idx_ticket_messages_ticket_id ON ticket_messages(ticket_id);
CREATE INDEX idx_chat_sessions_user_id ON chat_sessions(user_id);
CREATE INDEX idx_chat_sessions_staff_id ON chat_sessions(staff_id);
CREATE INDEX idx_chat_messages_session_id ON chat_messages(session_id);

-- Create indexes for existing tables
CREATE INDEX idx_users_phone ON users(phone);
CREATE INDEX idx_users_referral_code ON users(referral_code);
CREATE INDEX idx_investments_user_id ON investments(user_id);
CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_kyc_documents_user_id ON kyc_documents(user_id);
CREATE INDEX idx_referrals_user_id ON referrals(user_id);
CREATE INDEX idx_referrals_referred_user_id ON referrals(referred_user_id);

-- Insert default admin user
INSERT INTO users (phone, name, email, is_admin, kyc_status)
VALUES ('+1234567890', 'Admin User', 'admin@milkpro.com', TRUE, 'approved');

-- Insert sample products
INSERT INTO products (name, type, price) VALUES
('Fresh Milk', 'milk', 2.50),
('Yogurt', 'dairy', 3.00),
('Cattle Feed', 'feed', 15.00);

-- Insert sample investment project
INSERT INTO projects (name, description, lock_days, profit_percent, min_investment, max_investment)
VALUES (
    'Dairy Farm Expansion',
    'Investment opportunity in expanding our dairy farm operations',
    90,
    15.00,
    1000.00,
    50000.00
);
