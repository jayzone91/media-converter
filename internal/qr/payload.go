package qr

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type VCard struct {
	FirstName string
	LastName  string

	Company  string
	Position string

	PhoneWork  string
	PhoneHome  string
	MobileWork string
	MobileHome string
	FaxWork    string
	Email      string
	Website    string
	Street     string
	PostalCode string
	City       string
	Region     string
	Country    string
}

type WiFi struct {
	SSID       string
	Password   string
	Encryption string
	Hidden     bool
}

type Event struct {
	Title       string
	Start       time.Time
	End         time.Time
	Location    string
	Description string
}

func URL(value string) (string, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return "", fmt.Errorf("URL darf nicht leer sein")
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return "", fmt.Errorf("ungültige URL: %w", err)
	}

	if parsed.Scheme != "http" &&
		parsed.Scheme != "https" {
		return "", fmt.Errorf(
			"URL muss http oder https verwenden",
		)
	}

	return value, nil
}

func Text(value string) (string, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return "", fmt.Errorf(
			"Text darf nicht leer sein",
		)
	}

	return value, nil
}

func Phone(value string) (string, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return "", fmt.Errorf(
			"Telefonnummer darf nicht leer sein",
		)
	}

	return "tel:" + value, nil
}

func WiFiPayload(data WiFi) (string, error) {
	ssid := strings.TrimSpace(data.SSID)

	if ssid == "" {
		return "", fmt.Errorf(
			"SSID darf nicht leer sein",
		)
	}

	encryption := strings.ToUpper(
		strings.TrimSpace(
			data.Encryption,
		),
	)

	switch encryption {
	case "":
		encryption = "WPA"

	case "WPA", "WEP":
		// gültig

	case "NOPASS":
		encryption = "nopass"

	default:
		return "", fmt.Errorf(
			"nicht unterstützte WLAN-Verschlüsselung: %s",
			data.Encryption,
		)
	}

	var builder strings.Builder

	builder.WriteString("WIFI:T:")
	builder.WriteString(encryption)

	builder.WriteString(";S:")
	builder.WriteString(
		escapeWiFi(ssid),
	)

	if encryption != "nopass" {
		builder.WriteString(";P:")
		builder.WriteString(
			escapeWiFi(data.Password),
		)
	}

	if data.Hidden {
		builder.WriteString(";H:true")
	}

	builder.WriteString(";;")

	return builder.String(), nil
}

func VCardPayload(
	data VCard,
) (string, error) {
	firstName := strings.TrimSpace(
		data.FirstName,
	)

	lastName := strings.TrimSpace(
		data.LastName,
	)

	if firstName == "" &&
		lastName == "" {
		return "", fmt.Errorf(
			"Vorname oder Nachname muss angegeben werden",
		)
	}

	fullName := strings.TrimSpace(
		firstName + " " + lastName,
	)

	lines := []string{
		"BEGIN:VCARD",
		"VERSION:3.0",

		"N:" +
			escapeVCard(lastName) + ";" +
			escapeVCard(firstName) +
			";;;",

		"FN:" +
			escapeVCard(fullName),
	}

	appendVCardField(
		&lines,
		"ORG:",
		data.Company,
	)

	appendVCardField(
		&lines,
		"TITLE:",
		data.Position,
	)

	appendVCardField(
		&lines,
		"TEL;TYPE=WORK,VOICE:",
		data.PhoneWork,
	)

	appendVCardField(
		&lines,
		"TEL;TYPE=HOME,VOICE:",
		data.PhoneHome,
	)

	appendVCardField(
		&lines,
		"TEL;TYPE=WORK,CELL:",
		data.MobileWork,
	)

	appendVCardField(
		&lines,
		"TEL;TYPE=HOME,CELL:",
		data.MobileHome,
	)

	appendVCardField(
		&lines,
		"TEL;TYPE=WORK,FAX:",
		data.FaxWork,
	)

	appendVCardField(
		&lines,
		"EMAIL;TYPE=INTERNET:",
		data.Email,
	)

	appendVCardField(
		&lines,
		"URL:",
		data.Website,
	)

	if hasAddress(data) {
		lines = append(
			lines,
			"ADR;TYPE=WORK:;;"+
				escapeVCard(data.Street)+";"+
				escapeVCard(data.City)+";"+
				escapeVCard(data.Region)+";"+
				escapeVCard(data.PostalCode)+";"+
				escapeVCard(data.Country),
		)
	}

	lines = append(
		lines,
		"END:VCARD",
	)

	return strings.Join(
		lines,
		"\r\n",
	), nil
}

func EventPayload(
	data Event,
) (string, error) {
	title := strings.TrimSpace(
		data.Title,
	)

	if title == "" {
		return "", fmt.Errorf(
			"Event-Titel darf nicht leer sein",
		)
	}

	if data.Start.IsZero() {
		return "", fmt.Errorf(
			"Event-Beginn fehlt",
		)
	}

	if data.End.IsZero() {
		return "", fmt.Errorf(
			"Event-Ende fehlt",
		)
	}

	if !data.End.After(data.Start) {
		return "", fmt.Errorf(
			"Event-Ende muss nach dem Beginn liegen",
		)
	}

	lines := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//Media Tools//QR Event//DE",
		"BEGIN:VEVENT",

		"DTSTART:" +
			formatICalTime(data.Start),

		"DTEND:" +
			formatICalTime(data.End),

		"SUMMARY:" +
			escapeICal(title),
	}

	if location := strings.TrimSpace(
		data.Location,
	); location != "" {
		lines = append(
			lines,
			"LOCATION:"+
				escapeICal(location),
		)
	}

	if description := strings.TrimSpace(
		data.Description,
	); description != "" {
		lines = append(
			lines,
			"DESCRIPTION:"+
				escapeICal(description),
		)
	}

	lines = append(
		lines,
		"END:VEVENT",
		"END:VCALENDAR",
	)

	return strings.Join(
		lines,
		"\r\n",
	), nil
}

func appendVCardField(
	lines *[]string,
	prefix string,
	value string,
) {
	value = strings.TrimSpace(value)

	if value == "" {
		return
	}

	*lines = append(
		*lines,
		prefix+escapeVCard(value),
	)
}

func hasAddress(
	data VCard,
) bool {
	return strings.TrimSpace(data.Street) != "" ||
		strings.TrimSpace(data.PostalCode) != "" ||
		strings.TrimSpace(data.City) != "" ||
		strings.TrimSpace(data.Region) != "" ||
		strings.TrimSpace(data.Country) != ""
}

func escapeVCard(
	value string,
) string {
	replacer := strings.NewReplacer(
		`\`,
		`\\`,
		";",
		`\;`,
		",",
		`\,`,
		"\r\n",
		`\n`,
		"\n",
		`\n`,
		"\r",
		`\n`,
	)

	return replacer.Replace(
		strings.TrimSpace(value),
	)
}

func escapeWiFi(
	value string,
) string {
	replacer := strings.NewReplacer(
		`\`,
		`\\`,
		";",
		`\;`,
		",",
		`\,`,
		":",
		`\:`,
		`"`,
		`\"`,
	)

	return replacer.Replace(value)
}

func escapeICal(
	value string,
) string {
	replacer := strings.NewReplacer(
		`\`,
		`\\`,
		";",
		`\;`,
		",",
		`\,`,
		"\r\n",
		`\n`,
		"\n",
		`\n`,
		"\r",
		`\n`,
	)

	return replacer.Replace(
		strings.TrimSpace(value),
	)
}

func formatICalTime(
	value time.Time,
) string {
	if value.Location() == time.UTC {
		return value.Format(
			"20060102T150405Z",
		)
	}

	return value.Format(
		"20060102T150405",
	)
}
