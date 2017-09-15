/*
 *
 * k6 - a next-generation load testing tool
 * Copyright (C) 2016 Load Impact
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package cmd

import (
	"bytes"
	"io"
	"sync"

	"github.com/loadimpact/k6/lib"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	null "gopkg.in/guregu/null.v3"
)

// Panic if the given error is not nil.
func must(err error) {
	if err != nil {
		panic(err)
	}
}

// Silently set an exit code.
type ExitCode struct {
	error
	Code int
}

// A writer that syncs writes with a mutex and, if the output is a TTY, clears before newlines.
type consoleWriter struct {
	Writer io.Writer
	IsTTY  bool
	Mutex  *sync.Mutex
}

func (w consoleWriter) Write(p []byte) (n int, err error) {
	if w.IsTTY {
		p = bytes.Replace(p, []byte{'\n'}, []byte{'\x1b', '[', '0', 'K', '\n'}, -1)
	}
	w.Mutex.Lock()
	n, err = w.Writer.Write(p)
	w.Mutex.Unlock()
	return
}

func registerOptions(flags *pflag.FlagSet) {
	flags.Int64P("vus", "u", 1, "number of virtual users")
	flags.Int64P("max", "m", 0, "max available virtual users")
	flags.DurationP("duration", "d", 0, "test duration limit")
	flags.Int64P("iterations", "i", 0, "script iteration limit")
	flags.StringSliceP("stage", "s", nil, "add a `stage`, as `[duration]:[target]`")
	flags.BoolP("paused", "p", false, "start the test in a paused state")
	flags.Int64("max-redirects", 10, "follow at most n redirects")
	flags.String("user-agent", "", "user agent for http requests")
	flags.Bool("insecure-skip-tls-verify", false, "skip verification of TLS certificates")
	flags.Bool("no-connection-reuse", false, "don't reuse connections between iterations")
	flags.BoolP("throw", "w", false, "throw warnings (like failed http requests) as errors")
}

func getNullBool(flags *pflag.FlagSet, key string) null.Bool {
	v, err := flags.GetBool(key)
	if err != nil {
		panic(err)
	}
	return null.NewBool(v, flags.Changed(key))
}

func getNullInt64(flags *pflag.FlagSet, key string) null.Int {
	v, err := flags.GetInt64(key)
	if err != nil {
		panic(err)
	}
	return null.NewInt(v, flags.Changed(key))
}

func getNullDuration(flags *pflag.FlagSet, key string) lib.NullDuration {
	v, err := flags.GetDuration(key)
	if err != nil {
		panic(err)
	}
	return lib.NullDuration{Duration: lib.Duration(v), Valid: flags.Changed(key)}
}

func getNullString(flags *pflag.FlagSet, key string) null.String {
	v, err := flags.GetString(key)
	if err != nil {
		panic(err)
	}
	return null.NewString(v, flags.Changed(key))
}

func getOptions(flags *pflag.FlagSet) (lib.Options, error) {
	opts := lib.Options{
		VUs:                   getNullInt64(flags, "vus"),
		VUsMax:                getNullInt64(flags, "max"),
		Duration:              getNullDuration(flags, "duration"),
		Iterations:            getNullInt64(flags, "iterations"),
		Paused:                getNullBool(flags, "paused"),
		MaxRedirects:          getNullInt64(flags, "max-redirects"),
		UserAgent:             getNullString(flags, "user-agent"),
		InsecureSkipTLSVerify: getNullBool(flags, "insecure-skip-tls-verify"),
		NoConnectionReuse:     getNullBool(flags, "no-connection-reuse"),
		Throw:                 getNullBool(flags, "throw"),
	}

	stageStrings, err := flags.GetStringSlice("stage")
	if err != nil {
		return opts, err
	}
	if len(stageStrings) > 0 {
		opts.Stages = make([]lib.Stage, len(stageStrings))
		for i, s := range stageStrings {
			var stage lib.Stage
			if err := stage.UnmarshalText([]byte(s)); err != nil {
				return opts, errors.Wrapf(err, "stage %d", i)
			}
			opts.Stages[i] = stage
		}
	}
	return opts, nil
}
