CREATE TYPE company_status AS ENUM ('pending', 'active', 'inactive');

CREATE TABLE companies (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id          UUID REFERENCES users(id),
    name             VARCHAR(255) NOT NULL,
    legal_name       VARCHAR(255) NOT NULL DEFAULT '',
    registration_no  VARCHAR(100) NOT NULL UNIQUE,
    tax_id           VARCHAR(50)  NOT NULL DEFAULT '',
    industry         VARCHAR(100) NOT NULL DEFAULT '',
    address          TEXT         NOT NULL DEFAULT '',
    city             VARCHAR(100) NOT NULL DEFAULT '',
    province         VARCHAR(100) NOT NULL DEFAULT '',
    postal_code      VARCHAR(20)  NOT NULL DEFAULT '',
    country          VARCHAR(100) NOT NULL DEFAULT 'Indonesia',
    phone_number     VARCHAR(50)  NOT NULL DEFAULT '',
    email            VARCHAR(255) NOT NULL DEFAULT '',
    website          VARCHAR(255) NOT NULL DEFAULT '',
    status           company_status NOT NULL DEFAULT 'pending',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ
);

CREATE INDEX idx_companies_user_id        ON companies(user_id);
CREATE INDEX idx_companies_status         ON companies(status);
CREATE INDEX idx_companies_registration   ON companies(registration_no);
CREATE INDEX idx_companies_deleted_at     ON companies(deleted_at);
