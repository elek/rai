package config

type Config struct {
	Providers []Provider `yaml:"providers"`
	Models    []Model    `yaml:"models"`
}

func (c Config) FindProvider(name string) (Provider, bool) {
	for _, p := range c.Providers {
		if p.Name == name {
			return p, true
		}
	}
	return Provider{}, false
}

func (c Config) FindModel(name string) (Model, bool) {
	for _, m := range c.Models {
		if m.Name == name {
			return m, true
		}
	}
	return Model{}, false
}

func (c Config) FindDefaultModel() (Model, bool) {
	for _, m := range c.Models {
		if m.Default {
			return m, true
		}
	}
	return Model{}, false
}

type Provider struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Key  string `yaml:"key"`
}

type Model struct {
	Name        string  `yaml:"name"`
	Provider    string  `yaml:"provider"`
	Model       string  `yaml:"model"`
	MaxToken    int     `yaml:"max_token"`
	Debug       bool    `yaml:"debug"`
	Temperature float64 `yaml:"temperature"`
	Default     bool    `yaml:"default"`
}
