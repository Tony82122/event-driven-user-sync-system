package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// ANSI
const (
	Reset    = "\033[0m"
	Bold     = "\033[1m"
	Dim      = "\033[2m"
	White    = "\033[97m"
	Black    = "\033[30m"
	Green    = "\033[32m"
	Yellow   = "\033[33m"
	Red      = "\033[31m"
	Cyan     = "\033[36m"
	BgGreen  = "\033[42m"
	BgYellow = "\033[43m"
	BgRed    = "\033[41m"
	BgCyan   = "\033[46m"
	BgDkGray = "\033[100m"
)

var (
	apiDB       *sql.DB
	crmDB       *sql.DB
	analyticsDB *sql.DB
)

func initDBConnections() {
	var err error
	apiDB, err = sql.Open("postgres", "postgres://postgres:postgres@localhost:5433/api_db?sslmode=disable")
	if err != nil {
		apiDB = nil
	}
	crmDB, err = sql.Open("postgres", "postgres://postgres:postgres@localhost:5433/crm_db?sslmode=disable")
	if err != nil {
		crmDB = nil
	}
	analyticsDB, err = sql.Open("postgres", "postgres://postgres:postgres@localhost:5433/analytics_db?sslmode=disable")
	if err != nil {
		analyticsDB = nil
	}
}

func main() {
	initDBConnections()
	clearScreen()
	printBanner()
	shellLoop()
}

func shellLoop() {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		prompt := buildPrompt()
		fmt.Print(prompt)

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch {
		case input == "exit" || input == "quit" || input == "q":
			fmt.Printf("\n%s%s  Bye %s\n\n", BgCyan, Black, Reset)
			return

		case input == "help" || input == "?":
			printHelp()

		case input == "clear" || input == "cls":
			clearScreen()
			printBanner()

		case input == "status" || input == "s":
			printFullStatus()

		case input == "git" || input == "g":
			printGitDetailed()

		case input == "docker" || input == "d":
			printDockerStatus()

		case input == "health" || input == "h":
			printHealthChecks()

		case input == "up":
			shellExec("docker", "compose", "up", "-d", "--build")

		case input == "down":
			shellExec("docker", "compose", "down", "-v")

		case input == "restart":
			shellExec("docker", "compose", "down", "-v")
			shellExec("docker", "compose", "up", "-d", "--build")

		case strings.HasPrefix(input, "logs"):
			parts := strings.Fields(input)
			if len(parts) > 1 {
				shellExec("docker", "compose", "logs", "-f", "--tail=50", parts[1])
			} else {
				shellExec("docker", "compose", "logs", "-f", "--tail=30")
			}

		case strings.HasPrefix(input, "create-user"):
			parts := strings.Fields(input)
			if len(parts) < 3 {
				fmt.Printf("  %sUsage: create-user <name> <email>%s\n", Red, Reset)
			} else {
				createUser(parts[1], parts[2])
			}

		case input == "list-users" || input == "users":
			listUsers()

		case strings.HasPrefix(input, "get-user "):
			getUserByID(strings.TrimPrefix(input, "get-user "))

		case input == "count-users":
			countUsers()

		case input == "queues" || input == "rabbit":
			printRabbitQueues()

		// --- CRM commands ---
		case input == "crm-syncs" || input == "crm-log":
			crmShowSyncLog()

		case input == "crm-count":
			crmCountSyncs()

		case input == "crm-recent":
			crmShowRecent()

		case input == "crm-failed":
			crmShowFailed()

		case input == "crm-keys":
			showIdempotencyKeys(crmDB, "crm")

		// --- Analytics commands ---
		case input == "analytics-metrics" || input == "metrics":
			analyticsShowMetrics()

		case input == "analytics-today" || input == "today":
			analyticsShowToday()

		case input == "analytics-daily" || input == "daily":
			analyticsShowDaily()

		case input == "analytics-total":
			analyticsShowTotal()

		case input == "analytics-keys":
			showIdempotencyKeys(analyticsDB, "analytics")

		// --- DB inspection ---
		case input == "tables-api":
			showTables(apiDB, "api")

		case input == "tables-crm":
			showTables(crmDB, "crm")

		case input == "tables-analytics":
			showTables(analyticsDB, "analytics")

		case strings.HasPrefix(input, "sql-api "):
			rawSQL(apiDB, "api", strings.TrimPrefix(input, "sql-api "))

		case strings.HasPrefix(input, "sql-crm "):
			rawSQL(crmDB, "crm", strings.TrimPrefix(input, "sql-crm "))

		case strings.HasPrefix(input, "sql-analytics "):
			rawSQL(analyticsDB, "analytics", strings.TrimPrefix(input, "sql-analytics "))

		default:
			// Pass through to system shell
			shellExecRaw(input)
		}

		fmt.Println()
	}
}

func buildPrompt() string {
	branch, dirty, staged, modified, untracked := getGitInfo()
	dir := getShortDir()

	barBg := BgGreen
	statusText := "clean"
	if dirty {
		barBg = BgYellow
		parts := []string{}
		if staged > 0 {
			parts = append(parts, fmt.Sprintf("%d staged", staged))
		}
		if modified > 0 {
			parts = append(parts, fmt.Sprintf("%d modified", modified))
		}
		if untracked > 0 {
			parts = append(parts, fmt.Sprintf("%d untracked", untracked))
		}
		statusText = strings.Join(parts, " | ")
	}

	// Check if inside a container
	containerTag := ""
	if isInsideContainer() {
		containerTag = fmt.Sprintf(" %s%s  CONTAINER %s", BgCyan, Black, Reset)
	}

	// Build the bar
	bar := fmt.Sprintf("%s%s %s  %s | %s %s%s",
		barBg, Black,
		dir,
		branch,
		statusText,
		Reset,
		containerTag,
	)

	return fmt.Sprintf("%s\n%s>%s ", bar, Cyan, Reset)
}

func getGitInfo() (branch string, dirty bool, staged, modified, untracked int) {
	branch = strings.TrimSpace(runCmd("git", "rev-parse", "--abbrev-ref", "HEAD"))
	if branch == "" {
		branch = "no-repo"
	}

	status := strings.TrimSpace(runCmd("git", "status", "--porcelain"))
	if status == "" {
		return branch, false, 0, 0, 0
	}

	for _, line := range strings.Split(status, "\n") {
		if len(line) < 2 {
			continue
		}
		x := line[0]
		y := line[1]
		if x == '?' {
			untracked++
		} else if x != ' ' {
			staged++
		}
		if y != ' ' && y != '?' {
			modified++
		}
	}

	return branch, true, staged, modified, untracked
}

func getShortDir() string {
	dir, _ := os.Getwd()
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(dir, home) {
		dir = "~" + dir[len(home):]
	}
	// Shorten to last 2 segments
	parts := strings.Split(dir, string(os.PathSeparator))
	if len(parts) > 2 {
		dir = "../" + strings.Join(parts[len(parts)-2:], "/")
	}
	return dir
}

func isInsideContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	data, err := os.ReadFile("/proc/1/cgroup")
	if err == nil && (strings.Contains(string(data), "docker") || strings.Contains(string(data), "kubepods")) {
		return true
	}
	return false
}

func printFullStatus() {
	printGitDetailed()
	fmt.Println()
	printDockerStatus()
	fmt.Println()
	printHealthChecks()
}

func printGitDetailed() {
	fmt.Printf("  %s%sGit%s\n", Bold, White, Reset)

	branch, dirty, staged, modified, untracked := getGitInfo()
	lastCommit := strings.TrimSpace(runCmd("git", "log", "--oneline", "-1"))

	if !dirty {
		fmt.Printf("  %s[*]%s %s -- clean\n", Green, Reset, branch)
	} else {
		fmt.Printf("  %s[*]%s %s -- modified\n", Yellow, Reset, branch)
		if staged > 0 {
			fmt.Printf("    %s+%d staged%s\n", Green, staged, Reset)
		}
		if modified > 0 {
			fmt.Printf("    %s~%d modified%s\n", Yellow, modified, Reset)
		}
		if untracked > 0 {
			fmt.Printf("    %s?%d untracked%s\n", Red, untracked, Reset)
		}
	}
	if lastCommit != "" {
		fmt.Printf("  %s%s%s\n", Dim, lastCommit, Reset)
	}
}

func printDockerStatus() {
	fmt.Printf("  %s%sDocker%s\n", Bold, White, Reset)

	output := strings.TrimSpace(runCmd("docker", "ps", "-a", "--filter", "name=awesomeproject",
		"--format", "{{.Names}}|{{.Status}}|{{.Ports}}"))

	if output == "" {
		fmt.Printf("  %s[-] no containers%s\n", Dim, Reset)
		return
	}

	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimPrefix(parts[0], "awesomeproject-")
		name = strings.TrimSuffix(name, "-1")
		status := parts[1]

		color := Red
		icon := "[-]"
		if strings.Contains(status, "Up") {
			color = Green
			icon = "[+]"
		}

		port := ""
		if len(parts) > 2 && parts[2] != "" {
			for _, p := range strings.Split(parts[2], ",") {
				p = strings.TrimSpace(p)
				if strings.Contains(p, "->") {
					host := strings.Split(p, "->")[0]
					host = strings.TrimPrefix(host, "0.0.0.0:")
					port = fmt.Sprintf(" %s-> %s%s", Dim, host, Reset)
				}
			}
		}

		fmt.Printf("  %s%s%s %-22s%s\n", color, icon, Reset, name, port)
	}
}

func printHealthChecks() {
	fmt.Printf("  %s%sHealth%s\n", Bold, White, Reset)

	endpoints := []struct {
		name string
		url  string
	}{
		{"api", "http://localhost:8081/health"},
		{"rabbitmq", "http://localhost:15672/"},
	}

	for _, ep := range endpoints {
		client := http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(ep.url)
		if err != nil {
			fmt.Printf("  %s[-]%s %-12s %soffline%s\n", Red, Reset, ep.name, Red, Reset)
			continue
		}
		resp.Body.Close()
		fmt.Printf("  %s[+]%s %-12s %sok%s\n", Green, Reset, ep.name, Green, Reset)
	}
}

func printRabbitQueues() {
	fmt.Printf("  %s%sRabbitMQ Queues%s\n", Bold, White, Reset)

	output := strings.TrimSpace(runCmd("docker", "exec", "awesomeproject-rabbitmq-1",
		"rabbitmqctl", "list_queues", "name", "messages", "consumers", "--quiet"))

	if output == "" {
		fmt.Printf("  %s[-] rabbitmq not reachable%s\n", Dim, Reset)
		return
	}

	fmt.Printf("  %s%-35s %8s %10s%s\n", Dim, "QUEUE", "MSGS", "CONSUMERS", Reset)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		color := Green
		if fields[1] != "0" {
			color = Yellow
		}
		fmt.Printf("  %s%-35s %s%8s%s %10s\n", Dim, fields[0], color, fields[1], Reset, fields[2])
	}
}

func createUser(name, email string) {
	body := fmt.Sprintf(`{"name":"%s","email":"%s"}`, name, email)
	resp, err := http.Post("http://localhost:8081/users", "application/json",
		strings.NewReader(body))
	if err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
		return
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	if resp.StatusCode == 201 {
		fmt.Printf("  %s[ok] created%s\n  %s\n", Green, Reset, buf.String())
	} else {
		fmt.Printf("  %s[x] %d%s %s\n", Red, resp.StatusCode, Reset, buf.String())
	}
}

func listUsers() {
	resp, err := http.Get("http://localhost:8081/users")
	if err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
		return
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	fmt.Printf("  %s\n", buf.String())
}

func printHelp() {
	fmt.Println()
	fmt.Printf("  %s%sCommands%s\n", Bold, White, Reset)
	fmt.Printf("  %sstatus%s  s    full dashboard\n", Green, Reset)
	fmt.Printf("  %sgit%s     g    git info\n", Green, Reset)
	fmt.Printf("  %sdocker%s  d    container status\n", Green, Reset)
	fmt.Printf("  %shealth%s  h    health checks\n", Green, Reset)
	fmt.Printf("  %squeues%s       rabbitmq queues\n", Green, Reset)
	fmt.Println()
	fmt.Printf("  %s--- Stack ---%s\n", Dim, Reset)
	fmt.Printf("  %sup%s           start stack\n", Green, Reset)
	fmt.Printf("  %sdown%s         stop stack\n", Green, Reset)
	fmt.Printf("  %srestart%s      restart stack\n", Green, Reset)
	fmt.Printf("  %slogs%s [svc]   tail logs\n", Green, Reset)
	fmt.Println()
	fmt.Printf("  %s--- API / Users ---%s\n", Dim, Reset)
	fmt.Printf("  %screate-user%s  <name> <email>\n", Green, Reset)
	fmt.Printf("  %susers%s        list users\n", Green, Reset)
	fmt.Printf("  %sget-user%s     <id>  get user by id\n", Green, Reset)
	fmt.Printf("  %scount-users%s  count users in api db\n", Green, Reset)
	fmt.Println()
	fmt.Printf("  %s--- CRM ---%s\n", Dim, Reset)
	fmt.Printf("  %scrm-syncs%s    sync log (last 20)\n", Green, Reset)
	fmt.Printf("  %scrm-count%s    events by type/status\n", Green, Reset)
	fmt.Printf("  %scrm-recent%s   last 5 events\n", Green, Reset)
	fmt.Printf("  %scrm-failed%s   show failed syncs\n", Green, Reset)
	fmt.Printf("  %scrm-keys%s     idempotency keys\n", Green, Reset)
	fmt.Println()
	fmt.Printf("  %s--- Analytics ---%s\n", Dim, Reset)
	fmt.Printf("  %smetrics%s      all metrics (with bars)\n", Green, Reset)
	fmt.Printf("  %stoday%s        today's metrics\n", Green, Reset)
	fmt.Printf("  %sdaily%s        daily totals (last 14d)\n", Green, Reset)
	fmt.Printf("  %sanalytics-total%s  all-time by type\n", Green, Reset)
	fmt.Printf("  %sanalytics-keys%s   idempotency keys\n", Green, Reset)
	fmt.Println()
	fmt.Printf("  %s--- DB ---%s\n", Dim, Reset)
	fmt.Printf("  %stables-api%s / %stables-crm%s / %stables-analytics%s\n", Green, Reset, Green, Reset, Green, Reset)
	fmt.Printf("  %ssql-api%s / %ssql-crm%s / %ssql-analytics%s <query>\n", Green, Reset, Green, Reset, Green, Reset)
	fmt.Println()
	fmt.Printf("  %sclear%s        clear screen\n", Green, Reset)
	fmt.Printf("  %sexit%s         quit shell\n", Green, Reset)
	fmt.Println()
	fmt.Printf("  %sAnything else is passed to your system shell.%s\n", Dim, Reset)
}

func printBanner() {
	fmt.Println()
	fmt.Printf("  %s%s>> Event-Driven User Sync System%s\n", Bold, Cyan, Reset)
	fmt.Printf("  %sType 'help' for commands, or use any shell command%s\n", Dim, Reset)
	fmt.Println()
}

// ---------------------------------------------------------------------------
// API DB commands
// ---------------------------------------------------------------------------

func getUserByID(id string) {
	if apiDB == nil || apiDB.Ping() != nil {
		fmt.Printf("  %s[x] api db not reachable%s\n", Red, Reset)
		return
	}
	var email, name string
	var created, updated time.Time
	err := apiDB.QueryRow("SELECT email, name, created_at, updated_at FROM users WHERE id = $1", id).
		Scan(&email, &name, &created, &updated)
	if err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
		return
	}
	fmt.Printf("  %sid:%s     %s\n", Dim, Reset, id)
	fmt.Printf("  %semail:%s  %s\n", Dim, Reset, email)
	fmt.Printf("  %sname:%s   %s\n", Dim, Reset, name)
	fmt.Printf("  %screated:%s %s\n", Dim, Reset, created.Format(time.RFC3339))
	fmt.Printf("  %supdated:%s %s\n", Dim, Reset, updated.Format(time.RFC3339))
}

func countUsers() {
	if apiDB == nil || apiDB.Ping() != nil {
		fmt.Printf("  %s[x] api db not reachable%s\n", Red, Reset)
		return
	}
	var count int
	apiDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	fmt.Printf("  %s%d%s users\n", Bold, count, Reset)
}

// ---------------------------------------------------------------------------
// CRM commands
// ---------------------------------------------------------------------------

func crmShowSyncLog() {
	if crmDB == nil || crmDB.Ping() != nil {
		fmt.Printf("  %s[x] crm db not reachable%s\n", Red, Reset)
		return
	}
	rows, err := crmDB.Query(`SELECT event_id, event_type, user_email, status, synced_at
		FROM crm_sync_log ORDER BY synced_at DESC LIMIT 20`)
	if err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
		return
	}
	defer rows.Close()

	fmt.Printf("  %s%-38s %-15s %-25s %-10s %s%s\n", Bold, "EVENT_ID", "TYPE", "EMAIL", "STATUS", "TIME", Reset)
	fmt.Printf("  %s%s%s\n", Dim, strings.Repeat("-", 110), Reset)
	for rows.Next() {
		var eventID, eventType, email, status string
		var syncedAt time.Time
		rows.Scan(&eventID, &eventType, &email, &status, &syncedAt)
		color := Green
		if status == "failed" {
			color = Red
		}
		fmt.Printf("  %-38s %-15s %-25s %s%-10s%s %s\n",
			eventID, eventType, email, color, status, Reset, syncedAt.Format("15:04:05"))
	}
}

func crmCountSyncs() {
	if crmDB == nil || crmDB.Ping() != nil {
		fmt.Printf("  %s[x] crm db not reachable%s\n", Red, Reset)
		return
	}
	rows, err := crmDB.Query(`SELECT event_type, status, COUNT(*)
		FROM crm_sync_log GROUP BY event_type, status ORDER BY event_type`)
	if err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
		return
	}
	defer rows.Close()
	fmt.Printf("  %s%-20s %-12s %s%s\n", Bold, "TYPE", "STATUS", "COUNT", Reset)
	for rows.Next() {
		var eventType, status string
		var count int
		rows.Scan(&eventType, &status, &count)
		color := Green
		if status == "failed" {
			color = Red
		}
		fmt.Printf("  %-20s %s%-12s%s %d\n", eventType, color, status, Reset, count)
	}
}

func crmShowRecent() {
	if crmDB == nil || crmDB.Ping() != nil {
		fmt.Printf("  %s[x] crm db not reachable%s\n", Red, Reset)
		return
	}
	rows, err := crmDB.Query(`SELECT event_type, user_email, status, synced_at
		FROM crm_sync_log ORDER BY synced_at DESC LIMIT 5`)
	if err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var eventType, email, status string
		var at time.Time
		rows.Scan(&eventType, &email, &status, &at)
		icon := fmt.Sprintf("%s[ok]%s", Green, Reset)
		if status == "failed" {
			icon = fmt.Sprintf("%s[fail]%s", Red, Reset)
		}
		fmt.Printf("  %s %s %s %s %s\n", icon, at.Format("15:04:05"), eventType, email, Dim+status+Reset)
	}
}

func crmShowFailed() {
	if crmDB == nil || crmDB.Ping() != nil {
		fmt.Printf("  %s[x] crm db not reachable%s\n", Red, Reset)
		return
	}
	rows, err := crmDB.Query(`SELECT event_id, event_type, user_email, synced_at
		FROM crm_sync_log WHERE status = 'failed' ORDER BY synced_at DESC LIMIT 20`)
	if err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
		return
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var eventID, eventType, email string
		var at time.Time
		rows.Scan(&eventID, &eventType, &email, &at)
		fmt.Printf("  %s[x]%s %-38s %s %s %s\n", Red, Reset, eventID, eventType, email, at.Format("15:04:05"))
		count++
	}
	if count == 0 {
		fmt.Printf("  %sNo failures%s\n", Green, Reset)
	}
}

// ---------------------------------------------------------------------------
// Analytics commands
// ---------------------------------------------------------------------------

func analyticsShowMetrics() {
	if analyticsDB == nil || analyticsDB.Ping() != nil {
		fmt.Printf("  %s[x] analytics db not reachable%s\n", Red, Reset)
		return
	}
	rows, err := analyticsDB.Query(`SELECT metric_date, event_type, event_count
		FROM analytics_metrics ORDER BY metric_date DESC, event_type LIMIT 30`)
	if err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
		return
	}
	defer rows.Close()

	fmt.Printf("  %s%-12s %-20s %s%s\n", Bold, "DATE", "TYPE", "COUNT", Reset)
	fmt.Printf("  %s%s%s\n", Dim, strings.Repeat("-", 45), Reset)
	for rows.Next() {
		var date, eventType string
		var count int
		rows.Scan(&date, &eventType, &count)
		bar := strings.Repeat("#", minInt(count, 40))
		fmt.Printf("  %-12s %-20s %s%s%s %d\n", date, eventType, Green, bar, Reset, count)
	}
}

func analyticsShowToday() {
	if analyticsDB == nil || analyticsDB.Ping() != nil {
		fmt.Printf("  %s[x] analytics db not reachable%s\n", Red, Reset)
		return
	}
	today := time.Now().Format("2006-01-02")
	rows, err := analyticsDB.Query(`SELECT event_type, event_count
		FROM analytics_metrics WHERE metric_date = $1 ORDER BY event_type`, today)
	if err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
		return
	}
	defer rows.Close()

	fmt.Printf("  %s%sToday (%s)%s\n", Bold, White, today, Reset)
	total := 0
	for rows.Next() {
		var eventType string
		var count int
		rows.Scan(&eventType, &count)
		bar := strings.Repeat("#", minInt(count, 40))
		fmt.Printf("  %-20s %s%s%s %d\n", eventType, Cyan, bar, Reset, count)
		total += count
	}
	fmt.Printf("  %stotal: %d%s\n", Dim, total, Reset)
}

func analyticsShowDaily() {
	if analyticsDB == nil || analyticsDB.Ping() != nil {
		fmt.Printf("  %s[x] analytics db not reachable%s\n", Red, Reset)
		return
	}
	rows, err := analyticsDB.Query(`SELECT metric_date, SUM(event_count) as total
		FROM analytics_metrics GROUP BY metric_date ORDER BY metric_date DESC LIMIT 14`)
	if err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
		return
	}
	defer rows.Close()

	fmt.Printf("  %s%sDaily Totals%s\n", Bold, White, Reset)
	for rows.Next() {
		var date string
		var total int
		rows.Scan(&date, &total)
		bar := strings.Repeat("#", minInt(total, 50))
		fmt.Printf("  %-12s %s%s%s %d\n", date, Green, bar, Reset, total)
	}
}

func analyticsShowTotal() {
	if analyticsDB == nil || analyticsDB.Ping() != nil {
		fmt.Printf("  %s[x] analytics db not reachable%s\n", Red, Reset)
		return
	}
	rows, err := analyticsDB.Query(`SELECT event_type, SUM(event_count) as total
		FROM analytics_metrics GROUP BY event_type ORDER BY total DESC`)
	if err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
		return
	}
	defer rows.Close()

	fmt.Printf("  %s%sAll-Time Totals%s\n", Bold, White, Reset)
	for rows.Next() {
		var eventType string
		var total int
		rows.Scan(&eventType, &total)
		bar := strings.Repeat("#", minInt(total, 50))
		fmt.Printf("  %-20s %s%s%s %d\n", eventType, Cyan, bar, Reset, total)
	}
}

// ---------------------------------------------------------------------------
// Shared DB helpers
// ---------------------------------------------------------------------------

func showIdempotencyKeys(db *sql.DB, label string) {
	if db == nil || db.Ping() != nil {
		fmt.Printf("  %s[x] %s db not reachable%s\n", Red, label, Reset)
		return
	}
	rows, err := db.Query("SELECT event_id, processed_at FROM idempotency_keys ORDER BY processed_at DESC LIMIT 10")
	if err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
		return
	}
	defer rows.Close()
	fmt.Printf("  %s%-38s %s%s\n", Bold, "EVENT_ID", "PROCESSED_AT", Reset)
	for rows.Next() {
		var id string
		var at time.Time
		rows.Scan(&id, &at)
		fmt.Printf("  %-38s %s\n", id, at.Format("2006-01-02 15:04:05"))
	}
}

func showTables(db *sql.DB, label string) {
	if db == nil || db.Ping() != nil {
		fmt.Printf("  %s[x] %s db not reachable%s\n", Red, label, Reset)
		return
	}
	rows, err := db.Query("SELECT tablename FROM pg_tables WHERE schemaname = 'public'")
	if err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
		return
	}
	defer rows.Close()
	fmt.Printf("  %s%s%s tables:\n", Bold, label, Reset)
	for rows.Next() {
		var name string
		rows.Scan(&name)
		fmt.Printf("  - %s\n", name)
	}
}

func rawSQL(db *sql.DB, label, query string) {
	if db == nil || db.Ping() != nil {
		fmt.Printf("  %s[x] %s db not reachable%s\n", Red, label, Reset)
		return
	}
	rows, err := db.Query(query)
	if err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
		return
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	fmt.Printf("  %s%s%s\n", Bold, strings.Join(cols, "\t"), Reset)
	vals := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	for rows.Next() {
		rows.Scan(ptrs...)
		parts := make([]string, len(cols))
		for i, v := range vals {
			parts[i] = fmt.Sprintf("%v", v)
		}
		fmt.Printf("  %s\n", strings.Join(parts, "\t"))
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func shellExec(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fmt.Printf("  %s[x] %v%s\n", Red, err, Reset)
	}
}

func shellExecRaw(input string) {
	shell := "sh"
	flag := "-c"
	if _, err := exec.LookPath("powershell.exe"); err == nil {
		shell = "powershell.exe"
		flag = "-Command"
	}
	if _, err := exec.LookPath("bash"); err == nil {
		shell = "bash"
		flag = "-c"
	}

	cmd := exec.Command(shell, flag, input)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Run()
}

func runCmd(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	cmd.Run()
	return out.String()
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}
