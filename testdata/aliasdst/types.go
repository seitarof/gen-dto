package aliasdst

import "github.com/seitarof/gen-dto/testdata/aliasapi"

type PatientProviderType = aliasapi.GetPatientProviderTypeResponse

type PatientDTO struct {
	ProviderType PatientProviderType
}
