// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/api/filesys"
)

// Cloner is a function that can clone a git repo.
type Cloner func(repoSpec *RepoSpec) error

func ClonerUsingGitExec(repoSpec *RepoSpec) error {
	if repoSpec.Ref == "" {
		repoSpec.Ref = "master"
	}

	gitProgram, err := exec.LookPath("git")
	if err != nil {
		return errors.Wrap(err, "no 'git' program on path")
	}
	gitRootDir, err := filesys.GitRootDir()

	if err != nil {
		return err
	}

	_ = os.Mkdir(gitRootDir.String(), 0755 | os.ModeDir)

	repoFolderName := strings.ReplaceAll(fmt.Sprintf("%s_%s_%s", repoSpec.Host, repoSpec.OrgRepo, repoSpec.Ref), "/", "_")
	repoSpec.Dir = filesys.ConfirmedDir(gitRootDir.Join(repoFolderName))

	log.Printf("visited git repo: %s", repoFolderName)

	if _, err := os.Stat(repoSpec.Dir.String()); os.IsNotExist(err) {
		tmpDir, err := filesys.NewTmpConfirmedDir()
		if err != nil {
			return err
		}

		//first clone
		//git clone -b branch --single-branch git@HOST:REPO.git
		cmd := exec.Command(
			gitProgram,
			"clone",
			"--depth",
			"1",
			"-b",
			repoSpec.Ref,
			"--single-branch",
			repoSpec.CloneSpec(),
			".")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		cmd.Dir = tmpDir.String()
		err = cmd.Run()
		if err != nil {
			log.Printf("Error setting git remote: %s", out.String())
			return errors.Wrapf(
				err,
				"trouble adding remote %s",
				repoSpec.CloneSpec())
		}
		err = os.Rename(tmpDir.String(), repoSpec.Dir.String())
		if err != nil {
			os.RemoveAll(tmpDir.String())
		}
	} else {
		//dir already exists
		//TODO maybe pull updates
	}

	return nil
}

// ClonerUsingGitExec uses a local git install, as opposed
// to say, some remote API, to obtain a local clone of
// a remote repo.
func ClonerUsingGitExecV1(repoSpec *RepoSpec) error {
	gitProgram, err := exec.LookPath("git")
	if err != nil {
		return errors.Wrap(err, "no 'git' program on path")
	}
	repoSpec.Dir, err = filesys.NewTmpConfirmedDir()
	if err != nil {
		return err
	}
	cmd := exec.Command(
		gitProgram,
		"init",
		repoSpec.Dir.String())
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	if err != nil {
		log.Printf("Error initializing empty git repo: %s", out.String())
		return errors.Wrapf(
			err,
			"trouble initializing empty git repo in %s",
			repoSpec.Dir.String())
	}

	cmd = exec.Command(
		gitProgram,
		"remote",
		"add",
		"origin",
		repoSpec.CloneSpec())
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Dir = repoSpec.Dir.String()
	err = cmd.Run()
	if err != nil {
		log.Printf("Error setting git remote: %s", out.String())
		return errors.Wrapf(
			err,
			"trouble adding remote %s",
			repoSpec.CloneSpec())
	}
	if repoSpec.Ref == "" {
		repoSpec.Ref = "master"
	}
	cmd = exec.Command(
		gitProgram,
		"fetch",
		"--depth=1",
		"origin",
		repoSpec.Ref)
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Dir = repoSpec.Dir.String()
	err = cmd.Run()
	if err != nil {
		cmd = exec.Command(
			gitProgram,
			"pull",
			"origin",
			"master")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Dir = repoSpec.Dir.String()
		err := cmd.Run()
		if err != nil {
			return errors.Wrapf(err, "trouble pulling %s", repoSpec.OrgRepo)
		}
		if repoSpec.Ref == "" {
			repoSpec.Ref = "master"
		}
		cmd = exec.Command(gitProgram, "checkout", repoSpec.Ref)
		cmd.Dir = repoSpec.Dir.String()
		err = cmd.Run()
		if err != nil {
			return errors.Wrapf(
				err, "trouble checking out href %s", repoSpec.Ref)
		}
	}

	cmd = exec.Command(
		gitProgram,
		"reset",
		"--hard",
		"FETCH_HEAD")
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Dir = repoSpec.Dir.String()
	err = cmd.Run()
	if err != nil {
		log.Printf("Error performing git reset: %s", out.String())
		return errors.Wrapf(
			err, "trouble hard resetting empty repository to %s", repoSpec.Ref)
	}

	cmd = exec.Command(
		gitProgram,
		"submodule",
		"update",
		"--init",
		"--recursive")
	cmd.Stdout = &out
	cmd.Dir = repoSpec.Dir.String()
	err = cmd.Run()
	if err != nil {
		return errors.Wrapf(err, "trouble fetching submodules for %s", repoSpec.Ref)
	}

	return nil
}

// DoNothingCloner returns a cloner that only sets
// cloneDir field in the repoSpec.  It's assumed that
// the cloneDir is associated with some fake filesystem
// used in a test.
func DoNothingCloner(dir filesys.ConfirmedDir) Cloner {
	return func(rs *RepoSpec) error {
		rs.Dir = dir
		return nil
	}
}
