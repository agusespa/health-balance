package testutil

import (
	"health-balance/internal/models"
)

type MockDB struct {
	GetUserProfileFunc   func() (*models.UserProfile, error)
	SaveUserProfileFunc  func(profile models.UserProfile) error
	SaveIngestFunc       func(run models.IngestRun, values []models.DailyMetric) (int64, error)
	GetDailyMetricsFunc  func(from, to string, metrics ...string) ([]models.DailyMetric, error)
	GetLastIngestRunFunc func(kind string) (*models.IngestRun, error)
	SaveInsightFunc      func(in models.Insight) error
	GetLatestInsightFunc func() (*models.Insight, error)
	CloseFunc            func() error
}

func (m *MockDB) GetUserProfile() (*models.UserProfile, error) {
	if m.GetUserProfileFunc != nil {
		return m.GetUserProfileFunc()
	}
	return nil, nil
}

func (m *MockDB) SaveUserProfile(profile models.UserProfile) error {
	if m.SaveUserProfileFunc != nil {
		return m.SaveUserProfileFunc(profile)
	}
	return nil
}

func (m *MockDB) SaveIngest(run models.IngestRun, values []models.DailyMetric) (int64, error) {
	if m.SaveIngestFunc != nil {
		return m.SaveIngestFunc(run, values)
	}
	return 0, nil
}

func (m *MockDB) GetDailyMetrics(from, to string, metrics ...string) ([]models.DailyMetric, error) {
	if m.GetDailyMetricsFunc != nil {
		return m.GetDailyMetricsFunc(from, to, metrics...)
	}
	return nil, nil
}

func (m *MockDB) GetLastIngestRun(kind string) (*models.IngestRun, error) {
	if m.GetLastIngestRunFunc != nil {
		return m.GetLastIngestRunFunc(kind)
	}
	return nil, nil
}

func (m *MockDB) SaveInsight(in models.Insight) error {
	if m.SaveInsightFunc != nil {
		return m.SaveInsightFunc(in)
	}
	return nil
}

func (m *MockDB) GetLatestInsight() (*models.Insight, error) {
	if m.GetLatestInsightFunc != nil {
		return m.GetLatestInsightFunc()
	}
	return nil, nil
}

func (m *MockDB) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
