package main

import (
	"os"
	"fmt"
	"time"
	"flag"
	"database/sql"
	"bytes"
	"strings"
	"os/exec"
	"io/ioutil"
	"crypto/md5"
	"encoding/json"
	_ "github.com/mattn/go-sqlite3"
)

type ArticleSource struct {
	Placeholder string      `json:"placeholder"`
	ListCommand string      `json:"listCommand"`
	TitleCommand string     `json:"titleCommand"`
	ContentCommand string   `json:"contentCommand"`
}
type ArticleSources []ArticleSource

func checkErr(e error) {
	if e != nil {
		panic(e)
	}
}

func removeEmptyString(slice []string) []string {
	newSlice := []string{}
	for _, element := range slice {
		if len(element) > 0 {
			newSlice = append(newSlice, element)
		}
	}
	return newSlice
}

func prepare(database string) {
	fmt.Println("Database:", database)

	// DB Schema
	
	// CREATE TABLE 'articles' (
	// 	...> 'id' TEXT PRIMARY KEY,
	// 	...> 'title' TEXT,
	// 	...> 'content' TEXT,
	// 	...> 'url' TEXT,
	// 	...> 'created' INTEGER,
    // 	...> 'read' BOOLEAN
	// 	...> );
	// Open SQLite database file

	db, err := sql.Open("sqlite3", database)
	checkErr(err)
	defer db.Close()

	stmt, err := db.Prepare(
		`CREATE TABLE 'articles' (
           'id' TEXT PRIMARY KEY,
           'title' TEXT,
           'content' TEXT,
           'url' TEXT,
           'created' INTEGER,
           'read' BOOLEAN,
           'ignored' BOOLEAN
         )`)
	checkErr(err)
	_, err = stmt.Exec()
	checkErr(err)

	stmt, err = db.Prepare(
		`CREATE TABLE 'archived' (
           'id' TEXT PRIMARY KEY,
           'title' TEXT,
           'content' TEXT,
           'url' TEXT,
           'created' INTEGER,
           'rating' INTEGER
         )`)
	checkErr(err)
	_, err = stmt.Exec()
	checkErr(err)
}

func scrape(sources, database string, interval, delay, limit int) {
	fmt.Println("Sources:", sources)
	fmt.Println("Database:", database)
	fmt.Println("Interval:", interval)
	fmt.Println("Delay:", delay)
	fmt.Println("Limit:", limit)

	// Open SQLite database file
	db, err := sql.Open("sqlite3", database)
	checkErr(err)
	defer db.Close()

	// Read article sources
	dat, err := ioutil.ReadFile(sources)
	checkErr(err)

	var sourceList ArticleSources
	err = json.Unmarshal(dat, &sourceList)
	checkErr(err)

	// Scrape news articles
	for {
		for _, source := range sourceList {
			var out bytes.Buffer
			cmd := exec.Command("sh", "-c", source.ListCommand)
			cmd.Stdout = &out
			err = cmd.Run()
			checkErr(err)

			urls := removeEmptyString(strings.Split(out.String(), "\n"))
			for _, url := range urls {
				fmt.Println(">> URL:", url)

				data := []byte(url)
				id := fmt.Sprintf("%x", md5.Sum(data))

				// Check if this article already exists in the database
				var dummy string
				err := db.QueryRow("SELECT title FROM articles WHERE id = ?", id).Scan(&dummy)
				if err != sql.ErrNoRows {
					fmt.Println("duplicate item.")
					continue
				}

				cmd = exec.Command("sh", "-c", strings.Replace(source.TitleCommand, source.Placeholder, url, -1))
				cmd.Stdout = &out
				out.Reset()
				err = cmd.Run()
				checkErr(err)
				title := strings.TrimSpace(out.String())
				fmt.Println(title)

				cmd = exec.Command("sh", "-c", strings.Replace(source.ContentCommand, source.Placeholder, url, -1))
				cmd.Stdout = &out
				out.Reset()
				err = cmd.Run()
				if err != nil {
					fmt.Println("Failed to retrieve an article")
					continue
				}
				//checkErr(err)
				content := strings.TrimSpace(out.String())

				t := time.Now()
				created := t.Unix()

				// Insert to the articles database
				stmt, err := db.Prepare("INSERT INTO articles(id, title, content, url, created, read, ignored) values(?,?,?,?,?,?,?)")
				checkErr(err)

				_, err = stmt.Exec(id, title, content, url, created, false, false)
				checkErr(err)

				time.Sleep(time.Duration(delay) * 1000 * time.Millisecond)
			}
		}

		// Delete excessive items
		stmt, err := db.Prepare("DELETE FROM articles WHERE id NOT IN (SELECT id FROM articles ORDER BY created DESC LIMIT ?)")
		checkErr(err)
		_, err = stmt.Exec(limit)
		checkErr(err)

		fmt.Printf(">> Wait for %d secs.. \n", interval)
		time.Sleep(time.Duration(interval) * 1000 * time.Millisecond)
	}
}

func list(database string, page, pageSize int, unreadOnly bool) {
	// fmt.Println("Database:", database)
	// fmt.Println("Page:", page)
	// fmt.Println("PageSize:", pageSize)
	// fmt.Println("UnreadOnly:", unreadOnly)

	// Open SQLite database file
	db, err := sql.Open("sqlite3", database)
	checkErr(err)
	defer db.Close()

	statement := "SELECT id, title, created, read FROM articles WHERE ignored = 0 "
	if unreadOnly {
		statement += "AND read = 0 "
	}
	statement += "ORDER BY created DESC LIMIT ? OFFSET ?"

    rows, err := db.Query(statement, pageSize, page * pageSize)
	checkErr(err)

	defer rows.Close()
	for rows.Next() {
		var id, title string
		var created int
		var read bool
		rows.Scan(&id, &title, &created, &read)
		checkErr(err)
		readNum := 0
		if read {
			readNum = 1
		}
		t := time.Unix(int64(created), int64(0))
		fmt.Printf("%s\t%s\t%d\t%s\n", id, t.Format(time.UnixDate), readNum, title)
	}
}

func show(database, id string) {
	// fmt.Println("Database:", database)
	// fmt.Println("Id:", id)

	// Open SQLite database file
	db, err := sql.Open("sqlite3", database)
	checkErr(err)
	defer db.Close()

    rows, err := db.Query("SELECT title, content, created FROM articles WHERE id = ?", id)
	checkErr(err)

	defer rows.Close()
	for rows.Next() {
		var title, content string
		var created int
		rows.Scan(&title, &content, &created)
		checkErr(err)

		t := time.Unix(int64(created), int64(0))
		fmt.Printf("%s\n%s\n\n%s\n", t.Format(time.UnixDate), title, content)
	}
}

func read(database, id string, unread bool) {
	// fmt.Println("Database:", database)
	// fmt.Println("Id:", id)
	// fmt.Println("Unread:", unread)

	// Open SQLite database file
	db, err := sql.Open("sqlite3", database)
	checkErr(err)
	defer db.Close()

	stmt, err := db.Prepare("UPDATE articles SET read = ? WHERE id = ?")
	checkErr(err)
	_, err = stmt.Exec(!unread, id)
	checkErr(err)
}

func ignore(database, id string) {
	// fmt.Println("Database:", database)
	// fmt.Println("Id:", id)

	// Open SQLite database file
	db, err := sql.Open("sqlite3", database)
	checkErr(err)
	defer db.Close()

	stmt, err := db.Prepare("UPDATE articles SET ignored = ? WHERE id = ?")
	checkErr(err)
	_, err = stmt.Exec(true, id)
	checkErr(err)
}

func archive(database, id string) {
	// fmt.Println("Database:", database)
	// fmt.Println("Id:", id)

	// Open SQLite database file
	db, err := sql.Open("sqlite3", database)
	checkErr(err)
	defer db.Close()

	// Check if this article already exists in the database
	var dummy string
	err = db.QueryRow("SELECT title FROM archived WHERE id = ?", id).Scan(&dummy)
	if err != sql.ErrNoRows {
		fmt.Println("duplicate item.")
		return
	}

	// Get article information from articles db
	var title, content, url string
	var created int
	err = db.QueryRow("SELECT title, content, url, created FROM articles WHERE id = ?", id).Scan(&title, &content, &url, &created)
	if err == sql.ErrNoRows {
		fmt.Println("invalid article id.")
		return
	}

	// Insert the article to archived db
	stmt, err := db.Prepare("INSERT INTO archived(id, title, content, url, created, rating) values(?,?,?,?,?,?)")
	checkErr(err)
	_, err = stmt.Exec(id, title, content, url, created, 0)
	checkErr(err)
}

func search(database, keyword string, containsContent bool) {
	// fmt.Println("Database:", database)
	// fmt.Println("Keyword:", keyword)
	// fmt.Println("ContainsContent:", containsContent)

	// Open SQLite database file
	db, err := sql.Open("sqlite3", database)
	checkErr(err)
	defer db.Close()

	whereCondition := fmt.Sprintf(" WHERE title LIKE '%%%s%%' ", keyword)
	if containsContent {
		whereCondition += fmt.Sprintf("OR content LIKE '%%%s%%' ", keyword)
	}
	statement := "SELECT id, title, created FROM articles" + whereCondition
	statement += "ORDER BY created DESC"
    rows, err := db.Query(statement)
	checkErr(err)

	defer rows.Close()
	for rows.Next() {
		var id, title string
		var created int
		rows.Scan(&id, &title, &created)
		checkErr(err)

		t := time.Unix(int64(created), int64(0))
		fmt.Printf("%s\t%s\t%s\n", id, t.Format(time.UnixDate), title)
	}
}

func main() {

	// Subcommands
	prepareCommand := flag.NewFlagSet("prepare", flag.ExitOnError)
	prepareDatabasePtr := prepareCommand.String("database", "", "SQLite file where articles will be stored")
	
	scrapeCommand := flag.NewFlagSet("scrape", flag.ExitOnError)
	scrapeSourcesPtr := scrapeCommand.String("sources", "", "JSON file defining news article sources")
	scrapeDatabasePtr := scrapeCommand.String("database", "", "SQLite file where articles will be stored")
	scrapeIntervalPtr := scrapeCommand.Int("interval", 600, "Interval between retrieval batch")
	scrapeDelayPtr := scrapeCommand.Int("delay", 1, "Delay between new article retrieval")
	scrapeLimitPtr := scrapeCommand.Int("limit", 10000, "Maximum articles to be saved")

	listCommand := flag.NewFlagSet("list", flag.ExitOnError)
	listDatabasePtr := listCommand.String("database", "", "SQLite file where articles are stored")
	listPagePtr := listCommand.Int("page", 0, "Page number when articles are sliced by page size unit")
	listPageSizePtr := listCommand.Int("pageSize", 10, "The number of articles to be displayed in a page")
	listUnreadOnlyPtr := listCommand.Bool("unreadOnly", false, "Exclude articles that are checked as read")

	showCommand := flag.NewFlagSet("show", flag.ExitOnError)
	showDatabasePtr := showCommand.String("database", "", "SQLite file where articles are stored")
	showIdPtr := showCommand.String("id", "", "Article ID")

	readCommand := flag.NewFlagSet("read", flag.ExitOnError)
	readDatabasePtr := readCommand.String("database", "", "SQLite file where articles are stored")
	readIdPtr := readCommand.String("id", "", "Article ID")
	readUnreadPtr := readCommand.Bool("unread", false, "Set as unread")

	ignoreCommand := flag.NewFlagSet("ignore", flag.ExitOnError)
	ignoreDatabasePtr := ignoreCommand.String("database", "", "SQLite file where articles are stored")
	ignoreIdPtr := ignoreCommand.String("id", "", "Article ID")

	archiveCommand := flag.NewFlagSet("archive", flag.ExitOnError)
	archiveDatabasePtr := archiveCommand.String("database", "", "SQLite file where articles are stored")
	archiveIdPtr := archiveCommand.String("id", "", "Article ID")

	searchCommand := flag.NewFlagSet("search", flag.ExitOnError)
	searchDatabasePtr := searchCommand.String("database", "", "SQLite file where articles are stored")
	searchKeywordPtr := searchCommand.String("keyword", "", "Keyword for search")
	searchContainsContentPtr := searchCommand.Bool("containsContent", false, "Contains the content of an article for search target")

	if len(os.Args) < 2 {
		fmt.Println("Subcommand required")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "prepare":
		prepareCommand.Parse(os.Args[2:])
		prepare(*prepareDatabasePtr)
	case "scrape":
		scrapeCommand.Parse(os.Args[2:])
		scrape(*scrapeSourcesPtr, *scrapeDatabasePtr, *scrapeIntervalPtr, *scrapeDelayPtr, *scrapeLimitPtr)
	case "list":
		listCommand.Parse(os.Args[2:])
		list(*listDatabasePtr, *listPagePtr, *listPageSizePtr, *listUnreadOnlyPtr)
	case "show":
		showCommand.Parse(os.Args[2:])
		show(*showDatabasePtr, *showIdPtr)
	case "read":
		readCommand.Parse(os.Args[2:])
		read(*readDatabasePtr, *readIdPtr, *readUnreadPtr)
	case "ignore":
		ignoreCommand.Parse(os.Args[2:])
		ignore(*ignoreDatabasePtr, *ignoreIdPtr)
	case "archive":
		archiveCommand.Parse(os.Args[2:])
		archive(*archiveDatabasePtr, *archiveIdPtr)
	case "search":
		searchCommand.Parse(os.Args[2:])
		search(*searchDatabasePtr, *searchKeywordPtr, *searchContainsContentPtr)
	default:
		fmt.Printf("Subcommand '%s' is not available.\n", os.Args[1])
		flag.PrintDefaults()
		os.Exit(1)
	}
}
