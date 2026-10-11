package config

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	AllowedUID  *uint32  `json:"allowed_uid"`
	ForwardUser string   `json:"forward_user"`
	Zone        string   `json:"zone"`
	Socket      string   `json:"socket"`
	StateDir    string   `json:"state_dir"`
	Upstreams   []string `json:"upstreams"`
	DoH         []string `json:"doh_addresses"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var result Config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return result, err
	}
	var extra Config
	if err := decoder.Decode(&extra); err != io.EOF {
		return result, os.ErrInvalid
	}
	return result, result.Validate()
}

func (c Config) Validate() error {
	const empty, zero, dnsPort = "", 0, "53"
	if c.AllowedUID == nil || *c.AllowedUID == zero || c.ForwardUser == empty {
		return os.ErrInvalid
	}
	if !filepath.IsAbs(c.Socket) || !filepath.IsAbs(c.StateDir) {
		return os.ErrInvalid
	}
	if _, err := time.LoadLocation(c.Zone); err != nil {
		return err
	}
	if len(c.Upstreams) == zero || len(c.DoH) == zero {
		return os.ErrInvalid
	}
	for _, upstream := range c.Upstreams {
		host, port, err := net.SplitHostPort(upstream)
		ip := net.ParseIP(host)
		if err != nil || ip == nil || ip.IsLoopback() || ip.IsUnspecified() || port != dnsPort {
			return os.ErrInvalid
		}
	}
	for _, address := range c.DoH {
		if net.ParseIP(address) == nil {
			return os.ErrInvalid
		}
	}
	return nil
}

func Demo() Config {
	const prefix, socket, state, zone = "saveme-demo-", "saveme.sock", "state", "UTC"
	uid := uint32(os.Getuid())
	dir := filepath.Join(os.TempDir(), prefix+strconv.Itoa(int(uid)))
	return Config{AllowedUID: &uid, Socket: filepath.Join(dir, socket), StateDir: filepath.Join(dir, state), Zone: zone}
}
