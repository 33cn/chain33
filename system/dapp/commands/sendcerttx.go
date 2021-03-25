package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"strings"
)

func OneStepSendCertTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "sendcerttx",
		Short:              "send cert tx in one step",
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			oneStepSendCertTx(cmd, os.Args[0], args)
		},
	}
	cmd.Flags().StringP("signType", "s", "sm2", "sign type")
	cmd.Flags().StringP("keyFilePath", "key", "", "private key file path")
	cmd.Flags().StringP("certFilePath", "cert", "", "cert file path")
	//cmd.MarkFlagRequired("key")
	return cmd
}

// one step send
func oneStepSendCertTx(cmd *cobra.Command, cmdName string, params []string) {
	if len(params) < 1 || params[0] == "help" || params[0] == "--help" || params[0] == "-h" {
		loadSendCertTxHelp()
		return
	}

	var createParams, keyParams []string
	//取出sendcerttx命令的signType,keyFilePath,certFilePath参数, 保留原始构建的参数列表
	for _, v := range params {
		if strings.HasPrefix(v, "-s=") || strings.HasPrefix(v, "signType") || strings.HasPrefix(v, "-key=") || strings.HasPrefix(v, "--keyFilePath=") || strings.HasPrefix(v, "-cert=") || strings.HasPrefix(v, "certFilePath") {
			keyParams = append(keyParams, v)
		} else {
			createParams = append(createParams, v)
		}
	}
	//调用send命令parse函数解析key参数
	err := cmd.Flags().Parse(keyParams)
	keyFilePath, _ := cmd.Flags().GetString("keyFilePath")
	certFilePath, _ := cmd.Flags().GetString("certFilePath")
	signType, _ := cmd.Flags().GetString("signType")
	//创建交易命令
	cmdCreate := exec.Command(cmdName, createParams...)
	createRes, err := execCmd(cmdCreate)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	//cli不传任何参数不会报错, 输出帮助信息
	if strings.Contains(createRes, "\n") {
		fmt.Println(createRes)
		return
	}
	//采用内部的构造交易命令,解析rpc_laddr地址参数
	createCmd, createFlags, _ := cmd.Root().Traverse(createParams)
	_ = createCmd.ParseFlags(createFlags)
	rpcAddr, _ := createCmd.Flags().GetString("rpc_laddr")
	//交易签名和发送
	signParams := []string{"wallet", "signWithCert", "-d", createRes, "-s", signType, "-k", keyFilePath, "-c", certFilePath, "--rpc_laddr", rpcAddr}
	cmdSign := exec.Command(cmdName, append(signParams, keyParams...)...)
	sendRes, err := execCmd(cmdSign)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(sendRes)
}

func loadSendCertTxHelp() {
	help := `[Integrate create/sign/send transaction operations in one command]
Usage:
  -cli sendcerttx [flags]

Examples:
cli send coins transfer -a 1 -n note -t toAddr  -s signType -key=keyFilePath -cert=certFilePath

equivalent to three steps: 
1. cli coins transfer -a 1 -n note -t toAddr   //create raw tx
2. cli wallet signWithCert -d rawTx -s signType -k keyFilePath -c certFilePath      //sign raw tx


Flags:
  -h, --help         help for send
  -s, --signType     sign type
  -key, --keyFilePath			 private key file path
  -cert, --certFilePaht       cert file path`
	fmt.Println(help)
}
