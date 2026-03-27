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
	Validate(token string) (username string, err error)
	//
	Refresh(refreshToken string) (accessToken, newRefreshToken string, err error)
	//
	IssueRefreshToken(username string) (refreshToken string, err error)
}

// User represents an authenticated user record.
type User struct {
	Username     string
	PasswordHash string // bcrypt or equivalent hash — never the plaintext password
}

// NewUser constructs a User from an already-hashed password.
func NewUser(username, hashedPassword string) *User {
	return &User{
		Username:     username,
		PasswordHash: hashedPassword,
	}
}
