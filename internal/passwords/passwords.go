package passwords

type User struct {
	Username    string
	Password    string
	AccessLevel int
	Active      bool
}

func NewUser() *User {
	return &User{
		Username:    "admin",
		Password:    "secret",
		AccessLevel: 1,
		Active:      true,
	}
}
