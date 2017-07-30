package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	calendar "google.golang.org/api/calendar/v3"

	"github.com/elos/cmd/internal/goog"
)

const help = `An elos command for calendars.

Subcommands:
  - goog`

func main() {
	ctx := context.Background()

	if len(os.Args) == 1 {
		fmt.Println(help)
		return
	}

	switch os.Args[1] {
	case "goog":
		runGoog(ctx, os.Args[2:])
	default:
		fmt.Println(help)
	}
}

const googHelp = `c goog

Subcommands:
  - get`

func runGoog(ctx context.Context, args []string) {
	if len(args) == 0 {
		fmt.Println(googHelp)
		return
	}

	s, err := goog.Calendar(ctx)
	if err != nil {
		fmt.Printf("goog.Calendar error: %v", err)
		os.Exit(1)
	}

	switch args[0] {
	case "get":
		runGoogGet(ctx, s, args[1:])
	case "search":
		runGoogSearch(ctx, s, args[1:])
	default:
		fmt.Println(googHelp)
	}
}

func runGoogGet(ctx context.Context, s *calendar.Service, args []string) {
	if len(args) == 0 {
		fmt.Println("c goog get [resource]\nResources:\n  - calendars\n  - events")
		return
	}

	switch args[0] {
	case "calendar", "calendars", "cal", "cals":
		if len(args) > 1 {
			c, err := s.Calendars.Get(args[1]).Context(ctx).Do()
			if err != nil {
				fmt.Printf("Get error: %v", err)
				os.Exit(1)
			}
			fmt.Printf(`---
ID:          %s
SUMMARY:     %s
DESCRIPTION: %s
LOCATION:    %s
TIMEZONE:    %s
---
`, c.Id, c.Summary, c.Description, c.Location, c.TimeZone)
		} else {
			cl, err := s.CalendarList.List().Context(ctx).Do()
			if err != nil {
				fmt.Printf("List error: %v", err)
				os.Exit(1)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 2, 1, ' ', 0)
			fmt.Fprintln(w, "ID\tSUMMARY\tTIMEZONE\t")
			for _, entry := range cl.Items {
				fmt.Fprintf(w, "%s\t%s\t%s\t\n", entry.Id, entry.Summary, entry.TimeZone)
			}
			w.Flush()
		}
	case "event", "events":
		if len(args) > 1 {
			e, err := s.Events.Get("primary", args[1]).Context(ctx).Do()
			if err != nil {
				fmt.Printf("Get error: %v", err)
				os.Exit(1)
			}
			fmt.Printf(`---
ID:          %s
ICALUID:     %s
SUMMARY:     %s
DESCRIPTION: %s
START:       %s
END:         %s
VISIBILITY:  %s
PRIV COPY:   %t
CREATOR:     %s
LINK:        %s
---
`, e.Id, e.ICalUID, e.Summary, e.Description, e.Start.DateTime, e.End.DateTime,
				e.Visibility, e.PrivateCopy, e.Creator.DisplayName, e.HtmlLink)
		} else {
			w := tabwriter.NewWriter(os.Stdout, 0, 2, 1, ' ', 0)
			fmt.Fprintln(w, "ID\tSUMMARY\t")
			if err := s.Events.List("primary").Pages(ctx, func(es *calendar.Events) error {
				for _, e := range es.Items {
					fmt.Fprintf(w, "%s\t%s\t\n", first(e.Id, 100), first(e.Summary, 20))
				}
				return nil
			}); err != nil {
				fmt.Printf("List error: %v", err)
				os.Exit(1)
			}
			w.Flush()
		}
	}

	return
}

func runGoogSearch(ctx context.Context, s *calendar.Service, args []string) {
	if len(args) == 0 {
		fmt.Println("c goog get [resource]\nResources:\n  - calendars\n  - events")
		return
	}

	switch args[0] {
	case "event", "events":
		if len(args) == 1 {
			fmt.Println("c goog search events [query]")
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 1, ' ', 0)
		fmt.Fprintln(w, "ID\tSUMMARY\t")
		if err := s.Events.List("primary").Q(args[1]).Pages(ctx, func(es *calendar.Events) error {
			for _, e := range es.Items {
				fmt.Fprintf(w, "%s\t%s\t\n", first(e.Id, 100), first(e.Summary, 20))
			}
			return nil
		}); err != nil {
			fmt.Printf("List error: %v", err)
			os.Exit(1)
		}
		w.Flush()
	}
	return
}

func first(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
