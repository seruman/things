package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/johnmccabe/go-bitbar"
	"github.com/jomei/notionapi"
)

type TaskStatus string
type TaskPriority string

const (
	TaskStatusToDo  TaskStatus = "To Do"
	TaskStatusDoing TaskStatus = "Doing"

	TaskPriorityHigh TaskPriority = "High 🔥"
)

var (
	NotionAPIToken   string
	NotionDatabaseID string
)

func main() {
	ctx := context.Background()
	client := notionapi.NewClient(notionapi.Token(NotionAPIToken))

	query := &notionapi.DatabaseQueryRequest{}
	resp, err := client.Database.Query(ctx, notionapi.DatabaseID(NotionDatabaseID), query)
	if err != nil {
		panic(err)
	}
	// TODO: pagination

	app := bitbar.New()

	app.StatusLine("Tasks")
	menu := app.NewSubMenu()
	menu.HR()

	var tasks Tasks
	for _, p := range resp.Results {
		tasks = append(tasks, newTask(p))
	}

	groups := []struct {
		title string
		tasks Tasks
		color string
	}{
		{
			title: fmt.Sprintf("%v - %v", TaskStatusToDo, TaskPriorityHigh),
			tasks: tasks.filter(and(byStatus(TaskStatusToDo), byPriority(TaskPriorityHigh))),
			color: "red",
		},
		{
			title: fmt.Sprintf("%v", TaskStatusDoing),
			tasks: tasks.filter(byStatus(TaskStatusDoing)),
			color: "orange",
		},

		// {
		// 	title: "Rest",
		// 	tasks: tasks.filter(
		// 		and(
		// 			not(
		// 				or(
		// 					byStatus(TaskStatusDoing),
		// 					byStatus(TaskStatusToDo),
		// 					byPriority(TaskPriorityHigh),
		// 				),
		// 			),
		// 		),
		// 	),
		// 	color: "gray",
		// },
	}

	for _, group := range groups {
		if len(group.tasks) == 0 {
			continue
		}

		menu.Line(group.title).Color(group.color).Refresh()
		for _, task := range group.tasks {
			menu.Line(task.title).Href(task.uri)
		}
		menu.HR()
	}

	app.Render()
}

type filterFunc func(t Task) bool

func byStatus(status TaskStatus) filterFunc {
	return func(t Task) bool {
		return t.status == status
	}
}

func byPriority(priority TaskPriority) filterFunc {
	return func(t Task) bool {
		return t.priority == priority
	}
}

func not(fn filterFunc) filterFunc {
	return func(t Task) bool {
		return !fn(t)
	}
}

func and(fns ...filterFunc) filterFunc {
	return func(t Task) bool {
		for _, fn := range fns {
			if !fn(t) {
				return false
			}
		}
		return true
	}
}

func or(fns ...filterFunc) filterFunc {
	return func(t Task) bool {
		var result bool
		for _, fn := range fns {
			result = result || fn(t)
		}
		return result
	}
}

type Tasks []Task

func (ts Tasks) filter(fn filterFunc) Tasks {
	var filtered Tasks
	for _, t := range ts {
		if fn(t) {
			filtered = append(filtered, t)
		}
	}

	return filtered
}

type Task struct {
	title    string
	priority TaskPriority
	status   TaskStatus
	uri      string
	url      string
}

func newTask(page notionapi.Page) Task {
	var title, priority, status string
	for property, value := range page.Properties {
		switch property {
		case "Name":
			t := value.(*notionapi.TitleProperty)
			title = "<Untitled>"
			if len(t.Title) > 0 {
				title = t.Title[0].Text.Content
			}
		case "Priority":
			s := value.(*notionapi.SelectProperty)
			priority = s.Select.Name
		case "Status":
			s := value.(*notionapi.SelectProperty)
			status = s.Select.Name
		}
	}

	if priority == "" {
		priority = "<?Priority>"
	}

	if status == "" {
		status = "<?Status>"
	}

	url := page.URL
	uri := strings.Replace(url, "https://", "notion://", 1)
	return Task{
		title:    title,
		priority: TaskPriority(priority),
		status:   TaskStatus(status),
		uri:      uri,
		url:      url,
	}
}
