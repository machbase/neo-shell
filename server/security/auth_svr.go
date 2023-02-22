package security

type AuthServer interface {
	ValidateClientToken(token string) (bool, error)
	ValidateClientCertificate(clientId string, certHash string) (bool, error)
}
