// Package clock abstracts time so logic that depends on it can be tested.
//
// Code takes a Clock instead of calling time.Now or time.AfterFunc directly.
// Production code passes Real(); tests pass a Fake and move time forward with
// Advance.
package clock
