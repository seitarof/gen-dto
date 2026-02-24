package dst

type ProviderPayloadDTO struct {
	Code string
}

type PatientProviderType = ProviderPayloadDTO

type PatientDTO struct {
	Provider PatientProviderType
}
