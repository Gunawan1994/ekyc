package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

type CalculateKYCRiskInput struct {
	KYCID          uuid.UUID
	IDDocumentURL  string
	SelfieURL      string
	LivenessScore  float64
	FaceMatchScore float64
	AssessedBy     *uuid.UUID
}

type CalculateKYBRiskInput struct {
	KYBID            uuid.UUID
	BusinessDocURL   string
	TaxDocURL        string
	DirectorIDDocURL string
	AssessedBy       *uuid.UUID

	// Company context for enhanced scoring
	CompanyCreatedAt time.Time
	CompanyIndustry  string
	CompanyStatus    string // "active", "inactive", "pending"
}

type ManualRiskOverrideInput struct {
	EntityType string
	EntityID   uuid.UUID
	RiskLevel  domain.RiskLevel
	Notes      string
	AssessedBy uuid.UUID
}

type RiskUsecase interface {
	CalculateKYCRisk(ctx context.Context, input CalculateKYCRiskInput) (*domain.RiskAssessment, error)
	CalculateKYBRisk(ctx context.Context, input CalculateKYBRiskInput) (*domain.RiskAssessment, error)
	ManualOverride(ctx context.Context, input ManualRiskOverrideInput) (*domain.RiskAssessment, error)
	GetLatest(ctx context.Context, entityType string, entityID uuid.UUID) (*domain.RiskAssessment, error)
	ListHistory(ctx context.Context, entityType string, entityID uuid.UUID) ([]domain.RiskAssessment, error)
}

type riskUsecase struct {
	riskRepo domain.RiskRepository
}

func NewRiskUsecase(riskRepo domain.RiskRepository) RiskUsecase {
	return &riskUsecase{riskRepo: riskRepo}
}

func scoreToLevel(score int) domain.RiskLevel {
	switch {
	case score <= 25:
		return domain.RiskLevelLow
	case score <= 50:
		return domain.RiskLevelMedium
	case score <= 75:
		return domain.RiskLevelHigh
	default:
		return domain.RiskLevelCritical
	}
}

// CalculateKYCRisk scores a KYC submission based on document completeness and
// biometric scores, persists the result, and updates the kyc_verifications row.
func (u *riskUsecase) CalculateKYCRisk(ctx context.Context, input CalculateKYCRiskInput) (*domain.RiskAssessment, error) {
	score := 0
	factors := map[string]any{}

	if input.IDDocumentURL == "" {
		score += 35
		factors["missing_id_document"] = true
	} else {
		factors["missing_id_document"] = false
	}

	if input.SelfieURL == "" {
		score += 25
		factors["missing_selfie"] = true
	} else {
		factors["missing_selfie"] = false
	}

	if input.LivenessScore == 0 {
		score += 20
		factors["liveness_score"] = 0
		factors["liveness_risk"] = "not_checked"
	} else if input.LivenessScore < 0.5 {
		score += 15
		factors["liveness_score"] = input.LivenessScore
		factors["liveness_risk"] = "low_score"
	} else if input.LivenessScore < 0.8 {
		score += 5
		factors["liveness_score"] = input.LivenessScore
		factors["liveness_risk"] = "moderate_score"
	} else {
		factors["liveness_score"] = input.LivenessScore
		factors["liveness_risk"] = "good"
	}

	if input.FaceMatchScore == 0 {
		score += 20
		factors["face_match_score"] = 0
		factors["face_match_risk"] = "not_checked"
	} else if input.FaceMatchScore < 0.5 {
		score += 15
		factors["face_match_score"] = input.FaceMatchScore
		factors["face_match_risk"] = "low_score"
	} else if input.FaceMatchScore < 0.8 {
		score += 5
		factors["face_match_score"] = input.FaceMatchScore
		factors["face_match_risk"] = "moderate_score"
	} else {
		factors["face_match_score"] = input.FaceMatchScore
		factors["face_match_risk"] = "good"
	}

	if score > 100 {
		score = 100
	}
	level := scoreToLevel(score)

	now := time.Now().UTC()
	ra := &domain.RiskAssessment{
		ID:          uuid.New(),
		EntityType:  "kyc",
		EntityID:    input.KYCID,
		RiskLevel:   level,
		RiskScore:   score,
		RiskFactors: factors,
		AssessedBy:  input.AssessedBy,
		AssessedAt:  now,
	}

	if err := u.riskRepo.Create(ctx, ra); err != nil {
		return nil, fmt.Errorf("risk usecase – kyc create: %w", err)
	}
	if err := u.riskRepo.UpdateKYCRisk(ctx, input.KYCID, level, score); err != nil {
		return nil, fmt.Errorf("risk usecase – kyc update row: %w", err)
	}

	return ra, nil
}

var highRiskIndustryKeywords = []string{
	"crypto", "bitcoin", "gambling", "judi", "pinjol", "lending",
	"money transfer", "remittance", "forex", "weapons", "senjata",
	"digital asset", "aset digital",
}

var mediumRiskIndustryKeywords = []string{
	"fintech", "financial", "keuangan", "real estate", "properti",
	"mining", "tambang", "construction", "konstruksi", "import", "ekspor",
}

func industryRiskScore(industry string) (int, string) {
	lower := strings.ToLower(industry)
	for _, kw := range highRiskIndustryKeywords {
		if strings.Contains(lower, kw) {
			return 25, "high"
		}
	}
	for _, kw := range mediumRiskIndustryKeywords {
		if strings.Contains(lower, kw) {
			return 10, "medium"
		}
	}
	return 0, "low"
}

// CalculateKYBRisk scores a KYB submission based on business document completeness.
func (u *riskUsecase) CalculateKYBRisk(ctx context.Context, input CalculateKYBRiskInput) (*domain.RiskAssessment, error) {
	score := 0
	factors := map[string]any{}

	if input.BusinessDocURL == "" {
		score += 35
		factors["missing_business_doc"] = true
	} else {
		factors["missing_business_doc"] = false
	}

	if input.TaxDocURL == "" {
		score += 30
		factors["missing_tax_doc"] = true
	} else {
		factors["missing_tax_doc"] = false
	}

	if input.DirectorIDDocURL == "" {
		score += 25
		factors["missing_director_id"] = true
	} else {
		factors["missing_director_id"] = false
	}

	missingCount := 0
	if input.BusinessDocURL == "" {
		missingCount++
	}
	if input.TaxDocURL == "" {
		missingCount++
	}
	if input.DirectorIDDocURL == "" {
		missingCount++
	}
	if missingCount >= 2 {
		score += 10
		factors["multiple_docs_missing"] = true
	}

	if !input.CompanyCreatedAt.IsZero() {
		ageMonths := int(time.Since(input.CompanyCreatedAt).Hours() / 730)
		var ageRisk string
		switch {
		case ageMonths < 6:
			score += 30
			ageRisk = "very_new"
		case ageMonths < 12:
			score += 20
			ageRisk = "new"
		case ageMonths < 36:
			score += 10
			ageRisk = "moderate"
		case ageMonths < 60:
			score += 5
			ageRisk = "established"
		default:
			ageRisk = "mature"
		}
		factors["company_age_months"] = ageMonths
		factors["company_age_risk"] = ageRisk
	}

	if input.CompanyIndustry != "" {
		industryScore, industryRisk := industryRiskScore(input.CompanyIndustry)
		score += industryScore
		factors["industry"] = input.CompanyIndustry
		factors["industry_risk"] = industryRisk
	}

	if input.CompanyStatus != "" {
		switch input.CompanyStatus {
		case "inactive":
			score += 25
			factors["company_status_risk"] = "high"
		case "pending":
			score += 10
			factors["company_status_risk"] = "medium"
		default:
			factors["company_status_risk"] = "low"
		}
		factors["company_status"] = input.CompanyStatus
	}

	if score > 100 {
		score = 100
	}
	level := scoreToLevel(score)

	now := time.Now().UTC()
	ra := &domain.RiskAssessment{
		ID:          uuid.New(),
		EntityType:  "kyb",
		EntityID:    input.KYBID,
		RiskLevel:   level,
		RiskScore:   score,
		RiskFactors: factors,
		AssessedBy:  input.AssessedBy,
		AssessedAt:  now,
	}

	if err := u.riskRepo.Create(ctx, ra); err != nil {
		return nil, fmt.Errorf("risk usecase – kyb create: %w", err)
	}
	if err := u.riskRepo.UpdateKYBRisk(ctx, input.KYBID, level, score); err != nil {
		return nil, fmt.Errorf("risk usecase – kyb update row: %w", err)
	}

	return ra, nil
}

// ManualOverride allows a risk_analyst to manually set the risk level.
func (u *riskUsecase) ManualOverride(ctx context.Context, input ManualRiskOverrideInput) (*domain.RiskAssessment, error) {
	levelScores := map[domain.RiskLevel]int{
		domain.RiskLevelLow:      10,
		domain.RiskLevelMedium:   40,
		domain.RiskLevelHigh:     65,
		domain.RiskLevelCritical: 90,
	}
	score := levelScores[input.RiskLevel]

	now := time.Now().UTC()
	assessedBy := input.AssessedBy
	ra := &domain.RiskAssessment{
		ID:          uuid.New(),
		EntityType:  input.EntityType,
		EntityID:    input.EntityID,
		RiskLevel:   input.RiskLevel,
		RiskScore:   score,
		RiskFactors: map[string]any{"manual_override": true, "set_by": assessedBy.String()},
		AssessedBy:  &assessedBy,
		Notes:       input.Notes,
		AssessedAt:  now,
	}

	if err := u.riskRepo.Create(ctx, ra); err != nil {
		return nil, fmt.Errorf("risk usecase – manual override create: %w", err)
	}

	switch input.EntityType {
	case "kyc":
		if err := u.riskRepo.UpdateKYCRisk(ctx, input.EntityID, input.RiskLevel, score); err != nil {
			return nil, fmt.Errorf("risk usecase – manual override kyc update: %w", err)
		}
	case "kyb":
		if err := u.riskRepo.UpdateKYBRisk(ctx, input.EntityID, input.RiskLevel, score); err != nil {
			return nil, fmt.Errorf("risk usecase – manual override kyb update: %w", err)
		}
	}

	return ra, nil
}

func (u *riskUsecase) GetLatest(ctx context.Context, entityType string, entityID uuid.UUID) (*domain.RiskAssessment, error) {
	ra, err := u.riskRepo.FindLatestByEntity(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("risk usecase – get latest: %w", err)
	}
	return ra, nil
}

func (u *riskUsecase) ListHistory(ctx context.Context, entityType string, entityID uuid.UUID) ([]domain.RiskAssessment, error) {
	list, err := u.riskRepo.ListByEntity(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("risk usecase – list history: %w", err)
	}
	return list, nil
}
