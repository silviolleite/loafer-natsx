//go:build test

package testsetup

import deadlock "github.com/sasha-s/go-deadlock"

func init() {
	deadlock.Opts.Disable = false
}
