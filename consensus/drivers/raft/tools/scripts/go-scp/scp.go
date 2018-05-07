package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"time"

	"io"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	//"io/ioutil"
	"errors"
	"flag"

	tml "github.com/BurntSushi/toml"
)

var configPath = flag.String("f", "servers.toml", "configfile")

type ScpInfo struct {
	UserName      string
	PassWord      string
	HostIp        string
	Port          int
	LocalFilePath string
	RemoteDir     string
}

type CmdInfo struct {
	userName  string
	passWord  string
	hostIp    string
	port      int
	cmd       string
	remoteDir string
}

type tomlConfig struct {
	Title   string
	Servers map[string]ScpInfo
}

func sshconnect(user, password, host string, port int) (*ssh.Session, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		session      *ssh.Session
		err          error
	)
	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(password))

	clientConfig = &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: 30 * time.Second,
		//需要验证服务端，不做验证返回nil就可以
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	// connet to ssh
	addr = fmt.Sprintf("%s:%d", host, port)
	if client, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}
	// create session
	if session, err = client.NewSession(); err != nil {
		return nil, err
	}
	return session, nil
}

func sftpconnect(user, password, host string, port int) (*sftp.Client, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		sftpClient   *sftp.Client
		err          error
	)
	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(password))

	clientConfig = &ssh.ClientConfig{
		User:    user,
		Auth:    auth,
		Timeout: 30 * time.Second,
		//需要验证服务端，不做验证返回nil就可以
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	// connet to ssh
	addr = fmt.Sprintf("%s:%d", host, port)

	if sshClient, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}
	// create sftp client
	if sftpClient, err = sftp.NewClient(sshClient); err != nil {
		return nil, err
	}
	return sftpClient, nil
}

func ScpFileFromLocalToRemote(si *ScpInfo) {
	sftpClient, err := sftpconnect(si.UserName, si.PassWord, si.HostIp, si.Port)
	if err != nil {
		fmt.Println("sftconnect have a err!")
		log.Fatal(err)
		panic(err)
	}
	defer sftpClient.Close()
	srcFile, err := os.Open(si.LocalFilePath)
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	defer srcFile.Close()

	var remoteFileName = path.Base(si.LocalFilePath)
	fmt.Println("remoteFileName:", remoteFileName)
	dstFile, err := sftpClient.Create(path.Join(si.RemoteDir, remoteFileName))
	if err != nil {
		log.Fatal(err)
	}
	defer dstFile.Close()
	//bufReader := bufio.NewReader(srcFile)
	//b := bytes.NewBuffer(make([]byte,0))

	buf := make([]byte, 1024000)
	for {
		//n, err := bufReader.Read(buf)
		n, _ := srcFile.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if n == 0 {
			break
		}
		dstFile.Write(buf[0:n])
	}
	fmt.Println("copy file to remote server finished!")
}

func RemoteExec(cmdInfo *CmdInfo) error {
	//A Session only accepts one call to Run, Start or Shell.
	session, err := sshconnect(cmdInfo.userName, cmdInfo.passWord, cmdInfo.hostIp, cmdInfo.port)
	if err != nil {
		return err
	}
	defer session.Close()
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	err = session.Run(cmdInfo.cmd)
	return err
}

func remoteScp(si *ScpInfo, reqnum chan struct{}) {
	defer func() {
		reqnum <- struct{}{}
	}()
	ScpFileFromLocalToRemote(si)
	//session, err := sshconnect("ubuntu", "Fuzamei#123456", "raft15258.chinacloudapp.cn", 22)
	fmt.Println("remoteScp file sucessfully!:")

}

func InitCfg(path string) *tomlConfig {
	var cfg tomlConfig
	if _, err := tml.DecodeFile(path, &cfg); err != nil {
		fmt.Println(err)
		os.Exit(0)

	}
	return &cfg
}

func main() {
	conf := InitCfg(*configPath)
	start := time.Now()
	if len(os.Args) == 1 || os.Args[1] == "-h" {
		LoadHelp()
		return
	}
	argsWithoutProg := os.Args[1:]
	switch argsWithoutProg[0] {
	case "-h": //使用帮助
		LoadHelp()
	case "start":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		if argsWithoutProg[1] == "all" {
			startAll(conf)
		}

	case "stop":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		if argsWithoutProg[1] == "all" {
			stopAll(conf)
		}
	case "clear":
		if len(argsWithoutProg) != 2 {
			fmt.Print(errors.New("参数错误").Error())
			return
		}
		if argsWithoutProg[1] == "all" {
			clearAll(conf)
		}

	}

	////读取当前目录下的文件
	//dir_list, e := ioutil.ReadDir("D:/Repository/src/gitlab.33.cn/chain33/chain33/consensus/drivers/raft/tools/scripts")
	//if e != nil {
	//	fmt.Println("read dir error")
	//	return
	//}
	//for i, v := range dir_list {
	//	fmt.Println(i, "=", v.Name())
	//}

	timeCommon := time.Now()
	log.Printf("read common cost time %v\n", timeCommon.Sub(start))
}

func LoadHelp() {
	fmt.Println("Available Commands:")
	fmt.Println(" start  : 启动服务 ")
	fmt.Println(" stop   : 停止服务")
	fmt.Println(" clear  : 清空数据")
}

func startAll(conf *tomlConfig) {
	//fmt.Println(getCurrentDirectory())
	arrMap := make(map[string]*CmdInfo)
	//多协程启动部署
	reqC := make(chan struct{}, len(conf.Servers))
	for index, sc := range conf.Servers {
		cmdInfo := &CmdInfo{}
		cmdInfo.hostIp = sc.HostIp
		cmdInfo.userName = sc.UserName
		cmdInfo.port = sc.Port
		cmdInfo.passWord = sc.PassWord
		cmdInfo.cmd = fmt.Sprintf("mkdir -p %s", sc.RemoteDir)
		cmdInfo.remoteDir = sc.RemoteDir
		RemoteExec(cmdInfo)
		go remoteScp(&sc, reqC)
		arrMap[index] = cmdInfo
	}
	for i := 0; i < len(conf.Servers); i++ {
		<-reqC
	}
	for i, cmdInfo := range arrMap {
		cmdInfo.cmd = fmt.Sprintf("cd %s;tar -xvf chain33.tgz;bash raft_conf.sh %s;bash run.sh start", cmdInfo.remoteDir, i)
		RemoteExec(cmdInfo)
	}
}

func stopAll(conf *tomlConfig) {
	//执行速度快，不需要多起多协程工作
	for _, sc := range conf.Servers {
		cmdInfo := &CmdInfo{}
		cmdInfo.hostIp = sc.HostIp
		cmdInfo.userName = sc.UserName
		cmdInfo.port = sc.Port
		cmdInfo.passWord = sc.PassWord
		cmdInfo.cmd = fmt.Sprintf("cd %s;bash run.sh stop", sc.RemoteDir)
		cmdInfo.remoteDir = sc.RemoteDir
		RemoteExec(cmdInfo)
	}
}

func clearAll(conf *tomlConfig) {
	for _, sc := range conf.Servers {
		cmdInfo := &CmdInfo{}
		cmdInfo.hostIp = sc.HostIp
		cmdInfo.userName = sc.UserName
		cmdInfo.port = sc.Port
		cmdInfo.passWord = sc.PassWord
		cmdInfo.cmd = fmt.Sprintf("cd %s;bash run.sh clear", sc.RemoteDir)
		cmdInfo.remoteDir = sc.RemoteDir
		RemoteExec(cmdInfo)
	}
}
