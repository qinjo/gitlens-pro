package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	insertCode = `e={user:{id:"88888888-8888-8888-8888-888888888888",name:"Neo",email:"x@x.com",status:"activated",createdDate:"2000-01-01T00:00:00.000Z"},licenses:{paidLicenses:{},effectiveLicenses:{"gitlens-pro":{organizationId:"Linux",latestStatus:"active",latestStartDate:"2024-01-01",latestEndDate:"2999-01-01",reactivationCount:99,nextOptInDate:"2999-01-01"}}},nextOptInDate:"2999-01-01"};`
)

type ExtensionPath struct {
	Name        string
	Path        string
	Description string
}

// 预设的 VSCode 扩展目录
var predefinedPaths = []ExtensionPath{
	{
		Name:        "VSCode",
		Path:        filepath.Join(".vscode", "extensions"),
		Description: "标准 VSCode",
	},
	{
		Name:        "VSCode Server",
		Path:        filepath.Join(".vscode-server", "extensions"),
		Description: "VSCode 服务端",
	},
	{
		Name:        "VSCode Insiders",
		Path:        filepath.Join(".vscode-insiders", "extensions"),
		Description: "VSCode 预览版",
	},
	{
		Name:        "Cursor",
		Path:        filepath.Join(".cursor", "extensions"),
		Description: "Cursor 编辑器",
	},
	{
		Name:        "Windsurf",
		Path:        filepath.Join(".windsurf", "extensions"),
		Description: "Windsurf 编辑器",
	},
	{
		Name:        "Kiro",
		Path:        filepath.Join(".kiro", "extensions"),
		Description: "Kiro 编辑器",
	},
	{
		Name:        "Antigravity",
		Path:        filepath.Join(".antigravity", "extensions"),
		Description: "Antigravity 编辑器",
	},
	{
		Name:        "Trae",
		Path:        filepath.Join(".trae", "extensions"),
		Description: "Trae 编辑器",
	},
}

func getExtensionsDir() string {
	// 优先使用命令行参数
	if len(os.Args) > 2 && os.Args[1] == "--ext-dir" {
		return os.Args[2]
	}

	// 其次使用环境变量
	if envDir := os.Getenv("VSCODE_EXTENSIONS_DIR"); envDir != "" {
		return envDir
	}

	// 获取用户主目录
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("警告: 无法获取用户主目录")
		home = "."
	}

	// 显示预设选项
	fmt.Println("\n请选择 VSCode 扩展目录:")
	for i, path := range predefinedPaths {
		fullPath := filepath.Join(home, path.Path)
		fmt.Printf("[%d] %s (%s)\n    路径: %s\n", i+1, path.Name, path.Description, fullPath)
	}
	fmt.Printf("[%d] 自定义路径\n", len(predefinedPaths)+1)

	// 获取用户选择
	choice := promptForSelection(len(predefinedPaths)+1, "请选择扩展目录")

	// 如果选择自定义路径
	if choice == len(predefinedPaths)+1 {
		fmt.Print("\n请输入自定义扩展目录路径: ")
		reader := bufio.NewReader(os.Stdin)
		customPath, _ := reader.ReadString('\n')
		return strings.TrimSpace(customPath)
	}

	// 返回选择的预设路径
	selectedPath := predefinedPaths[choice-1]
	return filepath.Join(home, selectedPath.Path)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "restore" {
		if err := Restore(); err != nil {
			fmt.Printf("恢复失败: %v\n", err)
			waitForKeyPress()
			return
		}
		fmt.Println("恢复完成! 请重启 VS Code。")
		waitForKeyPress()
		return
	}
	// 获取扩展目录
	extensionsDir := getExtensionsDir()

	fmt.Println(extensionsDir)

	// 获取最新版本的 GitLens 目录
	extensionPath, err := getLatestGitLensPath(extensionsDir)
	if err != nil {
		fmt.Printf("查找 GitLens 扩展失败: %v\n", err)
		waitForKeyPress()
		return
	}

	// 获取目录名以检查版本
	dirName := filepath.Base(extensionPath)

	// 需要修改的文件列表
	filesToModify := []string{
		filepath.Join(extensionPath, "dist", "gitlens.js"),
		filepath.Join(extensionPath, "dist", "browser", "gitlens.js"),
	}

	// 如果是版本17及以上，添加graph.js文件
	if isVersion17OrAbove(dirName) {
		filesToModify = append(filesToModify, filepath.Join(extensionPath, "dist", "webviews", "graph.js"))
	}

	// 处理每个文件
	for _, file := range filesToModify {
		if err := processFile(file); err != nil {
			fmt.Printf("处理文件 %s 失败: %v\n", file, err)
			continue
		}
	}

	fmt.Println("\n激活完成! 请重启 VS Code 以使更改生效。")
	waitForKeyPress()
}

// 等待用户按键
func waitForKeyPress() {
	fmt.Print("\n按回车键退出...")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}

func getLatestGitLensPath(extensionsDir string) (string, error) {
	entries, err := os.ReadDir(extensionsDir)
	if err != nil {
		return "", fmt.Errorf("读取扩展目录失败: %v", err)
	}

	var gitLensDirs []string
	pattern := regexp.MustCompile(`^eamodio\.gitlens-\d+\.\d+\.\d+(?:-universal)?$`)

	for _, entry := range entries {
		if entry.IsDir() && pattern.MatchString(entry.Name()) {
			gitLensDirs = append(gitLensDirs, entry.Name())
		}
	}

	if len(gitLensDirs) == 0 {
		return "", fmt.Errorf("未找到 GitLens 扩展")
	}

	// 按版本号排序
	sort.Sort(sort.Reverse(sort.StringSlice(gitLensDirs)))

	// 如果只有一个版本，直接返回
	if len(gitLensDirs) == 1 {
		return filepath.Join(extensionsDir, gitLensDirs[0]), nil
	}

	// 显示可用版本供用户选择
	fmt.Println("\n发现多个 GitLens 版本:")
	for i, dir := range gitLensDirs {
		fmt.Printf("[%d] %s\n", i+1, dir)
	}

	selectedVersion := promptForSelection(len(gitLensDirs), "请选择要激活的 GitLens 版本")
	return filepath.Join(extensionsDir, gitLensDirs[selectedVersion-1]), nil
}

// 提示用户选择
func promptForSelection(maxChoice int, prompt string) int {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("\n%s (输入数字): ", prompt)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		choice, err := strconv.Atoi(input)
		if err == nil && choice > 0 && choice <= maxChoice {
			return choice
		}
		fmt.Printf("无效的选择，请输入 1 到 %d 之间的数字\n", maxChoice)
	}
}

// 添加版本检测函数
func isVersion15(dirName string) bool {
	pattern := regexp.MustCompile(`^eamodio\.gitlens-15\.\d+\.\d+(?:-universal)?$`)
	return pattern.MatchString(dirName)
}

func isVersion17OrAbove(dirName string) bool {
	pattern := regexp.MustCompile(`^eamodio\.gitlens-(17|18)\.\d+\.\d+(?:-universal)?$`)
	return pattern.MatchString(dirName)
}

// 修改 processFile 函数
func processFile(filePath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", filePath)
	}

	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %v", err)
	}

	// 创建备份
	backupPath := filePath + ".backup"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		if err := os.WriteFile(backupPath, content, 0644); err != nil {
			return fmt.Errorf("创建备份失败: %v", err)
		}
		fmt.Printf("已创建备份: %s\n", backupPath)
	}

	// 获取目录名以检查版本
	dirName := getExtensionDirName(filePath)

	if isVersion15(dirName) {
		return processVersion15File(filePath, content)
	}

	if isVersion17OrAbove(dirName) {
		return processVersion17File(filePath, content)
	}

	// 默认使用版本16的处理方式
	return processVersion16File(filePath, content)
}

// 处理 15.x 版本的文件
func processVersion15File(filePath string, content []byte) error {
	contentStr := string(content)

	// 替换模式（将无序的map改成有序的切片，避免先替换"qn.Community"导致"qn.CommunityWithAccount"被意外替换）
	replacements := []struct{ old, new string }{
		{"qn.CommunityWithAccount", "qn.Enterprise"},
		{"qn.Community", "qn.Enterprise"},
		{"qn.Pro", "qn.Enterprise"},
	}

	modified := false
	for _, replacement := range replacements {
		if strings.Contains(contentStr, replacement.old) {
			contentStr = strings.ReplaceAll(contentStr, replacement.old, replacement.new)
			modified = true
			fmt.Printf("替换 %s 为 %s\n", replacement.old, replacement.new)
		}
	}

	if !modified {
		return fmt.Errorf("未找到需要替换的内容")
	}

	// 写入修改后的内容
	if err := os.WriteFile(filePath, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fmt.Printf("成功修改文件: %s\n", filePath)
	return nil
}

// 处理 16.x 版本的文件
func processVersion16File(filePath string, content []byte) error {
	// 查找匹配模式 - 支持多字母变量名（如 let a,b,c={id:e.user.id,name:）
	pattern := regexp.MustCompile(`let ([a-zA-Z,]+)=\{id:e\.user\.id,name:e\.user\.name`)
	matches := pattern.FindStringSubmatch(string(content))

	if len(matches) < 2 {
		return fmt.Errorf("未找到匹配模式")
	}

	matchedVariables := matches[1]
	exactMatch := fmt.Sprintf("let %s={id:e.user.id,name:e.user.name", matchedVariables)

	// 注入激活代码
	newContent := strings.Replace(string(content), exactMatch, insertCode+exactMatch, 1)

	// 写入修改后的内容
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fmt.Printf("成功修改文件: %s\n", filePath)
	fmt.Printf("匹配的变量: %s\n", matchedVariables)

	return nil
}

// 处理 17.x 及 18.x 版本的文件
func processVersion17File(filePath string, content []byte) error {
	fmt.Printf("处理版本17/18的文件: %s\n", filePath)

	// 如果是graph.js文件，只执行版本17/18的处理
	if strings.Contains(filePath, "webviews") && strings.Contains(filePath, "graph.js") {
		return processVersion17GraphFile(filePath, content)
	}

	// 其他文件（gitlens.js）执行版本16的处理
	return processVersion16File(filePath, content)
}

// 处理 17.x 及 18.x 版本的graph.js文件
func processVersion17GraphFile(filePath string, content []byte) error {
	contentStr := string(content)

	// 查找匹配模式 - 对应JavaScript中的 /(\(|=)(this\.(graphS|s)tate\.allowed)/g
	// v17 的 gate 遮罩用 ?hidden=${!1!==this.state.allowed} 绑定，通过取反 this.state.allowed 来隐藏遮罩
	pattern := regexp.MustCompile(`(\(|=)(this\.(graphS|s)tate\.allowed)`)
	matches := pattern.FindAllStringSubmatch(contentStr, -1)

	if len(matches) == 0 {
		// v18 起 gate 遮罩改由 gl-feature-gate 组件内部根据订阅状态控制：
		// render(){if(lC(this.state))return;...}，其中 lC(t){return null!=t&&(t===lf.Trial||t===lf.Paid)}
		// 同时标题栏搜索行/头部元素受 graphState.allowed 门控（来自 updateState 的响应式属性赋值）
		// 这里分别强制 lC 恒返回 true（遮罩不渲染）、allowed 恒为 true（搜索行/头部元素不隐藏）
		newContent := contentStr
		modified := false

		lCPattern := regexp.MustCompile(`null!=t&&\(t===lf\.Trial\|\|t===lf\.Paid\)`)
		if lCPattern.MatchString(newContent) {
			newContent = lCPattern.ReplaceAllString(newContent, `!0`)
			modified = true
		}

		allowedPattern := regexp.MustCompile(`case"allowed":this\.allowed=t\.allowed\?\?!1;break;`)
		if allowedPattern.MatchString(newContent) {
			newContent = allowedPattern.ReplaceAllString(newContent, `case"allowed":this.allowed=!0;break;`)
			modified = true
		}

		if !modified {
			fmt.Printf("在文件中未找到匹配模式: %s\n", filePath)
			return nil // 不报错，只是跳过
		}

		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("写入文件失败: %v", err)
		}

		fmt.Printf("成功修改文件: %s (v18 gate 遮罩及 allowed 解锁)\n", filePath)
		return nil
	}

	// 替换所有匹配项，在匹配的变量前添加感叹号
	newContent := pattern.ReplaceAllString(contentStr, `$1!$2`)

	// 写入修改后的内容
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fmt.Printf("成功修改文件: %s\n", filePath)
	fmt.Printf("找到 %d 个匹配项\n", len(matches))

	return nil
}

// 向上遍历路径直到找到扩展目录名
func getExtensionDirName(filePath string) string {
	p := filePath
	for {
		base := filepath.Base(p)
		if strings.HasPrefix(base, "eamodio.gitlens-") {
			return base
		}
		parent := filepath.Dir(p)
		if parent == p {
			return ""
		}
		p = parent
	}
}
