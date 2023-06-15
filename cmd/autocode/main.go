package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	packageNameFlag     string
	structNameFlag      string
	varNameFlag         string
	interfaceNameFlag   string
	repoNameFlag        string
	serviceFilePathFlag string
	apiFilePathFlag     string
	serviceTemplatePath string
	apiTemplatePath     string
)

var rootCmd = &cobra.Command{
	Use:   "codegen",
	Short: "Generate code based on templates",
	Run:   runCodeGeneration,
}

func main() {
	addFlags()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func addFlags() {
	rootCmd.Flags().StringVarP(&serviceTemplatePath, "service-template", "s", "", "Service template file path")
	rootCmd.Flags().StringVarP(&apiTemplatePath, "api-template", "a", "", "API template file path")

	rootCmd.MarkFlagRequired("service-template")
	rootCmd.MarkFlagRequired("api-template")
}

func runCodeGeneration(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)

	// 读取 Service 模板文件内容
	serviceTemplate, err := ioutil.ReadFile(serviceTemplatePath)
	if err != nil {
		fmt.Println("读取 Service 模板文件失败:", err)
		return
	}

	// 读取 Api 模板文件内容
	apiTemplate, err := ioutil.ReadFile(apiTemplatePath)
	if err != nil {
		fmt.Println("读取 Api 模板文件失败:", err)
		return
	}

	// 读取包名
	fmt.Print("请输入包名：")
	packageName, _ := reader.ReadString('\n')
	packageName = strings.TrimSpace(packageName)

	// 读取结构体名
	fmt.Print("请输入结构体名：")
	structName, _ := reader.ReadString('\n')
	structName = strings.TrimSpace(structName)

	// 读取变量名
	fmt.Print("请输入变量名：")
	varName, _ := reader.ReadString('\n')
	varName = strings.TrimSpace(varName)

	// 读取接口名
	fmt.Print("请输入接口名：")
	interfaceName, _ := reader.ReadString('\n')
	interfaceName = strings.TrimSpace(interfaceName)

	// 读取存储库名
	fmt.Print("请输入存储库名：")
	repoName, _ := reader.ReadString('\n')
	repoName = strings.TrimSpace(repoName)

	// 读取生成的 Service 代码文件路径
	fmt.Print("请输入生成的 Service 代码文件路径：")
	serviceFilePath, _ := reader.ReadString('\n')
	serviceFilePath = strings.TrimSpace(serviceFilePath)

	// 读取生成的 API 代码文件路径
	fmt.Print("请输入生成的 API 代码文件路径：")
	apiFilePath, _ := reader.ReadString('\n')
	apiFilePath = strings.TrimSpace(apiFilePath)

	// 定义字段和值
	fields := map[string]string{
		"PackageName":   packageName,
		"StructName":    structName,
		"VarName":       varName,
		"InterfaceName": interfaceName,
		"RepoName":      repoName,
	}

	// 替换 Service 模板中的字段
	serviceResult := replaceFields(serviceTemplate, fields)

	// 替换 API 模板中的字段
	apiResult := replaceFields(apiTemplate, fields)

	// 写入生成的代码文件
	if err := writeCodeToFile(serviceResult, serviceFilePath); err != nil {
		fmt.Println("写入 Service 代码文件失败:", err)
		return
	}
	if err := writeCodeToFile(apiResult, apiFilePath); err != nil {
		fmt.Println("写入 API 代码文件失败:", err)
		return
	}

	fmt.Println("代码生成成功！")
}

func replaceFields(template []byte, fields map[string]string) string {
	result := string(template)
	for field, value := range fields {
		result = strings.ReplaceAll(result, fmt.Sprintf("{{%s}}", field), value)
	}
	return result
}

func writeCodeToFile(code string, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return err
			}
		}
	}

	defer file.Close()

	_, err = file.WriteString(code)
	if err != nil {
		return err
	}

	return nil
}
