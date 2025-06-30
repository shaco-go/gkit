package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var verboseFlag bool

// newCmd 表示new命令
var newCmd = &cobra.Command{
	Use:   "new [github-repo] [project-name]",
	Short: "Create a new project from template",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		repoURL := args[0]
		projectName := args[1]

		// 确保仓库URL有http或https前缀
		if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
			repoURL = "https://" + repoURL
		}

		// 提取不带http的模块路径，用于替换导入路径
		moduleImportPath := repoURL
		moduleImportPath = strings.TrimPrefix(moduleImportPath, "http://")
		moduleImportPath = strings.TrimPrefix(moduleImportPath, "https://")

		err := runNewProject(repoURL, moduleImportPath, projectName, verboseFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
	newCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Show detailed output")
}

func runNewProject(repoURL string, moduleImportPath string, projectName string, verbose bool) error {
	// 1. 验证项目名合法性
	if err := validateProjectName(projectName); err != nil {
		return err
	}

	// 2. 检查目标目录是否存在
	if _, err := os.Stat(projectName); err == nil {
		return fmt.Errorf("directory '%s' already exists", projectName)
	}

	// 3. 克隆模板仓库
	if err := cloneTemplate(repoURL, projectName, verbose); err != nil {
		return err
	}

	// 4. 替换模块名
	if err := replaceModuleName(projectName, moduleImportPath, verbose); err != nil {
		return err
	}

	// 5. 处理依赖
	if err := handleDependencies(projectName, verbose); err != nil {
		return err
	}

	// 6. 删除.git目录
	if err := removeGitDir(projectName, verbose); err != nil {
		return err
	}

	fmt.Printf("Project '%s' created successfully!\n", projectName)
	return nil
}

func validateProjectName(name string) error {
	// 检查项目名是否符合Go模块命名规范
	validName := regexp.MustCompile(`^[a-zA-Z0-9_\-./]+$`)
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid project name: must contain only letters, numbers, underscores, hyphens, dots, and slashes")
	}
	return nil
}

func cloneTemplate(repoURL string, projectName string, verbose bool) error {
	// 使用git clone命令克隆指定的模板仓库
	fmt.Println("Cloning template repository...")

	args := []string{"clone", "--depth", "1", repoURL, projectName}
	if verbose {
		fmt.Printf("Running: git %s\n", strings.Join(args, " "))
	}

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func replaceModuleName(projectName string, moduleImportPath string, verbose bool) error {
	fmt.Println("Replacing module name...")

	// 1. 修改go.mod文件中的模块名 - 使用projectName作为模块名
	if verbose {
		fmt.Printf("Running: go mod edit -module %s\n", projectName)
	}

	cmd := exec.Command("go", "mod", "edit", "-module", projectName)
	cmd.Dir = projectName
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update module name: %v", err)
	}

	// 2. 遍历所有.go文件，替换导入路径
	return filepath.Walk(projectName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过.git目录
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// 处理.go文件
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			if verbose {
				fmt.Printf("Processing file: %s\n", path)
			}

			// 读取文件内容
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %v", path, err)
			}

			// 替换导入路径：从moduleImportPath替换为projectName
			newContent := strings.ReplaceAll(string(content), "github.com/shaco-go/gkit-layout", projectName)

			// 写回文件
			if err := os.WriteFile(path, []byte(newContent), info.Mode()); err != nil {
				return fmt.Errorf("failed to write file %s: %v", path, err)
			}
		}

		return nil
	})
}

func handleDependencies(projectName string, verbose bool) error {
	fmt.Println("Updating dependencies...")

	// 1. 执行go mod tidy
	if verbose {
		fmt.Println("Running: go mod tidy")
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = projectName
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to tidy dependencies: %v", err)
	}

	// 2. 执行go mod download
	if verbose {
		fmt.Println("Running: go mod download")
	}

	cmd = exec.Command("go", "mod", "download")
	cmd.Dir = projectName
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download dependencies: %v", err)
	}

	return nil
}

// 删除.git目录
func removeGitDir(projectName string, verbose bool) error {
	gitDir := filepath.Join(projectName, ".git")

	if verbose {
		fmt.Printf("Removing .git directory: %s\n", gitDir)
	}

	err := os.RemoveAll(gitDir)
	if err != nil {
		return fmt.Errorf("failed to remove .git directory: %v", err)
	}

	fmt.Println("Removed .git directory successfully")
	return nil
}
