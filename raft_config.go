package raft

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Address struct {
	Ip   string `yaml:"ip"`
	Port string `yaml:"port"`
}

type ClussterConfig struct {
	Address Address `yaml:"address"`
}

type ServerConfig struct {
	Id      int     `yaml:"id"`
	Address Address `yaml:"address"`
}

type RaftConfig struct {
	Cluster ClussterConfig `yaml:"cluster"`
	Server  ServerConfig   `yaml:"server"`
}

type Config struct {
	Raft RaftConfig `yaml:"raft"`
}

func NewRaftConfig(path string) (*RaftConfig, error) {
	if len(path) == 0 {
		return nil, errors.New("can not found config file")
	}

	buffer, err := loadFile(path)
	if err != nil {
		return nil, err
	}

	config := new(RaftConfig)
	if err = yaml.Unmarshal(buffer, config); err != nil {
		return nil, nil
	}
	return config, nil
}

func loadFile(path string) (buffer []byte, err error) {
	if buffer, err = os.ReadFile(path); err != nil {
		return nil, err
	}
	return buffer, nil
}
