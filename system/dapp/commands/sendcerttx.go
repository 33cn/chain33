package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// OneStepSendCertTxCmd send cmd
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
	cmd.Flags().StringP("keyFilePath", "k", "", "private key file path")
	cmd.Flags().StringP("certFilePath", "c", "", "cert file path")
	return cmd
}

// oneStepSendCertTx send cmd
func oneStepSendCertTx(cmd *cobra.Command, cmdName string, params []string) {
	if len(params) < 1 || params[0] == "help" || params[0] == "--help" || params[0] == "-h" {
		loadSendCertTxHelp()
		return
	}

	var createParams, keyParams []string
	//取出sendcerttx命令的signType,keyFilePath,certFilePath参数, 保留原始构建的参数列表
	for _, v := range params {
		if strings.HasPrefix(v, "-s=") || strings.HasPrefix(v, "--signType=") || strings.HasPrefix(v, "-k=") || strings.HasPrefix(v, "--keyFilePath=") || strings.HasPrefix(v, "-c=") || strings.HasPrefix(v, "--certFilePath=") {
			keyParams = append(keyParams, v)
		} else {
			createParams = append(createParams, v)
		}
	}
	//调用send命令parse函数解析key参数
	cmd.Flags().Parse(keyParams)
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
	//交易签名
	signParams := []string{"wallet", "signWithCert", "-d", createRes, "-s", signType, "-k", keyFilePath, "-c", certFilePath}
	cmdSign := exec.Command(cmdName, append(signParams, keyParams...)...)
	signRes, err := execCmd(cmdSign)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	//交易发送命令调用
	sendParams := []string{"wallet", "send", "-d", signRes, "--rpc_laddr", rpcAddr}
	cmdSend := exec.Command(cmdName, sendParams...)
	sendRes, err := execCmd(cmdSend)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(sendRes)
}

// loadSendCertTxHelp
func loadSendCertTxHelp() {
	help := `[Integrate create/sign/send transaction operations in one command]
Usage:
  -cli sendcerttx [flags]

Examples:
cli sendcerttx coins transfer -a 1 -n note -t toAddr  -s=signType -k=keyFilePath -c=certFilePath

equivalent to three steps: 
1. cli coins transfer -a 1 -n note -t toAddr                                        //create raw tx
2. cli wallet signWithCert -d rawTx -s signType -k keyFilePath -c certFilePath      //sign raw tx
3. cli wallet send -d signTx                                                        //send tx to block chain

Flags:
  -h, --help                     help for send
  -s, --signType                 sign type
  -k, --keyFilePath			     private key file path
  -c, --certFilePaht             cert file path`
	fmt.Println(help)
}
