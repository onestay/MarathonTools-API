package donationProviders

type MockDonationProvider struct {
	InitalValue float64
}

func (m *MockDonationProvider) GetTotalAmount() float64 {
	return m.InitalValue + 1
}
