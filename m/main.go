package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/mail"
	"os"
	"text/tabwriter"

	"github.com/elos/cmd/internal/goog"

	gmail "google.golang.org/api/gmail/v1"
)

const help = `An elos command for mail.

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
		os.Exit(runGoog(ctx, os.Args[2:]))
	}
}

const (
	googHelp = `m goog

subcommands:
  - inbox
  - get
  - list`
)

func runGoog(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Println(googHelp)
		return 0
	}

	c, err := goog.Gmail(ctx)
	if err != nil {
		fmt.Printf("GmailClient error: %v\n", err)
		return 1
	}

	switch args[0] {
	case "get":
		if len(args) == 1 {
			fmt.Println("m goog get [subcommand]\n\nsubcommands:\n\t* message")
			return 0
		}

		switch args[1] {
		case "message", "messages":
			googGetMessage(ctx, c, args[2:])
			return 0
		case "thread", "threads":
			googGetThread(ctx, c, args[2:])
			return 0
		}
	case "search":
		if len(args) == 1 {
			fmt.Println("m goog search <query>")
			return 0
		}

		switch args[1] {
		case "message", "messages":
			googSearchMessage(ctx, c, args[2:])
			return 0
		case "thread", "threads":
			googSearchThread(ctx, c, args[2:])
			return 0
		}
	case "inbox":
		googInboxThreads(ctx, c, args[1:])
		return 0
	}

	return 0
}

func id(m *gmail.Message) string {
	return m.Id
}

func first(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func subject(m *gmail.Message) string {
	if m.Payload == nil {
		return ""
	}

	for _, h := range m.Payload.Headers {
		if h.Name == "Subject" {
			return h.Value
		}
	}

	return ""
}

func from(m *gmail.Message) string {
	if m.Payload == nil {
		return ""
	}

	for _, h := range m.Payload.Headers {
		if h.Name == "From" {
			e, err := mail.ParseAddress(h.Value)
			if err != nil {
				log.Fatal(err)
			}

			if e.Name == "" {
				return e.Address
			}
			return e.Name
		}
	}

	return ""
}

func threadRow(w io.Writer, t *gmail.Thread) {
	fmt.Fprintln(w, fmt.Sprintf("%s\t%s\t%s\t", t.Id, from(t.Messages[0]), first(subject(t.Messages[0]), 40)))
}

func messageRow(w io.Writer, m *gmail.Message) {
	fmt.Fprintln(w, fmt.Sprintf("%s\t%s\t%s\t", id(m), from(m), first(subject(m), 40)))
}

func googInboxThreads(ctx context.Context, c *gmail.Service, args []string) {
	ts := gmail.NewUsersThreadsService(c)
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 1, ' ', 0)
	fmt.Fprintln(w, "ID\tFROM\tSUBJECT\t")
	var results bool
	if err := ts.List("me").Q("label:inbox").Pages(ctx, func(l *gmail.ListThreadsResponse) error {
		for _, t := range l.Threads {
			results = true
			t, err := ts.Get("me", t.Id).Do()
			if err != nil {
				fmt.Errorf("Get error: %v", err)
				os.Exit(1)
			}
			threadRow(w, t)
		}
		return nil
	}); err != nil {
		fmt.Errorf("List error: %v", err)
		os.Exit(1)
	}
	if results {
		w.Flush()
	} else {
		fmt.Println("No mail.")
	}
	return
}

func googInboxMessages(ctx context.Context, c *gmail.Service, args []string) {
	ms := gmail.NewUsersMessagesService(c)
	if err := ms.List("me").Q("label:inbox").Pages(ctx, func(l *gmail.ListMessagesResponse) error {
		for _, m := range l.Messages {
			m, err := ms.Get("me", m.Id).Do()
			if err != nil {
				fmt.Errorf("Get error: %v", err)
				os.Exit(1)
			}

			fmt.Println(subject(m))
		}
		return nil
	}); err != nil {
		fmt.Errorf("List error: %v", err)
		os.Exit(1)
	}
	return
}

const googGetThreadHelp = `m goog get thread <id>`

func googGetThread(ctx context.Context, c *gmail.Service, args []string) {
	if len(args) == 0 {
		fmt.Println(googGetThreadHelp)
		return
	}

	ts := gmail.NewUsersThreadsService(c)
	t, err := ts.Get("me", args[0]).Do()
	if err != nil {
		fmt.Errorf("Get: %v", err)
	}

	fmt.Printf("%d messages\n", len(t.Messages))
	fmt.Println("---")
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 1, ' ', 0)
	fmt.Fprintln(w, "ID\tFROM\tSUBJECT\t")
	for _, m := range t.Messages {
		messageRow(w, m)
	}
	w.Flush()
}

const googGetMessageHelp = `m goog get message [id]`

func googGetMessage(ctx context.Context, c *gmail.Service, args []string) {
	ms := gmail.NewUsersMessagesService(c)
	switch len(args) {
	case 0:
		var num int

		if err := ms.List("me").Pages(ctx, func(l *gmail.ListMessagesResponse) error {
			for _, m := range l.Messages {
				num++
				if num > 100 {
					return nil
				}

				fmt.Println(m.Id)
			}
			return nil
		}); err != nil {
			fmt.Printf("UserMessagesService.List error: %v", err)
			os.Exit(1)
			return
		}

		fmt.Printf("%d messages", num)
	case 1:
		m, err := ms.Get("me", args[0]).Context(ctx).Do()
		if err != nil {
			fmt.Printf("UserMessagesService.Get error: %v", err)
			os.Exit(1)
			return
		}

		fmt.Println(m.Snippet)
		fmt.Println("Parts:")
		printPart(m.Payload, "  ")
	}
}

func googSearchMessage(ctx context.Context, c *gmail.Service, args []string) {
	ms := gmail.NewUsersMessagesService(c)

	if len(args) == 0 {
		fmt.Println("m goog search message <query>")
		return
	}

	if err := ms.List("me").Pages(ctx, func(l *gmail.ListMessagesResponse) error {
		for _, m := range l.Messages {
			fmt.Println(m.Id)
		}
		return nil
	}); err != nil {
		fmt.Printf("UserMessagesService.List error: %v", err)
		os.Exit(1)
		return
	}
}

func googSearchThread(ctx context.Context, c *gmail.Service, args []string) {
	ts := gmail.NewUsersThreadsService(c)

	if len(args) == 0 {
		fmt.Println("m goog search thread <query>")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 1, ' ', 0)
	fmt.Fprintln(w, "ID\tSUBJECT\t")

	var results bool
	if err := ts.List("me").Q(args[0]).Pages(ctx, func(l *gmail.ListThreadsResponse) error {
		for _, t := range l.Threads {
			results = true
			t, err := ts.Get("me", t.Id).Do()
			if err != nil {
				fmt.Errorf("Get error: %v", err)
				os.Exit(1)
			}
			threadRow(w, t)
		}
		return nil
	}); err != nil {
		fmt.Printf("List error: %v", err)
		os.Exit(1)
		return
	}

	if !results {
		fmt.Println("No results")
	}

	w.Flush()
	return
}

func printPart(p *gmail.MessagePart, prefix string) {
	//	fmt.Printf("%sBody: %s\n", prefix, p.Body.Data)
	fmt.Printf("%sMimeType: %s\n", prefix, p.MimeType)
	fmt.Printf("%sHeaders:\n", prefix)
	for _, h := range p.Headers {
		fmt.Printf("%s%s=%s,", prefix, h.Name, h.Value)
	}
	fmt.Println()
	decoded, err := base64.URLEncoding.DecodeString(p.Body.Data)
	if err != nil {
		fmt.Println("decode error:", err)
		return
	}
	fmt.Printf(`%sBody:
%s
`, prefix, decoded)
	for i, p := range p.Parts {
		fmt.Printf("%d.\n", i)
		printPart(p, prefix+"  ")
	}
}
