package main

import (
	"os"
	"strings"

	"go.uber.org/zap/zapcore"
	yaml "gopkg.in/yaml.v3"
)

const (
	_defaultGodocServer = "pkg.go.dev"
	_defaultBranch      = "master"
)

// Config represents the structure of the yaml file
type Config struct {
	URL      string             `yaml:"url"`
	Packages map[string]Package `yaml:"packages"`
	Godoc    GodocConfig        `yaml:"godoc"`
}

func (cfg *Config) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("url", cfg.URL)
	if !cfg.Godoc.empty() {
		enc.AddObject("godoc", &cfg.Godoc)
	}
	return enc.AddObject("packages", packageGroup(cfg.Packages))
}

// GodocConfig is the configuration for the godoc documentation server.
type GodocConfig struct {
	Host string `yaml:"host"`
}

func (gc *GodocConfig) empty() bool {
	return gc.Host == ""
}

func (gc *GodocConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("host", gc.Host)
	return nil
}

type packageGroup map[string]Package

func (ps packageGroup) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for name, p := range ps {
		if err := enc.AddObject(name, p); err != nil {
			return err
		}
	}
	return nil
}

// Package details the options available for each repo
type Package struct {
	Repo   string `yaml:"repo"`
	Branch string `yaml:"branch"`
	URL    string `yaml:"url"`

	Desc string `yaml:"description"` // plain text only
}

func (pkg Package) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("repo", pkg.Repo)
	enc.AddString("branch", pkg.Branch)
	if len(pkg.URL) > 0 {
		enc.AddString("url", pkg.URL)
	}
	if len(pkg.Desc) > 0 {
		enc.AddString("description", pkg.Desc)
	}
	return nil
}

// Parse takes a path to a yaml file and produces a parsed Config
func Parse(path string) (*Config, error) {
	var c Config

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}

	if c.Godoc.Host == "" {
		c.Godoc.Host = _defaultGodocServer
	} else {
		host := c.Godoc.Host
		host = strings.TrimPrefix(host, "https://")
		host = strings.TrimPrefix(host, "http://")
		host = strings.TrimSuffix(host, "/")
		c.Godoc.Host = host
	}

	// set default branch
	for v, p := range c.Packages {
		if p.Branch == "" {
			p.Branch = _defaultBranch
			c.Packages[v] = p
		}
	}

	return &c, err
}
