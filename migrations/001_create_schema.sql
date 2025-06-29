-- Create schema_migrations table to track migration versions
CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP DEFAULT NOW()
);

-- Create accounts table
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    balance NUMERIC(38,10) NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT positive_balance CHECK (balance >= 0)
);

-- Create transactions table
CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_account_id UUID,
    destination_account_id UUID NOT NULL,
    amount NUMERIC(38,10) NOT NULL,
    reference VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    
    CONSTRAINT positive_amount CHECK (amount > 0),
    CONSTRAINT valid_status CHECK (status IN ('pending', 'completed', 'failed')),
    CONSTRAINT different_accounts CHECK (source_account_id != destination_account_id OR source_account_id IS NULL),
    
    FOREIGN KEY (source_account_id) REFERENCES accounts(id),
    FOREIGN KEY (destination_account_id) REFERENCES accounts(id)
);

-- Create idempotency_keys table for request deduplication
CREATE TABLE idempotency_keys (
    key_hash VARCHAR(64) PRIMARY KEY,
    request_body TEXT NOT NULL,
    response_body TEXT,
    response_status INTEGER,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP DEFAULT NOW() + INTERVAL '24 hours'
);

-- Create indexes for performance
CREATE INDEX idx_accounts_balance ON accounts(balance);
CREATE INDEX idx_transactions_source ON transactions(source_account_id);
CREATE INDEX idx_transactions_destination ON transactions(destination_account_id);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);
CREATE INDEX idx_transactions_reference ON transactions(reference);
CREATE INDEX idx_idempotency_expires ON idempotency_keys(expires_at);

-- Insert migration version
INSERT INTO schema_migrations (version) VALUES ('001') ON CONFLICT DO NOTHING; 