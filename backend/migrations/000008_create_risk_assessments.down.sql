DROP TABLE IF EXISTS risk_assessments;

ALTER TABLE kyb_verifications
    DROP COLUMN IF EXISTS risk_score,
    DROP COLUMN IF EXISTS risk_level;

ALTER TABLE kyc_verifications
    DROP COLUMN IF EXISTS risk_score,
    DROP COLUMN IF EXISTS risk_level;
