CREATE TYPE verification_status AS ENUM ('pending', 'in_review', 'approved', 'rejected');

CREATE TABLE kyc_verifications (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id      UUID NOT NULL REFERENCES customers(id),
    reviewer_id      UUID REFERENCES users(id),
    submitted_by     UUID NOT NULL REFERENCES users(id),
    status           verification_status NOT NULL DEFAULT 'pending',
    id_document_url  TEXT NOT NULL DEFAULT '',
    selfie_url       TEXT NOT NULL DEFAULT '',
    liveness_score   NUMERIC(5,2) NOT NULL DEFAULT 0,
    face_match_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    rejection_reason TEXT NOT NULL DEFAULT '',
    notes            TEXT NOT NULL DEFAULT '',
    submitted_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE kyb_verifications (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id           UUID NOT NULL REFERENCES companies(id),
    reviewer_id          UUID REFERENCES users(id),
    submitted_by         UUID NOT NULL REFERENCES users(id),
    status               verification_status NOT NULL DEFAULT 'pending',
    business_doc_url     TEXT NOT NULL DEFAULT '',
    tax_doc_url          TEXT NOT NULL DEFAULT '',
    director_id_doc_url  TEXT NOT NULL DEFAULT '',
    rejection_reason     TEXT NOT NULL DEFAULT '',
    notes                TEXT NOT NULL DEFAULT '',
    submitted_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at          TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_kyc_customer_id  ON kyc_verifications(customer_id);
CREATE INDEX idx_kyc_reviewer_id  ON kyc_verifications(reviewer_id);
CREATE INDEX idx_kyc_status       ON kyc_verifications(status);
CREATE INDEX idx_kyc_submitted_at ON kyc_verifications(submitted_at DESC);

CREATE INDEX idx_kyb_company_id   ON kyb_verifications(company_id);
CREATE INDEX idx_kyb_reviewer_id  ON kyb_verifications(reviewer_id);
CREATE INDEX idx_kyb_status       ON kyb_verifications(status);
CREATE INDEX idx_kyb_submitted_at ON kyb_verifications(submitted_at DESC);
