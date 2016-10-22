package main

import (
	"os"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"os/exec"
	"net/http"
	"golang.org/x/net/html"
)

func printErrorAndExit(module string, message string, stack []string) {
	fmt.Printf("%s : %s\n", module, message)
	fmt.Println("STACK =", stack)
	os.Exit(1)
}

func fillFormatString(stack []string, last int) (filled string, numConsumed int) {
	formatString := stack[last]
	last--

	numItems := strings.Count(formatString, "{}")
	if last < numItems - 1 {
		printErrorAndExit("fillFormatString()", "Invalid number of arguments", stack)
	}

	for i := 0; i < numItems; i++ {
		formatString = strings.Replace(formatString, "{}", stack[last], 1)
		last--
	}

	return formatString, numItems + 1
}

func print(stack []string) []string {
	last := len(stack)-1
	last--

	if last < 0 {
		printErrorAndExit("print()", "Invalid number of arguments", stack)
	}

	filled, numConsumed := fillFormatString(stack, last)
	last -= numConsumed

	fmt.Println(filled)
	return stack[:last+1]
}

func expr(stack []string) []string {
	last := len(stack)-1
	last--

	if last < 0 {
		printErrorAndExit("expr()", "Invalid number of arguments", stack)
	}

	filled, numConsumed := fillFormatString(stack, last)
	filled = strings.Replace(filled, ",", "", -1)
	last -= numConsumed

	cmd := fmt.Sprintf("echo \"%s\" | bc", filled)
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		printErrorAndExit("expr()", "Cannot execute an external command", stack)
	}

	last++
	stack[last] = strings.TrimSpace(string(out))
	return stack[:last+1]
}

// Selector Examples

// #hnmain > tbody > tr:nth-child(1) > td > table > tbody > tr > td:nth-child(3) > span > a
// body > div.container > div:nth-child(128) > pre
// #question > table > tbody > tr:nth-child(1) > td.postcell > div > div.post-text > p

func htmlAggregateText(node *html.Node, text string) string {
	aggregate := text
	if node.Type == html.TextNode {
		aggregate += node.Data
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		aggregate = htmlAggregateText(child, aggregate)
	}
	return aggregate
}

func htmlSelectElement(node *html.Node, selectorList []string, idx int) (string, bool) {
	if len(selectorList) == 0 {
		return htmlAggregateText(node.Parent, ""), true
	}

	id, class, tag, nth := "", "", "", ""
	item := selectorList[0]

	var re *regexp.Regexp
	var matched []string

	// id
	re = regexp.MustCompile("#(.+)")
	matched = re.FindStringSubmatch(item)
	if len(matched) > 0 {
		id = matched[1]
	}

	// class
	re = regexp.MustCompile("\\.([^\\.]+)")
	matched = re.FindStringSubmatch(item)
	if len(matched) > 0 {
		class = matched[1]
	}

	// tag
	re = regexp.MustCompile("^([^\\.#:]+)")
	matched = re.FindStringSubmatch(item)
	if len(matched) > 0 {
		tag = matched[1]
	}

	// nth
	re = regexp.MustCompile(":nth-child\\(([0-9]+)\\)")
	matched = re.FindStringSubmatch(item)
	if len(matched) > 0 {
		nth = matched[1]
	}

	idCondition := false
	if id == "" {
		idCondition = true
	} else {
		for _, attr := range node.Attr {
			if attr.Key == "id" && attr.Val == id {
				idCondition = true
				break
			}
		}
	}

	classCondition := false
	if class == "" {
		classCondition = true
	} else {
		for _, attr := range node.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, class) {
				classCondition = true
				break
			}
		}
	}

	tagCondition := false
	if tag == "" {
		tagCondition = true
	} else if node.Data == tag {
		tagCondition = true
	}

	nthCondition := false
	if nth == "" {
		nthCondition = true
	} else {
		i, _ := strconv.Atoi(nth)
		if i == idx {
			nthCondition = true
		}
	}

	selectorNext := selectorList
	if idCondition && classCondition && tagCondition && nthCondition {
		selectorNext = selectorList[1:]
	}

	idxChild := 1
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		text, found := htmlSelectElement(child, selectorNext, idxChild)
		if found {
			return text, found
		}

		if child.Type == html.ElementNode {
			idxChild++
		}
	}
	return "", false
}

func get(stack []string) []string {
	last := len(stack)-1
	last--

	if last < 1 {
		printErrorAndExit("get()", "Invalid number of arguments", stack)
	}

	selector := stack[last]
	last--
	url := stack[last]
	last--

	resp, err := http.Get(url)
	if err != nil {
		printErrorAndExit("get()", "Cannot retrieve a web page", stack)
	}

	root, err := html.Parse(resp.Body)
	if err != nil {
		printErrorAndExit("get()", "Cannot parse a web page", stack)
	}

	tokens := strings.Split(selector, ">")
 	for i := 0; i < len(tokens); i++ {
 		tokens[i] = strings.TrimSpace(tokens[i])
 	}
	text, _ := htmlSelectElement(root, tokens, 1)

	last++
	stack[last] = strings.TrimSpace(text)
	return stack[:last+1]
}

func main() {
	commands := make(map[string]func([]string)([]string))
	commands[":print:"] = print
	commands[":expr:"] = expr
	commands[":get:"] = get

	stack := []string{}
	for i := 1; i < len(os.Args); i++ {
		stack = append(stack, os.Args[i])

		function, ok := commands[stack[len(stack)-1]]
		if ok {
			stack = function(stack)
		}
	}
}
