package chaincfg

var configMap = make(map[string]string)

func Register(name, cfg string) {
	if _, ok := configMap[name]; ok {
		panic("chain default config name " + name + " is exist")
	}
	configMap[name] = cfg
}

func Load(name string) string {
	return configMap[name]
}

func LoadAll() map[string]string {
	return configMap
}
