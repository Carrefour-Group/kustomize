// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/internal/utils"
)

// gitRunner runs the external git binary.
type gitRunner struct {
	gitProgram string
	duration   time.Duration
	dir        filesys.ConfirmedDir
}

// newCmdRunner returns a gitRunner if it can find the binary.
// It also creats a temp directory for cloning repos.
func newCmdRunner(timeout time.Duration) (*gitRunner, error) {
	gitProgram, err := exec.LookPath("git")
	if err != nil {
		return nil, errors.Wrap(err, "no 'git' program on path")
	}
	dir, err := filesys.NewTmpConfirmedDir()
	if err != nil {
		return nil, err
	}
	return &gitRunner{
		gitProgram: gitProgram,
		duration:   timeout,
		dir:        dir,
	}, nil
}

// run a command with a timeout.
func (r gitRunner) run(args ...string) (string, error) {
	output := []byte{}
	//nolint: gosec
	cmd := exec.Command(r.gitProgram, args...)
	cmd.Dir = r.dir.String()

	err := utils.TimedCall(
		cmd.String(),
		r.duration,
		func() error {
			o, err := cmd.CombinedOutput()
			if err != nil {
				return errors.Wrapf(err, "git cmd = '%s'", cmd.String())
			}
			output = o
			return err
		})
	return string(output), err
}
