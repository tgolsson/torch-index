package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gocolly/colly"
)

var (
	URLs = []string{
		"https://download.pytorch.org/whl/",
		"https://download.pytorch.org/whl/nightly/",
		"https://download.pytorch.org/whl/cu118/",
		"https://download.pytorch.org/whl/cu116/",
		"https://download.pytorch.org/whl/cu110/",
		"https://download.pytorch.org/whl/cu111/",
		"https://download.pytorch.org/whl/cpu/",
		"https://download.pytorch.org/whl/nightly/cpu/",
	}
)

type Version struct {
	Title string
	URL   string
}

type Project struct {
	Name     string
	Versions []Version
}

type Index struct {
	URL      string
	Projects []Project
}

func collectProject(name, url string) Project {
	fmt.Println("collecting project", url)
	c := colly.NewCollector()

	versions := make([]Version, 0)
	project := Project{
		Name: name,
	}

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		version := Version{
			Title: e.Text,
			URL:   e.Attr("href"),
		}
		fmt.Println("got version", version)
		versions = append(versions, version)
	})

	c.Visit(url)

	project.Versions = versions

	fmt.Println("collected project", url)
	return project
}

func collectIndex(url string) Index {
	fmt.Println("collecting index", url)
	c := colly.NewCollector()
	wg := sync.WaitGroup{}

	projects := make([]Project, 0)
	ch := make(chan Project, 10000)
	index := Index{
		URL: url,
	}

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		wg.Add(1)

		go func(e *colly.HTMLElement) {
			defer wg.Done()

			project := collectProject(e.Text, fmt.Sprintf("%s%s", url, e.Attr("href")))
			ch <- project
		}(e)
	})

	c.Visit(url)

	wg.Wait()

	fmt.Println("collected all projects", url)
	close(ch)
	for project := range ch {
		projects = append(projects, project)
	}

	index.Projects = projects
	fmt.Println("collected index", url)

	return index
}

func writeProject(variant string, project Project) {
	fmt.Println("writing project", project.Name)

	os.MkdirAll(fmt.Sprintf("%s/%s", variant, project.Name), 0755)
	out, err := os.Create(fmt.Sprintf("%s/%s/index.html", variant, project.Name))
	if err != nil {
		panic(err)
	}

	out.WriteString("<html><body>")
	for _, version := range project.Versions {
		out.WriteString(fmt.Sprintf("<a href=\"https://download.pytorch.org%s\">%s</a><br>", version.URL, version.Title))
	}
	out.WriteString("</body></html>")
	fmt.Println("wrote project", project.Name)
}

func writeIndex(index Index) {
	fmt.Println("writing index", index.URL)

	variant := strings.Replace(index.URL, "https://download.pytorch.org/", "", 1)
	if strings.HasSuffix(variant, "/") {
		variant = variant[:len(variant)-1]
	}

	os.MkdirAll(variant, 0755)
	out, err := os.Create(fmt.Sprintf("%s/index.html", variant))
	if err != nil {
		panic(err)
	}

	out.WriteString("<html><body>")
	for _, project := range index.Projects {
		out.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a><br>", project.Name, project.Name))
	}
	out.WriteString("</body></html>")

	fmt.Println("wrote index", index.URL)

	for _, project := range index.Projects {
		writeProject(variant, project)
	}
}

func main() {
	wg := sync.WaitGroup{}
	wg.Add(len(URLs))

	for idx, url := range URLs {
		go func(url string, idx int) {
			defer wg.Done()

			index := collectIndex(url)
			writeIndex(index)
		}(url, idx)
	}

	wg.Wait()
}
