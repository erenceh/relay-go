package auth

// AuthService validates credentials and identity tokens.
// Register creates a new account, Login authenticates and returns a session token,
// and Validate verifies a token and returns the associated username.
type AuthService interface {
	// Register creates a new user with the given username and plaintext password.
	// Returns an error if the username is already taken or the input is invalid.
	Register(username, password string) error
	// Login authenticates the user and returns a session token on success.
	Login(username, password string) (accessToken, refreshToken string, err error)
	// Validate checks the session token and returns the associated username.
	Validate(tokenString string) (username, userID string, err error)
	// Refresh validates the given refresh token, revokes it, and returns a new access token
	// and a new refresh token. Returns an error if the token is missing or unrecognized.
	Refresh(refreshToken string) (accessToken, newRefreshToken string, err error)
	// IssueRefreshToken generates a cryptographically random refresh token for the given username
	// and stores it for later validation. Returns an error if the username is empty or token generation fails.
	IssueRefreshToken(username string) (refreshToken string, err error)
}
