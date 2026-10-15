package config

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	AllowedUID  *uint32  `json:"allowed_uid"`
	AllowedSID  string   `json:"allowed_sid,omitempty"`
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
	const zero, dnsPort = 0, "53"
	if !c.validIdentity() {
		return os.ErrInvalid
	}
	if !validSocket(c.Socket) || !filepath.IsAbs(c.StateDir) {
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
