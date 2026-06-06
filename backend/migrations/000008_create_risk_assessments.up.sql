ALTER TYPE verification_status ADD VALUE IF NOT EXISTS 'additional_docs_required';

ALTER TABLE kyc_verifications
    ADD COLUMN IF NOT EXISTS risk_level  VARCHAR(20) NOT NULL DEFAULT 'low',
    ADD COLUMN IF NOT EXISTS risk_score  INTEGER     NOT NULL DEFAULT 0;

ALTER TABLE kyb_verifications
    ADD COLUMN IF NOT EXISTS risk_level  VARCHAR(20) NOT NULL DEFAULT 'low',
    ADD COLUMN IF NOT EXISTS risk_score  INTEGER     NOT NULL DEFAULT 0;

CREATE TABLE risk_assessments (
    id           UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type  VARCHAR(10) NOT NULL CHECK (entity_type IN ('kyc', 'kyb')),
    entity_id    UUID        NOT NULL,
    risk_level   VARCHAR(20) NOT NULL CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    risk_score   INTEGER     NOT NULL CHECK (risk_score >= 0 AND risk_score <= 100),
    risk_factors JSONB       NOT NULL DEFAULT '{}',
    assessed_by  UUID        REFERENCES users(id),
    notes        TEXT        NOT NULL DEFAULT '',
    assessed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_risk_entity ON risk_assessments(entity_type, entity_id);
