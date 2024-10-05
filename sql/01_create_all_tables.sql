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