package main

/*
本工程主要时辅助用户创建执行器工程的辅助工具，用户可以选择使用简单模式或高级模式来完成对应的创建要求。

高级模式的使用说明：
1. 高级模式的命令行示例
ecg advance -n ProjectName -a ActionName -p ProtobufFile -t TemplateFilePath
2. 参数介绍
	-n 设定输出的执行器包名、执行器中主体逻辑的类名，必填参数
	-a 设定执行器的行为类型名，可以不填，默认为ProjectName+Action
	-p 设定生成执行器动作行为类型的protobuf协议文件，可以不填，默认为 config/ProjectName.proto
	-t 设定生成执行器的模板配置文件路径，可以不填，默认为 template/template
3. 模板文件的设定
	可以根据自己的需要，在模板文件配置中添加各种文件，目前支持对指定标签的文件名和指定标签的文件进行替换
	文件名替换的标签是 ${CLASSNAME}
	文件内容替换的标签是 ${PROJECTNAME} ${CLASSNAME} ${ACTIONNAME}
4. 程序运行时，会先将模板内的文件复制到当前运行目录下 output/ProjectName 文件夹内，然后替换所有标签。
5. 如果复制的文件是go文件，且需要放在chain33目录下，可以在文件名后面增加.tmp，这样程序在复制过程中会自动删除
6. ptypes目录下的
*/
