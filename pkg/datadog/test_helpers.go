package datadog

// If running mockgen on this package, you will unfortunately need to comment everything out below beforehand
// and then uncomment again afterwards

import (
	"os"

	"github.com/golang/mock/gomock"

	mocks "github.com/fairwindsops/astro/pkg/mocks"
)

// GetMock will return a mock datadog client API
func GetMock(ctrl *gomock.Controller) *mocks.MockClientAPI {
	os.Setenv("DEFINITIONS_PATH", "../config/test_conf.yml")
	os.Setenv("DD_API_KEY", "test")
	os.Setenv("DD_APP_KEY", "test")

	ddMon := GetInstance()
	ddMock := mocks.NewMockClientAPI(ctrl)
	ddMon.Datadog = ddMock

	return ddMock
}
