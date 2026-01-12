package testutil

import (
	"health-balance/internal/models"
)

type MockDB struct {
	GetAllDatesWithDataFunc       func() ([]string, error)
	GetRecentHealthMetricsFunc    func(limit int) ([]models.HealthMetrics, error)
	GetRecentFitnessMetricsFunc   func(limit int) ([]models.FitnessMetrics, error)
	GetRecentCognitionMetricsFunc func(limit int) ([]models.CognitionMetrics, error)
	SaveHealthMetricsFunc         func(m models.HealthMetrics) error
	SaveFitnessMetricsFunc        func(m models.FitnessMetrics) error
	SaveCognitionMetricsFunc      func(m models.CognitionMetrics) error
	GetHealthMetricsByDateFunc    func(date string) (*models.HealthMetrics, error)
	GetFitnessMetricsByDateFunc   func(date string) (*models.FitnessMetrics, error)
	GetCognitionMetricsByDateFunc func(date string) (*models.CognitionMetrics, error)
	DeleteHealthMetricsFunc       func(date string) error
	DeleteFitnessMetricsFunc      func(date string) error
	DeleteCognitionMetricsFunc    func(date string) error
	GetRHRBaselineFunc            func() (int, error)
	GetUserProfileFunc            func() (*models.UserProfile, error)
	SaveUserProfileFunc           func(profile models.UserProfile) error
	SavePushSubscriptionFunc      func(sub models.PushSubscription) error
	GetAllSubscriptionsFunc       func() ([]models.PushSubscription, error)
	GetAnyPushSubscriptionFunc    func() (*models.PushSubscription, error)
	DeletePushSubscriptionFunc    func(endpoint string) error
	CloseFunc                     func() error
}

func (m *MockDB) GetAllDatesWithData() ([]string, error) {
	if m.GetAllDatesWithDataFunc != nil {
		return m.GetAllDatesWithDataFunc()
	}
	return nil, nil
}

func (m *MockDB) GetRecentHealthMetrics(limit int) ([]models.HealthMetrics, error) {
	if m.GetRecentHealthMetricsFunc != nil {
		return m.GetRecentHealthMetricsFunc(limit)
	}
	return nil, nil
}

func (m *MockDB) GetRecentFitnessMetrics(limit int) ([]models.FitnessMetrics, error) {
	if m.GetRecentFitnessMetricsFunc != nil {
		return m.GetRecentFitnessMetricsFunc(limit)
	}
	return nil, nil
}

func (m *MockDB) GetRecentCognitionMetrics(limit int) ([]models.CognitionMetrics, error) {
	if m.GetRecentCognitionMetricsFunc != nil {
		return m.GetRecentCognitionMetricsFunc(limit)
	}
	return nil, nil
}

func (m *MockDB) SaveHealthMetrics(m1 models.HealthMetrics) error {
	if m.SaveHealthMetricsFunc != nil {
		return m.SaveHealthMetricsFunc(m1)
	}
	return nil
}

func (m *MockDB) SaveFitnessMetrics(m1 models.FitnessMetrics) error {
	if m.SaveFitnessMetricsFunc != nil {
		return m.SaveFitnessMetricsFunc(m1)
	}
	return nil
}

func (m *MockDB) SaveCognitionMetrics(m1 models.CognitionMetrics) error {
	if m.SaveCognitionMetricsFunc != nil {
		return m.SaveCognitionMetricsFunc(m1)
	}
	return nil
}

func (m *MockDB) GetHealthMetricsByDate(date string) (*models.HealthMetrics, error) {
	if m.GetHealthMetricsByDateFunc != nil {
		return m.GetHealthMetricsByDateFunc(date)
	}
	return nil, nil
}

func (m *MockDB) GetFitnessMetricsByDate(date string) (*models.FitnessMetrics, error) {
	if m.GetFitnessMetricsByDateFunc != nil {
		return m.GetFitnessMetricsByDateFunc(date)
	}
	return nil, nil
}

func (m *MockDB) GetCognitionMetricsByDate(date string) (*models.CognitionMetrics, error) {
	if m.GetCognitionMetricsByDateFunc != nil {
		return m.GetCognitionMetricsByDateFunc(date)
	}
	return nil, nil
}

func (m *MockDB) DeleteHealthMetrics(date string) error {
	if m.DeleteHealthMetricsFunc != nil {
		return m.DeleteHealthMetricsFunc(date)
	}
	return nil
}

func (m *MockDB) DeleteFitnessMetrics(date string) error {
	if m.DeleteFitnessMetricsFunc != nil {
		return m.DeleteFitnessMetricsFunc(date)
	}
	return nil
}

func (m *MockDB) DeleteCognitionMetrics(date string) error {
	if m.DeleteCognitionMetricsFunc != nil {
		return m.DeleteCognitionMetricsFunc(date)
	}
	return nil
}

func (m *MockDB) GetRHRBaseline() (int, error) {
	if m.GetRHRBaselineFunc != nil {
		return m.GetRHRBaselineFunc()
	}
	return 0, nil
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

func (m *MockDB) SavePushSubscription(sub models.PushSubscription) error {
	if m.SavePushSubscriptionFunc != nil {
		return m.SavePushSubscriptionFunc(sub)
	}
	return nil
}

func (m *MockDB) GetAllSubscriptions() ([]models.PushSubscription, error) {
	if m.GetAllSubscriptionsFunc != nil {
		return m.GetAllSubscriptionsFunc()
	}
	return nil, nil
}

func (m *MockDB) GetAnyPushSubscription() (*models.PushSubscription, error) {
	if m.GetAnyPushSubscriptionFunc != nil {
		return m.GetAnyPushSubscriptionFunc()
	}
	return nil, nil
}

func (m *MockDB) DeletePushSubscription(endpoint string) error {
	if m.DeletePushSubscriptionFunc != nil {
		return m.DeletePushSubscriptionFunc(endpoint)
	}
	return nil
}

func (m *MockDB) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
