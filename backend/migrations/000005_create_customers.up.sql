CREATE TYPE id_type AS ENUM ('ktp', 'passport', 'sim');

CREATE TABLE customers (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id        UUID REFERENCES users(id),
    company_id     UUID NOT NULL REFERENCES companies(id),
    full_name      VARCHAR(255) NOT NULL,
    date_of_birth  DATE         NOT NULL DEFAULT '1900-01-01',
    place_of_birth VARCHAR(100) NOT NULL DEFAULT '',
    gender         VARCHAR(20)  NOT NULL DEFAULT '',
    nationality    VARCHAR(100) NOT NULL DEFAULT 'Indonesia',
    id_type        id_type      NOT NULL DEFAULT 'ktp',
    id_number      VARCHAR(100) NOT NULL UNIQUE,
    address        TEXT         NOT NULL DEFAULT '',
    city           VARCHAR(100) NOT NULL DEFAULT '',
    province       VARCHAR(100) NOT NULL DEFAULT '',
    postal_code    VARCHAR(20)  NOT NULL DEFAULT '',
    country        VARCHAR(100) NOT NULL DEFAULT 'Indonesia',
    phone_number   VARCHAR(50)  NOT NULL DEFAULT '',
    email          VARCHAR(255) NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

CREATE INDEX idx_customers_company_id ON customers(company_id);
CREATE INDEX idx_customers_user_id    ON customers(user_id);
CREATE INDEX idx_customers_id_number  ON customers(id_number);
CREATE INDEX idx_customers_deleted_at ON customers(deleted_at);
