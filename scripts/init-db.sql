-- Create videos table
CREATE TABLE IF NOT EXISTS videos (
  id VARCHAR(36) PRIMARY KEY,
  title VARCHAR(255) NOT NULL,
  description TEXT,
  duration INT NOT NULL,
  size_bytes BIGINT NOT NULL,
  upload_status ENUM('PENDING', 'UPLOADING', 'COMPLETED', 'FAILED') DEFAULT 'PENDING',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_created_at_id (created_at DESC, id DESC),
  INDEX idx_status (upload_status)
);

-- Create uploads table (track upload sessions)
CREATE TABLE IF NOT EXISTS uploads (
  id VARCHAR(36) PRIMARY KEY,
  video_id VARCHAR(36) NOT NULL,
  user_id VARCHAR(36) NOT NULL,
  s3_upload_id VARCHAR(255) NOT NULL DEFAULT '',
  total_size BIGINT NOT NULL DEFAULT 0,
  uploaded_size BIGINT NOT NULL DEFAULT 0,
  uploaded_chunks INT NOT NULL DEFAULT 0,
  total_chunks INT NOT NULL DEFAULT 0,
  status ENUM('INITIATED', 'IN_PROGRESS', 'COMPLETED', 'FAILED') DEFAULT 'INITIATED',
  percentage DOUBLE NOT NULL DEFAULT 0,
  speed_mbps DOUBLE DEFAULT NULL,
  estimated_seconds BIGINT DEFAULT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  completed_at TIMESTAMP NULL,
  FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE,
  INDEX idx_user_id (user_id),
  INDEX idx_status (status)
);

-- Create download sessions table
CREATE TABLE IF NOT EXISTS downloads (
  id VARCHAR(36) PRIMARY KEY,
  video_id VARCHAR(36) NOT NULL,
  user_id VARCHAR(36) NOT NULL,
  started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP NULL,
  status ENUM('INITIATED', 'IN_PROGRESS', 'COMPLETED', 'FAILED') DEFAULT 'INITIATED',
  FOREIGN KEY (video_id) REFERENCES videos(id) ON DELETE CASCADE,
  INDEX idx_user_id (user_id),
  INDEX idx_video_id (video_id)
);

-- Create users table (auth: email + bcrypt password hash)
CREATE TABLE IF NOT EXISTS users (
  id VARCHAR(36) PRIMARY KEY,
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_email (email)
);

-- Subscription plans (seeded below)
CREATE TABLE IF NOT EXISTS plans (
  id VARCHAR(36) PRIMARY KEY,
  name VARCHAR(50) UNIQUE NOT NULL,
  amount_minor INT NOT NULL,           -- price in the smallest currency unit (e.g. paise)
  currency CHAR(3) NOT NULL DEFAULT 'INR',
  period_days INT NOT NULL,            -- length of the access window granted on payment
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User subscriptions (one active window per user+plan)
CREATE TABLE IF NOT EXISTS subscriptions (
  id VARCHAR(36) PRIMARY KEY,
  user_id VARCHAR(36) NOT NULL,
  plan_id VARCHAR(36) NOT NULL,
  status ENUM('PENDING_PAYMENT','ACTIVE','EXPIRED','CANCELLED') NOT NULL DEFAULT 'PENDING_PAYMENT',
  current_period_end TIMESTAMP NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (plan_id) REFERENCES plans(id),
  INDEX idx_user_id (user_id),
  INDEX idx_status (status)
);

-- Payments: opaque provider references only, never card data (PCI)
CREATE TABLE IF NOT EXISTS payments (
  id VARCHAR(36) PRIMARY KEY,
  user_id VARCHAR(36) NOT NULL,
  subscription_id VARCHAR(36) NOT NULL,
  amount_minor INT NOT NULL,
  currency CHAR(3) NOT NULL DEFAULT 'INR',
  status ENUM('CREATED','PENDING','CAPTURED','FAILED','REFUNDED') NOT NULL DEFAULT 'CREATED',
  provider VARCHAR(50) NOT NULL,
  provider_order_id VARCHAR(255),       -- e.g. Razorpay payment-link id
  provider_payment_id VARCHAR(255),     -- e.g. Razorpay payment id
  payment_url VARCHAR(512),             -- hosted checkout link (returned on idempotent re-request)
  idempotency_key VARCHAR(255) NOT NULL UNIQUE,  -- our reference_id, makes order creation idempotent
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE CASCADE,
  INDEX idx_user_id (user_id),
  INDEX idx_subscription_id (subscription_id),
  INDEX idx_status (status)
);

-- Webhook dedupe ledger: one payment emits many events, so dedupe lives here
CREATE TABLE IF NOT EXISTS webhook_events (
  provider_event_id VARCHAR(255) PRIMARY KEY,  -- x-razorpay-event-id (unique per event)
  received_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Seed default plans (idempotent)
INSERT INTO plans (id, name, amount_minor, currency, period_days)
VALUES
  ('00000000-0000-0000-0000-000000000001', 'free',    0,     'INR', 36500),
  ('00000000-0000-0000-0000-000000000002', 'premium', 29900, 'INR', 30)
ON DUPLICATE KEY UPDATE name = name;
