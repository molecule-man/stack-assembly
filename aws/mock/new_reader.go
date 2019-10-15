// +build awsmock,acceptance

package mock

func New(testID, featureID, scenarioID string) *ReadProvider {
	return &ReadProvider{
		testID:     testID,
		featureID:  featureID,
		scenarioID: scenarioID,
	}
}

func IsMockEnabled() bool {
	return true
}
