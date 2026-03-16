package model

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
	APIV1Prefix                        = "api/v1"
)
