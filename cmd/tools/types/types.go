package types

const (
	// 通过使用 go vendor 工具，从三方代码库中下载对应的插件代码
	KeyImportPackage            = "import_package"
	KeyCreateSimpleExecProject  = "create_simple_project"
	KeyCreateAdvanceExecProject = "create_advance_project"
	KeyConfigFolder             = "config_folder"
	KeyProjectName              = "project_name"
	KeyClassName                = "class_name"
	KeyExecutorName             = "executor_name"
	KeyActionName               = "action_name"
	KeyProtobufFile             = "protobuf_file"
	KeyTemplateFilePath         = "template_file_path"
	KeyUpdateInit               = "update_init"

	DEF_CPM_CONFIGFILE = "chain33.cpm.toml"

	TagProjectName     = "${PROJECTNAME}" // 项目名称替换标签
	TagClassName       = "${CLASSNAME}"   // 主类类名替换标签
	TagActionName      = "${ACTIONNAME}"  // 执行器中使用的Action替换标签
	TagExecName        = "${EXECNAME}"    // 执行器名称替换标签
	TagExecNameFB      = "${EXECNAME_FB}" // 首字母大写的执行器名称替换标签
	TagTyLogActionType = "${TYLOGACTIONTYPE}"
	TagActionIDText    = "${ACTIONIDTEXT}"
	TagLogMapText      = "${LOGMAPTEXT}"
	TagTypeMapText     = "${TYPEMAPTEXT}"
	TagTypeName        = "${TYPENAME}"
)
