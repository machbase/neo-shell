package security

type AuthServer interface {
	ValidateClientToken(token string) (bool, error)
}
