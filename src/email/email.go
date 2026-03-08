package email

import (
	"bytes"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net/smtp"
	"regexp"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
	"git.handmade.network/hmn/hmn/src/templates"
)

// TODO(asaf): Adjust this once we test on the server
const ExpectedEmailSendDuration = time.Millisecond * 1500

type RegistrationEmailData struct {
	Name                    string
	HomepageUrl             string
	CompleteRegistrationUrl string
}

func SendRegistrationEmail(
	toAddress string,
	toName string,
	username string,
	completionToken string,
	destination string,
	perf *perf.RequestPerf,
) error {
	defer perf.StartBlock("EMAIL", "Registration email").End()

	b1 := perf.StartBlock("EMAIL", "Rendering template")
	defer b1.End()
	contents, err := renderTemplate("email_registration.html", RegistrationEmailData{
		Name:                    toName,
		HomepageUrl:             hmnurl.BuildHomepage(),
		CompleteRegistrationUrl: hmnurl.BuildEmailConfirmation(username, completionToken, destination),
	})
	if err != nil {
		return err
	}
	b1.End()

	b2 := perf.StartBlock("EMAIL", "Sending email")
	defer b2.End()
	err = sendMail(toAddress, toName, "[Handmade Network] Registration confirmation", contents)
	if err != nil {
		return oops.New(err, "Failed to send email")
	}
	b2.End()

	return nil
}

type ExistingAccountEmailData struct {
	Name        string
	Username    string
	HomepageUrl string
	LoginUrl    string
}

func SendExistingAccountEmail(
	toAddress string,
	toName string,
	username string,
	destination string,
	perf *perf.RequestPerf,
) error {
	defer perf.StartBlock("EMAIL", "Existing account email").End()

	b1 := perf.StartBlock("EMAIL", "Rendering template")
	defer b1.End()
	contents, err := renderTemplate("email_account_existing.html", ExistingAccountEmailData{
		Name:        toName,
		Username:    username,
		HomepageUrl: hmnurl.BuildHomepage(),
		LoginUrl:    hmnurl.BuildLoginPage(destination, ""),
	})
	if err != nil {
		return err
	}
	b1.End()

	b2 := perf.StartBlock("EMAIL", "Sending email")
	defer b2.End()
	err = sendMail(toAddress, toName, "[Handmade Network] You already have an account!", contents)
	if err != nil {
		return oops.New(err, "Failed to send email")
	}
	b2.End()

	return nil
}

type PasswordResetEmailData struct {
	Name               string
	DoPasswordResetUrl string
	Expiration         time.Time
}

func SendPasswordReset(toAddress string, toName string, username string, resetToken string, expiration time.Time, perf *perf.RequestPerf) error {
	defer perf.StartBlock("EMAIL", "Password reset email").End()

	b1 := perf.StartBlock("EMAIL", "Rendering template")
	defer b1.End()
	contents, err := renderTemplate("email_password_reset.html", PasswordResetEmailData{
		Name:               toName,
		DoPasswordResetUrl: hmnurl.BuildDoPasswordReset(username, resetToken),
		Expiration:         expiration,
	})
	if err != nil {
		return err
	}
	b1.End()

	b2 := perf.StartBlock("EMAIL", "Sending email")
	defer b2.End()
	err = sendMail(toAddress, toName, "[Handmade Network] Your password reset request", contents)
	if err != nil {
		return oops.New(err, "Failed to send email")
	}
	b2.End()

	return nil
}

type TimeMachineEmailData struct {
	ProfileUrl      string
	Username        string
	UserEmail       string
	DiscordUsername string
	MediaUrls       []string
	DeviceInfo      string
	Description     string
}

func SendTimeMachineEmail(profileUrl, username, userEmail, discordUsername string, mediaUrls []string, deviceInfo, description string, perf *perf.RequestPerf) error {
	defer perf.StartBlock("EMAIL", "Time machine email").End()

	contents, err := renderTemplate("email_time_machine.html", TimeMachineEmailData{
		ProfileUrl:      profileUrl,
		Username:        username,
		UserEmail:       userEmail,
		DiscordUsername: discordUsername,
		MediaUrls:       mediaUrls,
		DeviceInfo:      deviceInfo,
		Description:     description,
	})
	if err != nil {
		return err
	}

	err = sendMail("team@handmade.network", "HMN Team", "[Time Machine] New submission", contents)
	if err != nil {
		return oops.New(err, "Failed to send email")
	}

	return nil
}

func SendTicketPurchaseEmail(toAddress string, toName string, ticket *models.Ticket) error {
	event, ok := hmndata.FindTicketEventBySlug(ticket.EventSlug)
	if !ok {
		return oops.New(nil, "failed to find event for ticket in email")
	}

	type TicketPurchaseEmailData struct {
		EventName string
		Name      string
		Email     string
		CodeURL   string
		TicketURL string
	}
	contents, err := renderTemplate("email_ticket_purchase.html", TicketPurchaseEmailData{
		EventName: event.Name,
		Name:      ticket.OwnerName,
		Email:     ticket.OwnerEmail,
		CodeURL:   hmnurl.BuildTicketQRCode(ticket.ID.String()),
		TicketURL: hmnurl.BuildTicketSingle(ticket.ID.String()),
	})
	if err != nil {
		return err
	}

	err = sendMail(toAddress, toName, fmt.Sprintf("Your ticket for the %s", event.Name), contents)
	if err != nil {
		return oops.New(err, "Failed to send email")
	}

	return nil
}

var EmailRegex = regexp.MustCompile(`^[^:\p{Cc} ]+@[^:\p{Cc} ]+\.[^:\p{Cc} ]+$`)

func IsEmail(address string) bool {
	return EmailRegex.Match([]byte(address))
}

func renderTemplate(name string, data any) ([]byte, error) {
	var buffer bytes.Buffer
	template, hasTemplate := templates.GetTemplate(name)
	if !hasTemplate {
		return nil, oops.New(nil, "template not found: %s", name)
	}
	err := template.Execute(&buffer, data)
	if err != nil {
		return nil, oops.New(err, "Failed to render template for email")
	}
	contentString := buffer.Bytes()
	contentString = bytes.ReplaceAll(contentString, []byte("\n"), []byte("\r\n"))
	return contentString, nil
}

func sendMail(toAddress, toName, subject string, contentHTML []byte) error {
	if config.Config.Email.ForceToAddress != "" {
		toAddress = config.Config.Email.ForceToAddress
	}
	processedHTML, err := preprocessEmailHTML(contentHTML)
	if err != nil {
		return err
	}
	contents := prepMailContents(
		makeHeaderAddress(toAddress, toName),
		makeHeaderAddress(config.Config.Email.FromAddress, config.Config.Email.FromName),
		subject,
		processedHTML,
	)
	return smtp.SendMail(
		fmt.Sprintf("%s:%d", config.Config.Email.ServerAddress, config.Config.Email.ServerPort),
		smtp.PlainAuth("", config.Config.Email.MailerUsername, config.Config.Email.MailerPassword, config.Config.Email.ServerAddress),
		config.Config.Email.FromAddress,
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

func prepMailContents(toLine string, fromLine string, subject string, contentHtml []byte) []byte {
	var builder strings.Builder

	fmt.Fprintf(&builder, "To: %s\r\n", toLine)
	fmt.Fprintf(&builder, "From: %s\r\n", fromLine)
	fmt.Fprintf(&builder, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	fmt.Fprintf(&builder, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	builder.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	builder.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	builder.WriteString("\r\n")
	writer := quotedprintable.NewWriter(&builder)
	writer.Write(contentHtml)
	writer.Close()
	builder.WriteString("\r\n")

	return []byte(builder.String())
}
