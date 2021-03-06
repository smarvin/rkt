// Copyright 2015 The rkt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/coreos/rkt/tools/common"
)

const (
	// goSeparator is used to separate tuple elements in go list
	// format string.
	goSeparator = "!_##_!"
	// goMakeFunction is a template for generating all files for a
	// given module in a given repo.
	goMakeFunction = "$(shell $(GO_ENV) \"$(DEPSGENTOOL)\" go --repo \"!!!REPO!!!\" --module \"!!!MODULE!!!\" --mode files)"
	goCmd          = "go"
)

type goDepsMode int

const (
	goMakeMode goDepsMode = iota
	goFilesMode
)

func init() {
	cmds[goCmd] = goDeps
}

func goDeps(args []string) string {
	target, repo, module, mode := goGetArgs(args)
	deps := goGetPackageDeps(repo, module)
	switch mode {
	case goMakeMode:
		return GenerateFileDeps(target, goGetMakeFunction(repo, module), deps)
	case goFilesMode:
		return strings.Join(deps, " ")
	}
	panic("Should not happen")
}

// goGetMakeFunction returns a make snippet which will call depsgen go
// with "files" mode.
func goGetMakeFunction(repo, module string) string {
	return replacePlaceholders(goMakeFunction, "REPO", repo, "MODULE", module)
}

// getArgs parses given parameters and returns target, repo, module and
// mode. If mode is "files", then target is optional.
func goGetArgs(args []string) (string, string, string, goDepsMode) {
	f, target := standardFlags(goCmd)
	repo := f.String("repo", "", "Go repo (example: github.com/coreos/rkt)")
	module := f.String("module", "", "Module inside Go repo (example: stage1)")
	mode := f.String("mode", "make", "Mode to use (make - print deps as makefile [default], files - print a list of files)")

	f.Parse(args)
	if *repo == "" {
		common.Die("--repo parameter must be specified and cannot be empty")
	}
	if *module == "" {
		common.Die("--module parameter must be specified and cannot be empty")
	}

	var dMode goDepsMode

	switch *mode {
	case "make":
		dMode = goMakeMode
		if *target == "" {
			common.Die("--target parameter must be specified and cannot be empty when using 'make' mode")
		}
	case "files":
		dMode = goFilesMode
	default:
		common.Die("unknown --mode parameter %q - expected either 'make' or 'files'", *mode)
	}
	return *target, *repo, *module, dMode
}

// goGetPackageDeps returns a list of files that are used to build a
// module in a given repo.
func goGetPackageDeps(repo, module string) []string {
	pkg := path.Join(repo, module)
	deps := []string{pkg}
	for _, d := range goGetDeps(pkg) {
		if strings.HasPrefix(d, repo) {
			deps = append(deps, d)
		}
	}
	return goGetFiles(repo, deps)
}

// goGetDeps gets all dependencies, direct or indirect, of a given
// package.
func goGetDeps(pkg string) []string {
	rawDeps := goRun(goList([]string{"Deps"}, []string{pkg}))
	// we expect only one line
	if len(rawDeps) != 1 {
		return []string{}
	}
	return goSliceRawSlice(rawDeps[0])
}

// goGetFiles returns a list of files that are in given packages. File
// paths are "relative" to passed repo.
func goGetFiles(repo string, pkgs []string) []string {
	params := []string{
		"ImportPath",
		"GoFiles",
		"CgoFiles",
	}
	allFiles := []string{}
	rawTuples := goRun(goList(params, pkgs))
	for _, raw := range rawTuples {
		tuple := goSliceRawTuple(raw)
		module := strings.TrimPrefix(tuple[0], repo+"/")
		files := append(goSliceRawSlice(tuple[1]), goSliceRawSlice(tuple[2])...)
		for i := 0; i < len(files); i++ {
			files[i] = filepath.Join(module, files[i])
		}
		allFiles = append(allFiles, files...)
	}
	return allFiles
}

// goList returns an array of strings describing go list invocation
// with format string consisting all given params separated with
// !_##_! for all given packages.
func goList(params, pkgs []string) []string {
	templateParams := make([]string, 0, len(params))
	for _, p := range params {
		templateParams = append(templateParams, "{{."+p+"}}")
	}
	return append([]string{
		"go",
		"list",
		"-f", strings.Join(templateParams, goSeparator),
	}, pkgs...)
}

// goRun executes given argument list and captures its output. The
// output is sliced into lines with empty lines being discarded.
func goRun(argv []string) []string {
	cmd := exec.Command(argv[0], argv[1:]...)
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		common.Die("Error running %s: %v: %s", strings.Join(argv, " "), err, stderr.String())
	}
	rawLines := strings.Split(stdout.String(), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

// goSliceRawSlice slices given string representation of a slice into
// slice of strings.
func goSliceRawSlice(s string) []string {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	a := strings.Split(s, " ")
	return a
}

// goSliceRawTuple slices given string along !_##_! goSeparator to slice
// of strings. Returned slice might need another round of slicing with
// goSliceRawSlice.
func goSliceRawTuple(t string) []string {
	return strings.Split(t, goSeparator)
}
