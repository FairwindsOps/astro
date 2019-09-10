package datadog

import (
	"os"

	mocks "github.com/fairwindsops/astro/pkg/mocks"
	"github.com/golang/mock/gomock"
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
