// SPDX-License-Identifier: AGPL-3.0-only AND MIT
// From https://github.com/racingmars/go3270/: Copyright 2025 by
// Matthew R. Wilson, under the MIT license (LICENSE-MIT in the project
// root).  D-Modem's changes: Copyright (C) 2026 D-Modem Authors, under
// the AGPL-3.0-only (LICENSE).

// This example demonstrates using the RunTransactions() approach to
// structuring a go3270 application.

// This file contains a mock in-memory database for the purposes of our
// example application.

package main

import (
	"errors"
	"sync"
	"time"
)

type User struct {
	Username   string
	Password   string
	Name       string
	SignupDate time.Time
}

type DB interface {
	GetUser(username string) (User, error)
	CreateUser(user User) (User, error)
	UpdateUser(user User) (User, error)
}

type dbstate struct {
	lock  sync.Mutex
	users map[string]User
}

// Connect returns a new, empty in-memory DB.
func Connect() DB {
	return &dbstate{
		users: make(map[string]User),
	}
}

var ErrUserExists = errors.New("username already exists")
var ErrNotFound = errors.New("record not found")

// GetUser returns the user with the given username, or ErrNotFound.
func (db *dbstate) GetUser(username string) (User, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	user, ok := db.users[username]
	if !ok {
		return User{}, ErrNotFound
	}

	return user, nil
}

// CreateUser stores a new user, setting its signup date to now, and returns the
// stored record. It returns ErrUserExists if the username is taken.
func (db *dbstate) CreateUser(user User) (User, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	if _, ok := db.users[user.Username]; ok {
		return User{}, ErrUserExists
	}

	user.SignupDate = time.Now().UTC().Truncate(time.Second)

	// Obviously in a real application you wouldn't store plaintext password,
	// but this is a simple example for the purposes of demonstrating go3270,
	// not demonstrating how to build a secure application.
	db.users[user.Username] = user

	return user, nil
}

// UpdateUser replaces an existing user's record, keeping its signup date, and
// its password if user.Password is empty. It returns the stored record, or
// ErrNotFound.
func (db *dbstate) UpdateUser(user User) (User, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	olduser, ok := db.users[user.Username]
	if !ok {
		return User{}, ErrNotFound
	}

	// Make sure original signup date is maintained
	user.SignupDate = olduser.SignupDate

	// Carry over existing password if we aren't setting a new password
	if user.Password == "" {
		user.Password = olduser.Password
	}

	db.users[user.Username] = user

	return user, nil
}
