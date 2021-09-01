package main

import (
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"net/smtp"
	"os"
	"strings"
	"time"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Printf(`The source code for this program can be found at git.handmade.network/hmn/hmn/adminmailer
The config data is compiled into the program. If you need to change the config,
find the adminmailer package in the hmn repo, modify config.go, and recompile.`)
		fmt.Printf("\n\nUsage:\n")
		fmt.Printf("You must provide a subject and message to send.\nMessage must be provided in stdin.\n\n")
		fmt.Printf("%s [subject]\n\n", os.Args[0])
		os.Exit(1)
	}

	subject := os.Args[1]

	message, err := io.ReadAll(os.Stdin)
	if err != nil && err != io.EOF {
		fmt.Printf("Error reading input: %v\n\n", err)
		os.Exit(1)
	}

	err = sendMail(RecvAddress, RecvName, subject, string(message))
	if err != nil {
		fmt.Printf("Failed to send email:\n%v\n\n", err)
		os.Exit(1)
	}
}

func sendMail(toAddress, toName, subject, contentHtml string) error {
	contents := prepMailContents(
		makeHeaderAddress(toAddress, toName),
		makeHeaderAddress(FromAddress, FromName),
		subject,
		contentHtml,
	)
	return smtp.SendMail(
		fmt.Sprintf("%s:%d", ServerAddress, ServerPort),
		smtp.PlainAuth("", FromAddress, FromAddressPassword, ServerAddress),
		FromAddress,
		[]string{toAddress},
		contents,
	)
}

func makeHeaderAddress(email, fullname string) string {
	if fullname != "" {
		encoded := mime.BEncoding.Encode("utf-8", fullname)
		if encoded == fullname {
			encoded = strings.ReplaceAll(encoded, `"`, `\"`)
			encoded = fmt.Sprintf("\"%s\"", encoded)
		}
		return fmt.Sprintf("%s <%s>", encoded, email)
	} else {
		return email
	}
}

func prepMailContents(toLine string, fromLine string, subject string, contentHtml string) []byte {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("To: %s\r\n", toLine))
	builder.WriteString(fmt.Sprintf("From: %s\r\n", fromLine))
	builder.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z)))
	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject)))
	builder.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	builder.WriteString("\r\n")
	writer := quotedprintable.NewWriter(&builder)
	writer.Write([]byte(contentHtml))
	writer.Close()
	builder.WriteString("\r\n")

	return []byte(builder.String())
}
