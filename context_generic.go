// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: © 2015 LabStack LLC and Echo contributors

package echo

import "errors"

// ErrNonExistentKey is error that is returned when key does not exist
var ErrNonExistentKey = errors.New("non existent key")

// ErrInvalidKeyType is error that is returned when the value is not castable to expected type.
var ErrInvalidKeyType = errors.New("invalid key type")

// ContextGet retrieves a typed value from the context store.
// Returns ErrNonExistentKey when the key is missing.
// Returns ErrInvalidKeyType when the stored value cannot be asserted to type T.
//
// This is the typed companion to Context.Get; both share the same underlying
// store and locking via storeGet.
func ContextGet[T any](c *Context, key string) (T, error) {
	val, ok := c.storeGet(key)
	if !ok {
		var zero T
		return zero, ErrNonExistentKey
	}

	typed, ok := val.(T)
	if !ok {
		var zero T
		return zero, ErrInvalidKeyType
	}

	return typed, nil
}

// ContextGetOr retrieves a value from the context store or returns a default value when the key
// is missing. Returns ErrInvalidKeyType error if the value is not castable to type T.
func ContextGetOr[T any](c *Context, key string, defaultValue T) (T, error) {
	typed, err := ContextGet[T](c, key)
	if err == ErrNonExistentKey {
		return defaultValue, nil
	}
	return typed, err
}
