package src

type ProviderPayload struct {
	Code string
}

type ProviderAlias = ProviderPayload

type Patient struct {
	Provider ProviderAlias
}
