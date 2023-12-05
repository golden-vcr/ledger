package cheer

type AuthorizeCheerInflowFunc func(s string) bool

func makeAuthorizeCheerInflowFunc(secretKey string) AuthorizeCheerInflowFunc {
	return func(s string) bool {
		return s == secretKey
	}
}
