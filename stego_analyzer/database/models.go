package database

type User struct {
	ID             int64  `db:"user_id" json:"user_id"`
	Name           string `db:"name" json:"name"`
	Email          string `db:"email" json:"email"`
	HashedPassword string `db:"hashed_password" json:"-"`
}
