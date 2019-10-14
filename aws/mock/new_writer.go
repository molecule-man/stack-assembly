// +build !awsmock,acceptance

package mock

func New(testID, featureID, scenarioID string) *WriteProvider {
	return &WriteProvider{
		testID:     testID,
		featureID:  featureID,
		scenarioID: scenarioID,
	}
}

func IsMockEnabled() bool {
	return false
}
