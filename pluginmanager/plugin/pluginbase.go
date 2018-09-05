package plugin

type PluginBase struct {

}

func (p *PluginBase) GetPackageName() string {
	return ""
}

func (p *PluginBase) GetExecutorName() string {
	return ""
}

func (p *PluginBase) Init()  {
}