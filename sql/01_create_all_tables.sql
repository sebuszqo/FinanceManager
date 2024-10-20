CREATE TABLE if NOT EXISTS users (
                        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                        email VARCHAR(35) UNIQUE NOT NULL,
                        login VARCHAR(30) UNIQUE NOT NULL,
                        password_hash VARCHAR(255) NOT NULL,
                        two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE,
                        two_factor_method VARCHAR(50),
                        hash_token VARCHAR(255),
                        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        is_active BOOLEAN DEFAULT FALSE
);

CREATE TABLE if NOT EXISTS user_email_verification_codes (
                        user_id UUID PRIMARY KEY,
                        code VARCHAR(6) NOT NULL,
                        expires_at TIMESTAMP NOT NULL,
                        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                        type VARCHAR(10) not null,
                        new_email VARCHAR(35),
                        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_email_verification_user_id ON user_email_verification_codes (user_id);

CREATE TABLE if NOT EXISTS user_two_factor_secrets (
                        user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
                        encrypted_secret TEXT NOT NULL,
                        created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_two_factor_user_id ON user_two_factor_secrets (user_id);

CREATE TABLE if NOT EXISTS portfolios (
                        id UUID PRIMARY KEY,
                        user_id UUID REFERENCES users(id) ON DELETE CASCADE,
                        name VARCHAR(100) NOT NULL,
                        description TEXT,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE if NOT EXISTS asset_types (
                       id SERIAL PRIMARY KEY,
                       type VARCHAR(50) NOT NULL UNIQUE
);


CREATE TABLE IF NOT EXISTS assets (
    id UUID PRIMARY KEY,
    portfolio_id UUID REFERENCES portfolios(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    ticker VARCHAR(10),
    asset_type_id INT REFERENCES asset_types(id) ON DELETE RESTRICT,
    coupon_rate NUMERIC(5, 2),                  -- Bonds: coupon rate
    maturity_date DATE,                         -- Bonds: maturity_date
    face_value NUMERIC(15, 2),                  -- Bonds: face_value
    dividend_yield NUMERIC(5, 2),               -- Stocks: dividend
    accumulation BOOLEAN,                       -- ETF: accumulating (true) distributing (false)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_assets_portfolio_id ON assets(portfolio_id);

INSERT INTO asset_types (type) VALUES ('Stock');
INSERT INTO asset_types (type) VALUES ('Bond');
INSERT INTO asset_types (type) VALUES ('ETF');
INSERT INTO asset_types (type) VALUES ('Cryptocurrency');
INSERT INTO asset_types (type) VALUES ('Savings Accounts');
INSERT INTO asset_types (type) VALUES ('Cash');

CREATE TABLE if NOT EXISTS transaction_types (
                        id SERIAL PRIMARY KEY,
                        type VARCHAR(20) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY,
    asset_id UUID REFERENCES assets(id) ON DELETE CASCADE,
    transaction_type_id INT REFERENCES transaction_types(id),
    quantity NUMERIC(15, 4) NOT NULL,
    price NUMERIC(15, 2) NOT NULL,
    transaction_date TIMESTAMP NOT NULL,
    dividend_amount NUMERIC(15, 2),
    coupon_amount NUMERIC(15, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transactions_asset_id ON transactions(asset_id);

INSERT INTO transaction_types (type) VALUES ('Buy');
INSERT INTO transaction_types (type) VALUES ('Sell');
INSERT INTO transaction_types (type) VALUES ('Dividend');
INSERT INTO transaction_types (type) VALUES ('Coupon Payment');
INSERT INTO transaction_types (type) VALUES ('Withdrawal');
INSERT INTO transaction_types (type) VALUES ('Interest Payment');
INSERT INTO transaction_types (type) VALUES ('Reinvestment');
INSERT INTO transaction_types (type) VALUES ('Fee');

ALTER TABLE assets
    ADD CONSTRAINT unique_asset_per_portfolio UNIQUE (portfolio_id, name);
