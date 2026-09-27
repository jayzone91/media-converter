package web

import (
	"fmt"
	"strings"
	"time"

	qrservice "github.com/jayzone91/media-converter/internal/qr"
)

func buildQRPayload(
	request qrGenerateRequest,
) (string, error) {
	switch strings.ToLower(
		strings.TrimSpace(
			request.Type,
		),
	) {
	case "url":
		return qrservice.URL(
			request.URL,
		)

	case "text":
		return qrservice.Text(
			request.Text,
		)

	case "phone":
		return qrservice.Phone(
			request.Phone,
		)

	case "wifi":
		return buildQRWiFiPayload(
			request.WiFi,
		)

	case "vcard":
		return buildQRVCardPayload(
			request.VCard,
		)

	case "event":
		return buildQREventPayload(
			request.Event,
		)

	default:
		return "", fmt.Errorf(
			"Unbekannter QR-Code-Typ",
		)
	}
}

func buildQRWiFiPayload(
	request qrWiFiRequest,
) (string, error) {
	return qrservice.WiFiPayload(
		qrservice.WiFi{
			SSID:       request.SSID,
			Password:   request.Password,
			Encryption: request.Encryption,
			Hidden:     request.Hidden,
		},
	)
}

func buildQRVCardPayload(
	request qrVCardRequest,
) (string, error) {
	return qrservice.VCardPayload(
		qrservice.VCard{
			FirstName: request.FirstName,
			LastName:  request.LastName,

			Company:  request.Company,
			Position: request.Position,

			PhoneWork:  request.PhoneWork,
			PhoneHome:  request.PhoneHome,
			MobileWork: request.MobileWork,
			MobileHome: request.MobileHome,
			FaxWork:    request.FaxWork,

			Email:   request.Email,
			Website: request.Website,

			Street:     request.Street,
			PostalCode: request.PostalCode,
			City:       request.City,
			Region:     request.Region,
			Country:    request.Country,
		},
	)
}

func buildQREventPayload(
	request qrEventRequest,
) (string, error) {
	start, err := parseQRDateTime(
		request.Start,
	)
	if err != nil {
		return "", fmt.Errorf(
			"Ungültiger Beginn",
		)
	}

	end, err := parseQRDateTime(
		request.End,
	)
	if err != nil {
		return "", fmt.Errorf(
			"Ungültiges Ende",
		)
	}

	return qrservice.EventPayload(
		qrservice.Event{
			Title:       request.Title,
			Start:       start,
			End:         end,
			Location:    request.Location,
			Description: request.Description,
		},
	)
}

func parseQRDateTime(
	value string,
) (time.Time, error) {
	value = strings.TrimSpace(
		value,
	)

	if value == "" {
		return time.Time{}, fmt.Errorf(
			"date is empty",
		)
	}

	return time.ParseInLocation(
		"2006-01-02T15:04",
		value,
		time.Local,
	)
}
