package main

import (
	"context"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"

	"saveme/internal/config"
)

func forwardIdentity(cfg config.Config) (uint32, uint32, error) {
	const bits, base, zero = 32, 10, 0
	account, err := user.Lookup(cfg.ForwardUser)
	if err != nil {
		return zero, zero, err
	}
	uid, err := strconv.ParseUint(account.Uid, base, bits)
	if err != nil || uid == zero || uint32(uid) == *cfg.AllowedUID {
		return zero, zero, os.ErrInvalid
	}
	gid, err := strconv.ParseUint(account.Gid, base, bits)
	if err != nil {
		return zero, zero, err
	}
	return uint32(uid), uint32(gid), nil
}

func startForwarder(ctx context.Context, cfg config.Config, uid, gid uint32) (*exec.Cmd, error) {
	const flag, upstreamFlag, delimiter = "-forwarder", "-upstreams", ","
	const path, zone = "PATH=/usr/sbin:/usr/bin:/sbin:/bin", "TZ=UTC"
	pathToBinary, err := os.Executable()
	if err != nil {
		return nil, err
	}
	child := exec.CommandContext(ctx, pathToBinary, flag, upstreamFlag, strings.Join(cfg.Upstreams, delimiter))
	child.Env = []string{path, zone}
	child.Stdout, child.Stderr = os.Stdout, os.Stderr
	child.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uid, Gid: gid}}
	if err := child.Start(); err != nil {
		return nil, err
	}
	return child, nil
}
