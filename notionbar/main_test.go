package main

import (
	"testing"

	cmpkg "github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestTasks_filter(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name     string
		tasks    Tasks
		filter   filterFunc
		expected Tasks
	}{
		{
			name: "filter by status",
			tasks: Tasks{
				{title: "1", status: "To-Do", priority: "High"},
				{title: "2", status: "To-Do", priority: "Low"},
				{title: "3", status: "Doing", priority: "Medium"},
			},
			filter: byStatus("To-Do"),
			expected: Tasks{
				{title: "1", status: "To-Do", priority: "High"},
				{title: "2", status: "To-Do", priority: "Low"},
			},
		},
		{
			name: "filter by priority",
			tasks: Tasks{
				{title: "1", status: "To-Do", priority: "High"},
				{title: "2", status: "To-Do", priority: "Low"},
				{title: "3", status: "Doing", priority: "High"},
			},
			filter: byPriority("High"),
			expected: Tasks{
				{title: "1", status: "To-Do", priority: "High"},
				{title: "3", status: "Doing", priority: "High"},
			},
		},
		{
			name: "filter by status and priority",
			tasks: Tasks{
				{title: "1", status: "To-Do", priority: "High"},
				{title: "2", status: "To-Do", priority: "Low"},
				{title: "3", status: "Doing", priority: "High"},
			},
			filter: and(byStatus("To-Do"), byPriority("High")),
			expected: Tasks{
				{title: "1", status: "To-Do", priority: "High"},
			},
		},
		{
			name: "filter by status and priority",
			tasks: Tasks{
				{title: "1", status: "To-Do", priority: "High"},
				{title: "2", status: "To-Do", priority: "Low"},
				{title: "3", status: "Doing", priority: "High"},
				{title: "4", status: "To-Do", priority: "Medium"},
			},
			filter: and(byStatus("To-Do"), not(or(byPriority("High"), byPriority("Low")))),
			expected: Tasks{
				{title: "4", status: "To-Do", priority: "Medium"},
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.tasks.filter(tc.filter)
			assert.Assert(t, cmp.DeepEqual(tc.expected, got, cmpkg.AllowUnexported(Task{})))

		})
	}
}
