package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"

	"github.com/PuerkitoBio/goquery"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(getCmd)
}

// getCmd 表示get命令
var getCmd = &cobra.Command{
	Use:   "get [package-name]",
	Short: "Download and install Go packages",
	Run: func(cmd *cobra.Command, args []string) {
		packageName := args[0]
		goGetPackage(packageName)
	},
}

const listHeight = 14

var p *tea.Program

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

// 包名item
type item struct {
	Package string
	Star    string
	From    string
}

type searchLoadMsg struct{}

type searchDoneMsg struct{}

type outputMsg struct {
	msg string
}
type outputDoneMsg struct{}

func (i item) FilterValue() string { return i.Package }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s (%s)", index+1, i.Package, i.Star)
	if i.From != "" {
		str = fmt.Sprintf("%d. %s (from %s)", index+1, i.Package, i.From)
	} else if i.Star == "" {
		str = fmt.Sprintf("%d. %s", index+1, i.Package)
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}
	fmt.Fprint(w, fn(str))
}

type model struct {
	spinner    spinner.Model
	list       list.Model
	output     []string
	readOutput bool
	quitting   bool
	loading    bool
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, searchPackages)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			switch keypress := msg.String(); keypress {
			case "q", "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				// 开始输出了
				execPackage(i.Package)
				return m, nil
			} else {
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case searchLoadMsg:
		m.loading = true
		return m, nil

	case searchDoneMsg:
		m.loading = false
		return m, nil

	case searchResultMsg:
		m.list.SetItems(msg.items)
		return m, nil
	case outputMsg:
		m.readOutput = true
		m.output = append(m.output, msg.msg)
		return m, nil
	case outputDoneMsg:
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return quitTextStyle.Render("")
	}
	if m.loading {
		return "\n  " + m.spinner.View() + " Searching for packages..."
	}
	if m.readOutput {
		var s strings.Builder
		for _, output := range m.output {
			s.WriteString(output + "\n")
		}
		return s.String()
	}
	return "\n" + m.list.View()
}

func execPackage(pb string) tea.Cmd {
	go func() {
		// 命令示例
		cmd := exec.Command("go", "get", "-u", pb)

		// 获取标准输出管道
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to get stdout: %v\n", err)
			os.Exit(1)
		}
		cmd.Stderr = cmd.Stdout

		// 启动命令
		if err := cmd.Start(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to start command: %v\n", err)
			os.Exit(1)
		}

		// 根据操作系统选择合适的阅读器
		var reader io.Reader = stdout

		// 在Windows系统上，转换编码
		if runtime.GOOS == "windows" {
			// 使用GBK解码器将GBK转为UTF-8
			reader = transform.NewReader(stdout, simplifiedchinese.GBK.NewDecoder())
		}

		// 创建缓冲读取器
		scanner := bufio.NewScanner(reader)

		// 读取并处理每一行
		for scanner.Scan() {
			p.Send(outputMsg{msg: scanner.Text()})
		}
		_ = cmd.Wait()
		p.Send(outputDoneMsg{})
	}()
	return nil
}

// 搜索 golang 模块
func searchPkgGo(kw string) ([]list.Item, error) {
	resp, err := http.Get(fmt.Sprintf("https://pkg.go.dev/search?q=%s&m=package", url.QueryEscape(kw)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("search for https://pkg.go.dev/search?q=%s Package error: %d %s", kw, resp.StatusCode, resp.Status)
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	var result []list.Item
	if doc.Find(".go-Content.SearchResults").Length() == 0 {
		result = append(result, item{
			Package: kw,
		})
		return result, nil
	}
	doc.Find(".go-Content.SearchResults  .SearchSnippet").Each(func(i int, s *goquery.Selection) {
		var path string
		path = strings.Trim(s.Find(".SearchSnippet-header-path").Text(), "(")
		path = strings.Trim(path, ")")
		importCount := s.Find("a[aria-label='Go to Imported By']  strong").Text()
		result = append(result, item{
			Star:    importCount,
			Package: path,
		})
		if val := s.Find(".SearchSnippet-sub.go-textSubtle strong").Text(); val != "" {
			subPath := strings.Replace(val, "Other packages in module ", "", 1)
			subPath = strings.Trim(subPath, ":")
			result = append(result, item{
				Star:    importCount,
				Package: subPath,
				From:    path,
			})
		}
	})
	return result, nil
}

// 搜索包并发送消息的命令
func searchPackages() tea.Msg {
	return searchLoadMsg{}
}

// 搜索结果消息
type searchResultMsg struct {
	items []list.Item
}

// go get
func goGetPackage(packageName string) {
	const defaultWidth = 30

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// 创建一个空列表，稍后填充
	l := list.New([]list.Item{}, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Please select the package to be imported."
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	l.SetHeight(20)
	l.SetStatusBarItemName("package", "packages")

	m := model{
		spinner: s,
		list:    l,
		loading: true,
	}

	p = tea.NewProgram(m)

	// 启动搜索协程
	go func() {
		items, err := searchPkgGo(packageName)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		// 更新列表并发送完成消息
		p.Send(searchResultMsg{items: items})
		p.Send(searchDoneMsg{})
	}()

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
