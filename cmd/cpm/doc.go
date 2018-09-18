package main

/*
chain33 package manager (cpm)主要时用来对chain33项目插件化以后的包依赖管理。
使用步骤（针对Go 1.9.2版本，其他版本可能有差异）：
1.安装Go1.9.2，并设置好相关环境变量
2.安装govendor 命令行输入 go get -u -v github.com/kardianos/govendor
2.从官方源码库中下载chain33源码到GOPATH/src/gitlab.33.cn/chain33/chain33目录下
3.在cpm目录下创建chain33.cpm.toml配置文件，按照以下格式分段填入需要下载的插件包
[blackwhite]
type = "dapp"
gitrepo = "gitlab.33.cn/sanghg/blackwhite"
version=""
其中type字段与具体业务相关，仅支持dapp、store、consensus三个关键字
4.在命令行执行 cpm import命令，执行成功以后会在vender目录下多出添加的插件
 */