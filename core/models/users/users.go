package users

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kirsle/blog/core/jsondb"
	"golang.org/x/crypto/bcrypt"
)

// DB is a reference to the parent app's JsonDB object.
var DB *jsondb.DB

// HashCost is the cost value given to bcrypt to hash passwords.
// TODO: make configurable from main package
var HashCost = 14

// User holds information about a user account.
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Admin    bool   `json:"admin"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

// ByName model maps usernames to their IDs.
type ByName struct {
	ID int `json:"id"`
}

// Create a new user.
func Create(u *User) error {
	// Sanity checks.
	u.Username = Normalize(u.Username)
	if len(u.Username) == 0 {
		return errors.New("username is required")
	} else if len(u.Password) == 0 {
		return errors.New("password is required")
	}

	// Make sure the username is available.
	if UsernameExists(u.Username) {
		return errors.New("that username already exists")
	}

	// Assign the next ID.
	u.ID = nextID()

	// Hash the password.
	u.SetPassword(u.Password)

	// TODO: check existing

	return u.Save()
}

// SetPassword sets a user's password by bcrypt hashing it. After this function,
// u.Password will contain the bcrypt hash.
func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), HashCost)
	if err != nil {
		return err
	}

	u.Password = string(hash)
	return nil
}

// UsernameExists checks if a username is taken.
func UsernameExists(username string) bool {
	username = Normalize(username)
	return DB.Exists("users/by-name/" + username)
}

// LoadUsername loads a user by username.
func LoadUsername(username string) (*User, error) {
	username = Normalize(username)
	u := &User{}

	// Look up the user ID by name.
	name := ByName{}
	err := DB.Get("users/by-name/"+username, &name)
	if err != nil {
		return u, fmt.Errorf("failed to look up user ID for username %s: %v", username, err)
	}

	// And load that user.
	return Load(name.ID)
}

// Load a user by their ID number.
func Load(id int) (*User, error) {
	u := &User{}
	err := DB.Get(fmt.Sprintf("users/by-id/%d", id), &u)
	return u, err
}

// Save the user.
func (u *User) Save() error {
	// Sanity check that we have an ID.
	if u.ID == 0 {
		return errors.New("can't save: user does not have an ID!")
	}

	// Save the main DB file.
	err := DB.Commit(u.key(), u)
	if err != nil {
		return err
	}

	// The username to ID mapping.
	err = DB.Commit(u.nameKey(), ByName{u.ID})
	if err != nil {
		return err
	}

	return nil
}

// Get the next user ID number.
func nextID() int {
	// Highest ID seen so far.
	var highest int

	users, err := DB.List("users/by-id")
	if err != nil {
		panic(err)
	}

	for _, doc := range users {
		fields := strings.Split(doc, "/")
		id, err := strconv.Atoi(fields[len(fields)-1])
		if err != nil {
			continue
		}

		if id > highest {
			highest = id
		}
	}

	// Return the highest +1
	return highest + 1
}

// DB key for users by ID number.
func (u *User) key() string {
	return fmt.Sprintf("users/by-id/%d", u.ID)
}

// DB key for users by username.
func (u *User) nameKey() string {
	return "users/by-name/" + u.Username
}

func (u *User) DocumentPath() string {
	return "users/by-id/%s"
}
