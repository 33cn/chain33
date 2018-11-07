package types

// manager action
const (
	ManageActionModifyConfig = iota
)

const (
	// log for config
	TyLogModifyConfig = 410
)

// config items
const (
	ConfigItemArrayConfig = iota
	ConfigItemIntConfig
	ConfigItemStringConfig
)
