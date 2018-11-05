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

	// 逻辑中用设定的项目名称替换的文字标签
	TagProjectName = "${PROJECTNAME}"
	// 逻辑中用设定的执行器类名替换的文字标签
	TagClassName = "${CLASSNAME}"
	// 逻辑中用设定的执行器中使用的Action名替换的文字标签
	TagActionName = "${ACTIONNAME}"
)
