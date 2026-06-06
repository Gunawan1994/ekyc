package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
)

const (
	dashboardStatsCacheKey = "dashboard:stats"
	dashboardStatsTTL      = time.Minute
)

// DashboardStats holds aggregated platform-wide counters for the dashboard.
type DashboardStats struct {
	TotalCustomers   int64 `json:"total_customers"`
	TotalCompanies   int64 `json:"total_companies"`
	TotalKYCPending  int64 `json:"total_kyc_pending"`
	TotalKYCApproved int64 `json:"total_kyc_approved"`
	TotalKYCRejected int64 `json:"total_kyc_rejected"`
	TotalKYBPending  int64 `json:"total_kyb_pending"`
	TotalKYBApproved int64 `json:"total_kyb_approved"`
	TotalKYBRejected int64 `json:"total_kyb_rejected"`
	// Risk summary derived from KYB verification statuses.
	// LowRiskCompanies: companies whose latest KYB is approved.
	// MediumRiskCompanies: companies with a pending KYB.
	// HighRiskCompanies: companies with a rejected KYB.
	LowRiskCompanies    int64 `json:"low_risk_companies"`
	MediumRiskCompanies int64 `json:"medium_risk_companies"`
	HighRiskCompanies   int64 `json:"high_risk_companies"`
}

// DashboardUsecase defines the application-level operations for the dashboard.
type DashboardUsecase interface {
	GetStats(ctx context.Context) (*DashboardStats, error)
}

// dashboardCache is the subset of the cache interface this usecase needs.
type dashboardCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
}

type dashboardUsecase struct {
	customerRepo domain.CustomerRepository
	companyRepo  domain.CompanyRepository
	kycRepo      domain.KYCRepository
	kybRepo      domain.KYBRepository
	cache        dashboardCache
}

// NewDashboardUsecase constructs a DashboardUsecase with the provided
// dependencies.
func NewDashboardUsecase(
	customerRepo domain.CustomerRepository,
	companyRepo domain.CompanyRepository,
	kycRepo domain.KYCRepository,
	kybRepo domain.KYBRepository,
	cache dashboardCache,
) DashboardUsecase {
	return &dashboardUsecase{
		customerRepo: customerRepo,
		companyRepo:  companyRepo,
		kycRepo:      kycRepo,
		kybRepo:      kybRepo,
		cache:        cache,
	}
}

// GetStats returns aggregated platform statistics.  Results are cached in
// Redis under the key "dashboard:stats" for 1 minute.  On a cache miss (or
// any cache error) the stats are computed fresh from the repositories and the
// result is written back to the cache before returning.
func (u *dashboardUsecase) GetStats(ctx context.Context) (*DashboardStats, error) {
	if stats, ok := u.loadFromCache(ctx); ok {
		return stats, nil
	}

	stats, err := u.computeStats(ctx)
	if err != nil {
		return nil, err
	}

	u.saveToCache(ctx, stats)

	return stats, nil
}

// loadFromCache attempts to read DashboardStats from the cache.  It returns
// (stats, true) on a hit and (nil, false) on a miss or error.
func (u *dashboardUsecase) loadFromCache(ctx context.Context) (*DashboardStats, bool) {
	raw, err := u.cache.Get(ctx, dashboardStatsCacheKey)
	if err != nil || raw == "" {
		return nil, false
	}

	var stats DashboardStats
	if err := json.Unmarshal([]byte(raw), &stats); err != nil {
		return nil, false
	}

	return &stats, true
}

// saveToCache writes DashboardStats to the cache with a 1-minute TTL.  Errors
// are discarded so that a cache write failure never surfaces to the caller.
func (u *dashboardUsecase) saveToCache(ctx context.Context, stats *DashboardStats) {
	raw, err := json.Marshal(stats)
	if err != nil {
		return
	}
	_ = u.cache.Set(ctx, dashboardStatsCacheKey, string(raw), dashboardStatsTTL)
}

// computeStats queries every repository in parallel and assembles the result.
// Any single repository error cancels the whole computation.
func (u *dashboardUsecase) computeStats(ctx context.Context) (*DashboardStats, error) {
	type result struct {
		field string
		value int64
		err   error
	}

	jobs := []struct {
		field string
		fn    func() (int64, error)
	}{
		{"total_customers", func() (int64, error) {
			_, total, err := u.customerRepo.FindAll(ctx, domain.ListParams{Page: 1, PageSize: 1})
			return total, err
		}},
		{"total_companies", func() (int64, error) {
			_, total, err := u.companyRepo.FindAll(ctx, domain.ListParams{Page: 1, PageSize: 1})
			return total, err
		}},
		{"total_kyc_pending", func() (int64, error) {
			return u.kycRepo.CountByStatus(ctx, domain.VerificationStatusPending)
		}},
		{"total_kyc_approved", func() (int64, error) {
			return u.kycRepo.CountByStatus(ctx, domain.VerificationStatusApproved)
		}},
		{"total_kyc_rejected", func() (int64, error) {
			return u.kycRepo.CountByStatus(ctx, domain.VerificationStatusRejected)
		}},
		{"total_kyb_pending", func() (int64, error) {
			return u.kybRepo.CountByStatus(ctx, domain.VerificationStatusPending)
		}},
		{"total_kyb_approved", func() (int64, error) {
			return u.kybRepo.CountByStatus(ctx, domain.VerificationStatusApproved)
		}},
		{"total_kyb_rejected", func() (int64, error) {
			return u.kybRepo.CountByStatus(ctx, domain.VerificationStatusRejected)
		}},
	}

	ch := make(chan result, len(jobs))

	for _, j := range jobs {
		j := j
		go func() {
			v, err := j.fn()
			ch <- result{field: j.field, value: v, err: err}
		}()
	}

	stats := &DashboardStats{}
	for range jobs {
		r := <-ch
		if r.err != nil {
			return nil, fmt.Errorf("dashboard compute stats – %s: %w", r.field, r.err)
		}
		switch r.field {
		case "total_customers":
			stats.TotalCustomers = r.value
		case "total_companies":
			stats.TotalCompanies = r.value
		case "total_kyc_pending":
			stats.TotalKYCPending = r.value
		case "total_kyc_approved":
			stats.TotalKYCApproved = r.value
		case "total_kyc_rejected":
			stats.TotalKYCRejected = r.value
		case "total_kyb_pending":
			stats.TotalKYBPending = r.value
		case "total_kyb_approved":
			stats.TotalKYBApproved = r.value
		case "total_kyb_rejected":
			stats.TotalKYBRejected = r.value
		}
	}

	return stats, nil
}
