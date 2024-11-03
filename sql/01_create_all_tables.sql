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

ALTER TABLE assets
    ADD COLUMN total_quantity NUMERIC(15, 4) DEFAULT 0,
    ADD COLUMN average_purchase_price NUMERIC(15, 4) DEFAULT 0,
    ADD COLUMN total_invested NUMERIC(15, 2) DEFAULT 0,
    ADD COLUMN unrealized_gain_loss NUMERIC(15, 2) DEFAULT 0,
    ADD COLUMN current_value NUMERIC(15, 2) DEFAULT 0;

ALTER TABLE assets
    ADD COLUMN currency VARCHAR(10),
    ADD COLUMN exchange VARCHAR(50);

ALTER TABLE assets
    ADD COLUMN interest_accrued NUMERIC(15, 2) DEFAULT 0;

CREATE TABLE verified_tickers (
                                  ticker VARCHAR(50) PRIMARY KEY,
                                  name VARCHAR(255) NOT NULL,
                                  asset_type VARCHAR(50) NOT NULL,
                                  last_verified_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE portfolio_snapshots (
                                     id UUID PRIMARY KEY,
                                     user_id UUID,
                                     portfolio_id UUID,
                                     total_value NUMERIC(15, 2),
                                     total_invested NUMERIC(15, 2),
                                     unrealized_gain_loss NUMERIC(15, 2),
                                     snapshot_date DATE,
                                     created_at TIMESTAMP,
                                     updated_at TIMESTAMP,
                                     FOREIGN KEY (portfolio_id)
                                         REFERENCES portfolios(id)
                                         ON DELETE CASCADE
);




CREATE TABLE instruments (
                             id SERIAL PRIMARY KEY,
                             symbol VARCHAR(20) NOT NULL,
                             name VARCHAR(255) NOT NULL,
                             exchange VARCHAR(50),
                             exchange_short VARCHAR(20),
                             asset_type_id INTEGER NOT NULL REFERENCES asset_types(id),
                             price NUMERIC(18,6),
                             currency VARCHAR(10),
                             UNIQUE(symbol, exchange_short)
);

ALTER TABLE instruments ADD COLUMN updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW();


CREATE INDEX idx_instruments_symbol ON instruments (symbol);
CREATE INDEX idx_instruments_asset_type_id ON instruments (asset_type_id);
CREATE INDEX idx_instruments_name ON instruments USING GIN (to_tsvector('english', name));

CREATE TABLE predefined_categories (
                                       id SERIAL PRIMARY KEY,
                                       name VARCHAR(50) NOT NULL UNIQUE,
                                       type VARCHAR(10) CHECK (type IN ('income', 'expense')) NOT NULL
);


CREATE TABLE user_categories (
                                 id SERIAL PRIMARY KEY,
                                 name VARCHAR(50) NOT NULL,
                                 user_id UUID REFERENCES users(id) NOT NULL,
                                 UNIQUE (name, user_id)
);


CREATE TABLE payment_methods (
                                 id SERIAL PRIMARY KEY,
                                 name VARCHAR(50) NOT NULL UNIQUE
);
INSERT INTO payment_methods (name) VALUES ('card'), ('cash'), ('BLIK'), ('bank_transaction');

CREATE TABLE payment_sources (
                                 id SERIAL PRIMARY KEY,
                                 user_id UUID REFERENCES users(id) NOT NULL,           -- ID użytkownika, do którego należy źródło płatności
                                 payment_method_id INT REFERENCES payment_methods(id) NOT NULL, -- Sposób płatności, np. karta, gotówka, BLIK
                                 name VARCHAR(50) NOT NULL,                           -- Nazwa źródła, np. „karta Visa”, „konto bankowe w PKO”
                                 details JSONB                                        -- Opcjonalne szczegóły, np. numer konta dla kont bankowych
);



CREATE TABLE personal_transactions (
                                       id SERIAL PRIMARY KEY,
                                       predefined_category_id INT REFERENCES predefined_categories(id),
                                       user_category_id INT REFERENCES user_categories(id),
                                       user_id UUID REFERENCES users(id) NOT NULL,
                                       amount DECIMAL(10, 2) NOT NULL,
                                       type VARCHAR(10) CHECK (type IN ('income', 'expense')) NOT NULL,
                                       date DATE NOT NULL,
                                       description TEXT,
                                       payment_method_id INT REFERENCES payment_methods(id),
                                       payment_source_id INT REFERENCES payment_sources(id),
                                       CHECK (
                                           predefined_category_id IS NOT NULL AND
                                           (user_category_id IS NULL OR user_category_id IS NOT NULL)
                                           )
);





